#!/bin/bash

# Useful commands:
# Show all rules/anchors
#   sudo pfctl -s rules
# Show all rules for "ivpn_firewall" anchor
#   sudo pfctl -a "ivpn_firewall" -s rules
#   sudo pfctl -a "ivpn_firewall/tunnel" -s rules
# Show table
#   sudo pfctl -a "ivpn_firewall" -t ivpn_servers -T show
#   sudo pfctl -a "ivpn_firewall" -t ivpn_exceptions -T show
# Logging:
#   sudo ifconfig pflog1 create         # create log interface
#   sudo tcpdump -nnn -e -ttt -i pflog1 # start realtime monitoring in terminal
#   Modify rules (example: "pass out log (all, to pflog1) from any to 8.8.8.8")
# Restoring:
#   sudo pfctl -d                       # disable PF
#   sudo pfctl -f /etc/pf.conf          # load default OS rules set

PATH=/sbin:/usr/sbin:$PATH

ANCHOR="ivpn_firewall"
SA_BLOCK_DNS="block_dns"
SA_TUNNEL="tunnel"

TBL_EXCEPTIONS="exceptions"
TBL_USER_EXCEPTIONS="user_exceptions"

# If IS_DO_ROUTING=1, all traffic will be intentionally routed through the VPN interface:
#   Any packets that do not follow the default routing configuration and still use a non-VPN interface
#   will be NAT-ed to the VPN interface's IP address and then routed through the VPN.
#
# This helps resolve issues like those in macOS 15.0, where certain apps (such as iMessage and FaceTime) stop working when the VPN is connected.
# These services ignore the routing configuration and continue using the "en0" interface, bypassing the VPN.
IS_DO_ROUTING=1

ROUTE_SA_INIT="route_init"
ROUTE_SA_ALL="route_all"
ROUTE_TBL_DNS="tbl_route_dns"

# Checks whether anchor is present in the system
# 0 - if anchor is present
# 1 - if not present
function get_anchor_present {
    pfctl -sr 2> /dev/null | grep -q "anchor.*${ANCHOR}"
}
function get_anchor_present_nat {
    pfctl -sn 2> /dev/null | grep -q "nat-anchor.*${ANCHOR}"
}

# Add IVPN Firewall anchor after existing pf rules.
function install_anchor {
    cat \
      <(pfctl -sr 2> /dev/null) \
      <(echo "anchor ${ANCHOR} all") \
       | pfctl -R -f -
}
function install_anchor_nat {
    cat \
      <(echo "nat-anchor '${ANCHOR}' all ") \
      <(pfctl -sn 2> /dev/null) \
       | pfctl -N -f -    
}

# Checks whether IVPN Firewall anchor exists
# and add it if require
function add_anchor_if_required {
    get_anchor_present
    if (( $? != 0 )) ; then    
        install_anchor
    fi

    if (( ${IS_DO_ROUTING} == 1 )) ; then
        get_anchor_present_nat
        if (( $? != 0 )) ; then    
            install_anchor_nat
        fi
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
    if [[ -n `pfctl -a $ANCHOR -sr` ]] ; then
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

    pfctl -a ${ANCHOR} -f - <<_EOF
      scrub all fragment reassemble
      
      nat-anchor ${ROUTE_SA_INIT} all
      nat-anchor ${ROUTE_SA_ALL} all

      table <${TBL_EXCEPTIONS}>       persist
      table <${TBL_USER_EXCEPTIONS}>  persist

      pass quick on lo0 all flags any keep state

      anchor ${ROUTE_SA_INIT}  all 

      pass out quick from any to <${TBL_EXCEPTIONS}>       flags S/SA  keep state
      pass in  quick from <${TBL_EXCEPTIONS}> to any       flags S/SA  keep state
      pass out quick from any to <${TBL_USER_EXCEPTIONS}>  flags any   keep state
      pass in  quick from <${TBL_USER_EXCEPTIONS}> to any  flags any   keep state
  
      pass out quick inet proto udp from any port = 68 to 255.255.255.255 port = 67 no state
      pass in  quick inet proto udp from any port = 67 to any             port = 68 no state

      anchor ${SA_BLOCK_DNS}  all           # IMPORTANT to block unwanted DNS requests before they are routed to the VPN
      anchor ${SA_TUNNEL}     all           # Allowing traffic to VPN interface and VPN server
      anchor ${ROUTE_SA_ALL}  all           # Intentionally ROUTE all the rest traffic through VPN interface

      block return out quick all
      block drop quick all
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
    client_disconnected

    # remove all entries in exceptions table
    pfctl -a ${ANCHOR} -t ${TBL_EXCEPTIONS}      -T flush
    pfctl -a ${ANCHOR} -t ${TBL_USER_EXCEPTIONS} -T flush

    # remove all rules from SA_BLOCK_DNS anchor
    pfctl -a ${ANCHOR}/${SA_BLOCK_DNS} -Fr

    # remove all the rules from anchor     
    pfctl -a ${ANCHOR} -Fr
    pfctl -a ${ANCHOR} -Fn

    local TOKEN=`echo 'show State:/Network/IVPN/PacketFilter' | scutil | grep Token | sed -e 's/.*: //' | tr -d ' \n'`
    pfctl -X "${TOKEN}"

    echo "IVPN Firewall disabled"
}

function client_connected {

    IFACE=$1
    SRC_ADDR=$2
    SRC_PORT=$3
    DST_ADDR=$4
    DST_PORT=$5
    PROTOCOL=$6

    # FILTER RULES (TUNNEL)
    pfctl -a ${ANCHOR}/${SA_TUNNEL} -f - <<_EOF
        pass quick on ${IFACE} all flags S/SA keep state                                          # Pass all traffic on VPN interface
        pass out quick proto ${PROTOCOL} from any to ${DST_ADDR} port = ${DST_PORT} keep state    # Pass all traffic to VPN server      
_EOF

    if (( ${IS_DO_ROUTING} == 1 )) ; then
        # NAT & ROUTING RULES
        #
        # All traffic will be intentionally routed through the VPN interface:
        #   Any packets that do not follow the default routing configuration and still use a non-VPN interface
        #   will be NAT-ed to the VPN interface's IP address and then routed through the VPN.
        #
        # This helps resolve issues like those in macOS 15.0, where certain apps (such as iMessage and FaceTime) stop working when the VPN is connected.
        # These services ignore the routing configuration and continue using the "en0" interface, bypassing the VPN.

        # Initialize intentional routing (NAT and filter rules) 
        pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -f - <<_EOF            
            table <${ROUTE_TBL_DNS}>       persist  # DNS addresses that need to be NAT-ed and routed through VPN interface
            table <${TBL_EXCEPTIONS}>      persist  # Table similar (copy) to table defined in ${ANCHOR} anchor
            table <${TBL_USER_EXCEPTIONS}> persist  # Table similar (copy) to table defined in ${ANCHOR} anchor

            #
            #   === NAT rules ===
            #   NAT rules are required to change SRC address for all traffic to VPN interface IP.
            #   First NAT rule wins, so we need to put more specific rules first
            #   NOTE: NAT rules are processed BEFORE ANY filter rules!
            #
          
            #   Do not NAT loopback packets
            no nat on lo0 all
            #   Do not NAT packets on VPN interface
            no nat on ${IFACE} all

            #   NAT: packets to internal DNS server
            #   NAT and ROUTE it before all other rules to avoid conflicts with exception addresses
            nat from any to <${ROUTE_TBL_DNS}> port 53 -> ${SRC_ADDR}

            #   Do not NAT addresses from EXCEPTIONS
            no nat from any to <${TBL_EXCEPTIONS}>
            no nat from any to <${TBL_USER_EXCEPTIONS}>
            no nat from <${TBL_USER_EXCEPTIONS}> to any

            #   Do not NAT LAN addresses
            no nat from any to { 172.16.0.0/12, 192.168.0.0/16, 10.0.0.0/8, 169.254.0.0/16, 255.255.255.255, 224.0.0.0/24, 239.0.0.0/8 }
            no nat from any to { fe80::/10, fc00::/7, ff01::/16, ff02::/16, ff03::/16, ff04::/16, ff05::/16, ff08::/16 }

            #   Do not NAT packets to remote server
            no nat inet from any to ${DST_ADDR}
         
            #   NAT: Change SRC address for all traffic to IP of VPN interface
            nat inet all -> ${SRC_ADDR}

            #   === FILTER rules ===

            #   ROUTE traffic to internal DNS server
            pass out quick route-to ${IFACE} inet  proto {udp, tcp} from any to <${ROUTE_TBL_DNS}> port 53 flags S/SA keep state            
            pass out quick route-to ${IFACE} inet6 proto {udp, tcp} from any to <${ROUTE_TBL_DNS}> port 53 flags S/SA keep state
_EOF

        # Route all other traffic
        # This anchor must be the last one in the list of anchors: 
        #  - all allowed traffic must be already passed
        #  - all unwanted traffic must be already blocked (e.g. DNS requests)
        pfctl -a ${ANCHOR}/${ROUTE_SA_ALL} -f - <<_EOF
          #   If we are here, then the DNS server IP is OK (unexpected DNS already blocked) 
          #   and not accessible through the VPN interface (it is on the local network)
          pass out quick proto { udp, tcp } from any to any port 53 flags S/SA keep state

          #   Route all traffic through VPN interface
          pass out quick route-to ${IFACE} inet  all flags S/SA keep state
          pass out quick route-to ${IFACE} inet6 all flags S/SA keep state
_EOF
    fi
}

function client_disconnected {
    pfctl -a ${ANCHOR}/${SA_TUNNEL} -Fr
    
    if (( ${IS_DO_ROUTING} == 1 )) ; then
      pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -t ${ROUTE_TBL_DNS}        -T flush
      pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -t ${TBL_EXCEPTIONS}       -T flush
      pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -t ${TBL_USER_EXCEPTIONS}  -T flush
      pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -Fn
      pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -Fr
      pfctl -a ${ANCHOR}/${ROUTE_SA_ALL} -Fr
    fi
}

function set_dns {
  
  # IS_LAN: "true" or "false":
  #  - if "true" then DNS is custom local non-routable IP (not in VPN network)
  #    This IP must be skipped from NAT-ing and routing through VPN interface
  #  - if "false" then DNS must be routed through VPN interface
  IS_LAN=$1 
  DNS=$2  

  # remove all rules in ${SA_BLOCK_DNS} anchor
  pfctl -a ${ANCHOR}/${SA_BLOCK_DNS} -Fr

  if (( ${IS_DO_ROUTING} == 1 )) ; then
    pfctl -a ${ANCHOR}/${ROUTE_SA_INIT} -t ${ROUTE_TBL_DNS}       -T flush
  fi

  if [[ -z "${DNS}" ]] ; then
      # DNS not defined. Block all connections to port 53
      pfctl -a ${ANCHOR}/${SA_BLOCK_DNS} -f - <<_EOF
        block return out quick proto udp from any to port = 53
        block return out quick proto tcp from any to port = 53
_EOF
      return 0
  fi

  if (( ${IS_DO_ROUTING} == 1 )) ; then
    if [[ "${IS_LAN}" = "false" ]] ; then
      # Add DNS server to the table of addresses that need to be NAT-ed (and routed through VPN interface)
      # DNS server is accessible only via VPN interface
      pfctl -a "${ANCHOR}/${ROUTE_SA_INIT}" -t "${ROUTE_TBL_DNS}"       -T replace ${DNS}
    fi
  fi

  # Block all DNS requests except to the specified DNS server
  pfctl -a "${ANCHOR}/${SA_BLOCK_DNS}" -f - <<_EOF
        block return out quick proto { udp, tcp } from any to ! ${DNS}  port = 53
_EOF

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
      pfctl -a "${ANCHOR}" -t "${TBL_EXCEPTIONS}" -T add $@
      
      if (( ${IS_DO_ROUTING} == 1 )) ; then
        pfctl -a "${ANCHOR}/${ROUTE_SA_ALL}" -t "${TBL_EXCEPTIONS}" -T add $@
      fi

    elif [[ $1 = "-remove_exceptions" ]]; then    

      shift
      pfctl -a "${ANCHOR}" -t "${TBL_EXCEPTIONS}" -T delete $@

      if (( ${IS_DO_ROUTING} == 1 )) ; then
        pfctl -a "${ANCHOR}/${ROUTE_SA_ALL}" -t "${TBL_EXCEPTIONS}" -T delete $@
      fi
    
    elif [[ $1 = "-set_user_exceptions" ]]; then    

      shift
      pfctl -a "${ANCHOR}" -t "${TBL_USER_EXCEPTIONS}" -T replace $@

      if (( ${IS_DO_ROUTING} == 1 )) ; then
        pfctl -a "${ANCHOR}/${ROUTE_SA_ALL}" -t "${TBL_USER_EXCEPTIONS}" -T replace $@
      fi

    elif [[ $1 = "-connected" ]]; then       
        
        IFACE=$2
        SRC_ADDR=$3
        SRC_PORT=$4
        DST_ADDR=$5
        DST_PORT=$6
        PROTOCOL=$7

        client_connected ${IFACE} ${SRC_ADDR} ${SRC_PORT} ${DST_ADDR} ${DST_PORT} ${PROTOCOL}

    elif [[ $1 = "-disconnected" ]]; then
        shift
        client_disconnected
    elif [[ $1 = "-set_dns" ]]; then    

        get_firewall_enabled || return 0
        
        IS_LAN=$2 # "true" or "false"; if true, then DNS is custom local non-routable IP (not in VPN network)
        IP=$3

        set_dns ${IS_LAN} ${IP}

    else
        echo "Unknown command"
        return 2
    fi
}

main $@


