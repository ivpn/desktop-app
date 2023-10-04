#!/bin/bash

LOGGING=0

if [[ "${LOGGING}" == "1" ]] ; then
    exec 1>> "/Library/Logs/ivpn-dns.sh.logs" 2>&1

    DATE=`date`
    echo "${DATE}: $0 $@"
fi

PATH=/usr/sbin:$PATH

PRI_IFACE=`echo 'show State:/Network/Global/IPv4' | scutil | grep PrimaryInterface | sed -e 's/.*PrimaryInterface : //'`
PSID=`echo 'show State:/Network/Global/IPv4' | scutil | grep PrimaryService | sed -e 's/.*PrimaryService : //'`

# There is 2 possible sources to copy DNS configuration:
# "State:/Network/IVPN/DNSBase" - normal usage
# "State:/Network/IVPN/DNSAlternate" - when alternate DNS defined (DNSBase is ignored in this case)
# To update value of 'IVPN_DNS_SOURCE_PATH' variable - call function 'update_IVPN_DNS_SOURCE_PATH'
IVPN_DNS_SOURCE_PATH="State:/Network/IVPN/DNSBase"

# Primary interface can be not detected due to switching WiFi network at current moment.
# Here we are trying to get primary interface during 5 seconds (giving a chance to connect/disconnect WiFi)
#
# We are 'waiting' for primary interface only for '-update' calls. All other commands are executing without delays.
#if [ "$1" = "-update" ] || [ "$1" = "-down" ] ; then
if [ "$1" = "-update" ] ; then
  for run in {1..10}
  do
    if [[ "${PRI_IFACE}" != "" ]]; then
        break;
    fi
    echo "($1) Primary interface not detected. Retry in 1 second..."
    sleep 1
    PRI_IFACE=`echo 'show State:/Network/Global/IPv4' | scutil | grep PrimaryInterface | sed -e 's/.*PrimaryInterface : //'`
    PSID=`echo 'show State:/Network/Global/IPv4' | scutil | grep PrimaryService | sed -e 's/.*PrimaryService : //'`
  done
fi

if [[ "${PSID}" == "" ]]; then
    echo "Warning: Primary interface not found"
else
    echo "Primary interface: '${PRI_IFACE}' PSID: '${PSID}'"
fi

function isPrimaryInterfaceDetected {
    if [[ "${PSID}" == "" ]]; then
        return 1
    fi
    return 0
}

function ensurePrimaryInterfaceDetected {
    if ! isPrimaryInterfaceDetected; then
        echo "Error: Primary interface not found"
        exit 1
    fi
}

function print_state {
    echo "*** DNS STATUS ***"
    echo "-------------------------------------"
    echo "***  scutil --dns ***"
    scutil --dns | grep -E 'resolver|nameserver'
    echo "-------------------------------------"
    echo "Primary interface: '${PRI_IFACE}' PSID: '${PSID}'"
    echo "-------------------------------------"
    echo "***  State:/Network/Global/IPv4 ***"
    echo "show State:/Network/Global/IPv4" | scutil
    echo "-------------------------------------"
    echo "***  State:/Network/IVPN/Original/DNS/State ***"
    echo "show State:/Network/IVPN/Original/DNS/State" | scutil
    echo "-------------------------------------"
    echo "***  State:/Network/IVPN/Original/DNS/Setup ***"
    echo "show State:/Network/IVPN/Original/DNS/Setup" | scutil
    echo "-------------------------------------"

    if isPrimaryInterfaceDetected; then
        echo "***  State:/Network/Service/${PSID}/DNS ***"
        echo "show State:/Network/Service/${PSID}/DNS" | scutil
        echo "-------------------------------------"
        echo "***  Setup:/Network/Service/${PSID}/DNS ***"
        echo "show Setup:/Network/Service/${PSID}/DNS" | scutil
    else
        echo "!!! Primary interface not detected !!!"
    fi

    echo "-------------------------------------"
    echo "***  State:/Network/IVPN/DNSAlternate ***"
    echo "show State:/Network/IVPN/DNSAlternate" | scutil
    echo "-------------------------------------"
    echo "***  State:/Network/IVPN/DNSBase ***"
    echo "show State:/Network/IVPN/DNSBase" | scutil
    echo "-------------------------------------"
    update_IVPN_DNS_SOURCE_PATH
    echo "***  IVPN DNS path: ${IVPN_DNS_SOURCE_PATH}"
    echo "-------------------------------------"
}

function is_dns_changed {

    PREFIX=$1
    update_IVPN_DNS_SOURCE_PATH;

    ensurePrimaryInterfaceDetected

    DNS_STATE=`echo "show ${PREFIX}:/Network/Service/${PSID}/DNS" | scutil`
    VPN_STATE=`echo "show ${IVPN_DNS_SOURCE_PATH}" | scutil`

    if [[ "${DNS_STATE}" == "${VPN_STATE}" ]]; then
        return 1
    fi

    return 0
}

function is_vpn_dns_set {
    echo "show State:/Network/IVPN/Original/DNS/State" | scutil | grep ServerAddresses >/dev/null
}

function is_dns_set_by_ivpn {
    PREFIX=$1

    ensurePrimaryInterfaceDetected
    echo "show ${PREFIX}:/Network/Service/${PSID}/DNS" | scutil | grep SetByIVPN >/dev/null
    return $?
}

function store_user_setting {
    PREFIX=$1

    if is_dns_set_by_ivpn "${PREFIX}"; then
        #echo "Ignoring: Current DNS change was made by IVPN"
        return 1
    fi

    ensurePrimaryInterfaceDetected

    # Save original DNS configuration for service (primary interface)
    echo "Storing ${PREFIX}:/ dns settings"
    scutil <<_EOF
    d.init
    get ${PREFIX}:/Network/Service/${PSID}/DNS
    set State:/Network/IVPN/Original/DNS/${PREFIX}
_EOF

    # Save information about the primary interface in which DNS configuration was backed up
    # In some situations, it is not possible to detect the primary interface.
    # E.g.: At the moment when switching WiFi networks, we use this info to recover original DNS servings (see "-down" command)
    scutil <<_EOF
    d.init
    d.add PrimaryInterface "${PRI_IFACE}"
    d.add PrimaryService "${PSID}"    
    set State:/Network/IVPN/PrimaryInterfaceInfo
_EOF
}

function restore_user_setting {
    PREFIX=$1

    ensurePrimaryInterfaceDetected

    echo "Restoring ${PREFIX}:/ dns settings"

    scutil <<_EOF
    d.init
    get State:/Network/IVPN/Original/DNS/${PREFIX}
    set ${PREFIX}:/Network/Service/${PSID}/DNS
_EOF
}

function update_setting {
    PREFIX=$1
    update_IVPN_DNS_SOURCE_PATH;

    ensurePrimaryInterfaceDetected

    scutil <<_EOF
    d.init
    get ${IVPN_DNS_SOURCE_PATH}
    set ${PREFIX}:/Network/Service/${PSID}/DNS
_EOF
}

function store_and_update {
    PREFIX=$1

    if [[ "${PREFIX}" == "" ]]; then
        return 127
    fi

    store_user_setting "${PREFIX}"
    update_setting "${PREFIX}"
}

################### IVPN DNS PARAMETERS DEFINITION #############################
function update_IVPN_DNS_SOURCE_PATH {
    if is_alternate_ivpn_dns_defined; then
      IVPN_DNS_SOURCE_PATH="State:/Network/IVPN/DNSAlternate"
    else
      IVPN_DNS_SOURCE_PATH="State:/Network/IVPN/DNSBase"
    fi
}

# Check if current IVPN DNS is alternative
function is_alternate_ivpn_dns_defined {
    echo "show State:/Network/IVPN/DNSAlternate" | scutil | grep SetByIVPN >/dev/null
    return $?
}

function define_alternate_ivpn_dns {
  DOMAIN_NAME=$1
  VPN_DNS=$2

  #echo "DOMAIN: $DOMAIN_NAME"
  echo "Set IVPN DNS: $VPN_DNS"

  scutil <<_EOF
      d.init
      d.add ServerAddresses * ${VPN_DNS}
      d.add DomainName "${DOMAIN_NAME}"
      d.add SetByIVPN "true"

      set State:/Network/IVPN/DNSAlternate
_EOF
}

function define_ivpn_dns {
  DOMAIN_NAME=$1
  VPN_DNS=$2

  #echo "DOMAIN: $DOMAIN_NAME"
  echo "Set IVPN DNS: $VPN_DNS"

  # save IVPN DNS parameters
  scutil <<_EOF
      d.init
      d.add ServerAddresses * ${VPN_DNS}
      d.add DomainName "${DOMAIN_NAME}"
      d.add SetByIVPN "true"

      set State:/Network/IVPN/DNSBase
_EOF
}

function ipv6_resolver_init {
    LOCAL_IPV6_ADDR=$1
    TUN_INTERFACE_NAME=$2

    scutil <<_EOF
        d.init
        d.add Addresses * ${LOCAL_IPV6_ADDR}
        d.add DestAddresses * ::ffff:ffff:ffff:ffff:0:0 ::
        d.add InterfaceName ${TUN_INTERFACE_NAME}
        set State:/Network/Service/ivpn_tunnel/IPv6
        set Setup:/Network/Service/ivpn_tunnel/IPv6
        quit
_EOF
}

#function ipv6_resolver_destroy {
#    scutil <<_EOF
#        remove State:/Network/Service/ivpn_tunnel/IPv6
#        remove Setup:/Network/Service/ivpn_tunnel/IPv6
#        quit
#_EOF
#}

################### PROCESSING ARGUMENTS #######################################

# '-up' is in use by OpenVPN server
if [ "$1" = "-up" ] ; then

    #OpenVPN store DNS IP into 'foreign_option_*' environment variable
    DOMAIN_NAME="ivpn-client"
    FOREIGN_OPTIONS=`env | grep -E '^foreign_option_' | sort | sed -e 's/foreign_option_.*=//'`

    while read -r option
    do
        case ${option} in
            *DOMAIN*)
                DOMAIN_NAME=${option//dhcp-option DOMAIN /}
                ;;
            *DNS*)
                VPN_DNS=${option//dhcp-option DNS /}
                ;;
        esac
    done <<< "${FOREIGN_OPTIONS}"

    define_ivpn_dns $DOMAIN_NAME $VPN_DNS

    store_and_update "Setup"
    store_and_update "State"

# same as '-up' but DNS IP takes as parameter to this command
elif [ "$1" = "-up_set_dns" ] ; then

        DOMAIN_NAME="ivpn-client"
        VPN_DNS=$2 #DNS IP

        define_ivpn_dns $DOMAIN_NAME $VPN_DNS

        store_and_update "Setup"
        store_and_update "State"

# required to be able to resolve IPv6 DNS addresses by the default macOS's domain name resolver
elif [ "$1" = "-up_init_ipv6_resolver" ] ; then
    
    LOCAL_IPV6_ADDR=$2
    TUN_INTERFACE_NAME=$3

    ipv6_resolver_init $LOCAL_IPV6_ADDR $TUN_INTERFACE_NAME

elif [ "$1" = "-update" ] ; then
    if ! is_vpn_dns_set ; then
        echo "Error: Cannot find original DNS configuration"
        exit 0
    fi

    if is_dns_changed "Setup"; then
        echo "DNS Setup:/ changed. Updating...";
        store_and_update "Setup"
    fi;

    if is_dns_changed "State"; then
        echo "DNS State:/ changed. Updating...";
        store_and_update "State"
    fi;

elif [ "$1" = "-set_alternate_dns" ] ; then

  DOMAIN_NAME="ivpn-client"
  VPN_DNS=$2 #DNS IP

  define_alternate_ivpn_dns $DOMAIN_NAME $VPN_DNS

  # update DNS only if it was already updated by us (-up or -up_set_dns)
  if isPrimaryInterfaceDetected; then
    if is_dns_set_by_ivpn "Setup"; then
        store_and_update "Setup"
        store_and_update "State"
    fi
  fi 

elif [ "$1" = "-delete_alternate_dns" ] ; then

  if ! is_alternate_ivpn_dns_defined; then
    echo "Alternate IVPN DNS not defined. Nothing to restore."
    exit 0
  fi;

  scutil <<_EOF
      remove State:/Network/IVPN/DNSAlternate
      quit
_EOF

# update DNS only if it was already updated by us (-up or -up_set_dns)
if isPrimaryInterfaceDetected; then
    if is_dns_set_by_ivpn "Setup"; then
        store_and_update "Setup"
        store_and_update "State"
    fi
fi

elif [ "$1" = "-down" ] ; then

    # TODO: killall -HUP mDNSResponder

    if ! is_vpn_dns_set ; then
        echo "DNS down: Cannot find original DNS configuration. Nothing to remove."
        exit 0
    fi

    if ! isPrimaryInterfaceDetected; then
        echo "Warning: Primary interface not found. Restoring info from backup...."
        PSID=`echo 'show State:/Network/IVPN/PrimaryInterfaceInfo' | scutil | grep PrimaryService | sed -e 's/.*PrimaryService : //'`
    fi

    if ! is_dns_changed "State" ; then
        restore_user_setting "State"
    fi

    if ! is_dns_changed "Setup" ; then
        restore_user_setting "Setup"
    fi

    # not necessary to call 'ipv6_resolver_destroy' since it's commands available in the next lines
    echo "Removing lefovers..."
    scutil <<_EOF
        remove State:/Network/IVPN/Original/DNS/Setup
        remove State:/Network/IVPN/Original/DNS/State
        remove State:/Network/IVPN/PrimaryInterfaceInfo
        remove State:/Network/IVPN/DNSBase

        remove State:/Network/Service/ivpn_tunnel/IPv6
        remove Setup:/Network/Service/ivpn_tunnel/IPv6

        quit
_EOF

elif [ "$1" = "-pause" ] ; then

    if ! is_vpn_dns_set ; then
        echo "Error: Cannot find original DNS configuration"
        exit 0
    fi

    if ! is_dns_changed "State" ; then
        restore_user_setting "State"
    fi

    if ! is_dns_changed "Setup" ; then
        restore_user_setting "Setup"
    fi

elif [ "$1" = "-resume" ] ; then

  if ! is_vpn_dns_set ; then
      echo "Error: Cannot find original DNS configuration"
      exit 0
  fi

  if is_dns_changed "Setup"; then
      store_and_update "Setup"
  fi;

  if is_dns_changed "State"; then
      store_and_update "State"
  fi;

elif [ "$1" = "-is_main_interface_detected" ] ; then
    ensurePrimaryInterfaceDetected

else
    print_state
    exit 1
fi
