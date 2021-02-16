#!/bin/bash

# Useful commands:
# Show all rules/anchors
#   sudo pfctl -s rules
# Show all rules for "ivpn_firewall" anchor
#   sudo pfctl -a "ivpn_firewall" -s rules
#   sudo pfctl -a "ivpn_firewall/tunnel" -s rules
# Show table
#   sudo pfctl -a "ivpn_firewall" -t ivpn_servers -T show

PATH=/sbin:/usr/sbin:$PATH

ANCHOR_NAME="ivpn_firewall"
EXCEPTIONS_TABLE="ivpn_servers"

# Checks whether anchor is present in the system
# 0 - if anchor is present
# 1 - if not present
function get_anchor_present {
    pfctl -sr 2> /dev/null | grep -q "anchor.*${ANCHOR_NAME}"
}

# Add IVPN Firewall anchor after existing pf rules.
function install_anchor {
    cat \
      <(pfctl -sr 2> /dev/null) \
      <(echo "anchor ${ANCHOR_NAME} all") \
       | pfctl -f -
}

# Checks whether IVPN Firewall anchor exists
# and add it if require
function add_anchor_if_required {
  
    get_anchor_present

    if (( $? != 0 )) ; then    
        install_anchor
    fi
}

# Checks if the IVPN Firewall is enabled
# 0 - if enabled
# 1 - if not enabled
function get_firewall_enabled {

    # Checks if anchor is present
    get_anchor_present
    if (( $? != 0 )) ; then
        return 1
    fi

    # Checks if pf is enabled
    pfctl -si 2> /dev/null | grep -i "status: enabled" > /dev/null
    if (( $? != 0 )) ; then
      return 1
    fi

    # Checks if rules are present in the anchor
    if [[ -n `pfctl -a $ANCHOR_NAME -sr` ]] ; then
      return 0
    fi

    return 1
}

# Load rules into the anchor and enable the firewall if disabled
function enable_firewall {
    get_firewall_enabled

    if (( $? == 0 )); then
      echo "Firewall is already enabled. Please disable it first" >&2
      return 0
    fi

    set -e

    pfctl -a ${ANCHOR_NAME} -f - <<_EOF
      block drop out on ! lo0 all
      block drop in on ! lo0 all

      table <${EXCEPTIONS_TABLE}> persist

      pass out from any to <${EXCEPTIONS_TABLE}>
      pass in from <${EXCEPTIONS_TABLE}> to any

      pass out inet proto udp from 0.0.0.0 to 255.255.255.255 port = 67
      pass in proto udp from any to any port = 68

      anchor tunnel all

_EOF

    local TOKEN=`pfctl -E 2>&1 | grep -i token | sed -e 's/.*oken.*://' | tr -d ' \n'`

    scutil <<_EOF
      d.init
      d.add Token "${TOKEN}"
      set State:/Network/IVPN/PacketFilter

      quit
_EOF

    set +e

    echo "IVPN Firewall enabled"
}


# Remove all rules from the anchor and disable the firewall
function disable_firewall {

    # remove all entries in exceptions table
    pfctl -a ${ANCHOR_NAME} -t ${EXCEPTIONS_TABLE} -T flush

    # remove all rules in tun anchor
    pfctl -a ${ANCHOR_NAME}/tunnel -Fr

    # remove all the rules in anchor
    pfctl -a ${ANCHOR_NAME} -Fr 

    local TOKEN=`echo 'show State:/Network/IVPN/PacketFilter' | scutil | grep Token | sed -e 's/.*: //' | tr -d ' \n'`
    pfctl -X "${TOKEN}"

    echo "IVPN Firewall disabled"
}

function client_connected {

    IFACE=$1
    pfctl -a ${ANCHOR_NAME}/tunnel -f - <<_EOF
        pass out on ${IFACE} from any to any
        pass in on ${IFACE} from any to any 
_EOF

}

function client_disconnected {
    pfctl -a ${ANCHOR_NAME}/tunnel -Fr
}

function main {

    if [[ $1 = "-enable" ]] ; then

      add_anchor_if_required
      enable_firewall

    elif [[ $1 = "-disable" ]] ; then

      disable_firewall

    elif [[ $1 = "-status" ]] ; then

      get_firewall_enabled

      if (( $? == 0 )); then
        echo "IVPN Firewall is enabled"
        return 0
      else
        echo "IVPN Firewall is disabled"      
        return 1
      fi

    elif [[ $1 = "-add_exceptions" ]]; then    

      shift
      pfctl -a "${ANCHOR_NAME}" -t "${EXCEPTIONS_TABLE}" -T add $@

    elif [[ $1 = "-remove_exceptions" ]]; then    

      shift
      pfctl -a "${ANCHOR_NAME}" -t "${EXCEPTIONS_TABLE}" -T delete $@

    elif [[ $1 = "-connected" ]]; then       
        
        IFACE=$2  
        client_connected ${IFACE} 

    elif [[ $1 = "-disconnected" ]]; then
        shift
        client_disconnected
    else
        echo "Unknown command"
        return 2
    fi
}

main $@


