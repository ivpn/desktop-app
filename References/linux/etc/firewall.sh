#!/bin/bash

IPv4BIN=/sbin/iptables
IPv6BIN=/sbin/ip6tables

IN_IVPN=IVPN-IN
OUT_IVPN=IVPN-OUT

IN_IVPN_IF=IVPN-IN-VPN
OUT_IVPN_IF=IVPN-OUT-VPN

# returns 0 if chain exists
function chain_exists()
{
    local bin=$1
    local chain_name=$2
    ${bin} -n -L ${chain_name} >/dev/null 2>&1
}

function create_chain()
{
  local bin=$1
  local chain_name=$2
  chain_exists ${bin} ${chain_name} || ${bin} -N ${chain_name}
}

# Checks if the IVPN Firewall is enabled
# 0 - if enabled
# 1 - if not enabled
function get_firewall_enabled {
  chain_exists ${IPv4BIN} ${OUT_IVPN}
}

# Load rules
function enable_firewall {
    get_firewall_enabled

    if (( $? == 0 )); then
      echo "Firewall is already enabled. Please disable it first" >&2
      return 1
    fi

    set -e

    ### IPv6 ###
    # IPv6: block everything by default
    ${IPv6BIN} -P INPUT DROP
    ${IPv6BIN} -P OUTPUT DROP
    # IPv6: define chains
    create_chain ${IPv6BIN} ${IN_IVPN}
    create_chain ${IPv6BIN} ${OUT_IVPN}
    # IPv6: allow  local (lo) interface
    ${IPv6BIN} -A ${OUT_IVPN} -o lo -j ACCEPT
    ${IPv6BIN} -A ${IN_IVPN} -i lo -j ACCEPT
    # IPv6: assign our chains to global (global -> IVPN_CHAIN -> IVPN_VPN_CHAIN)
    ${IPv6BIN} -A OUTPUT -j ${OUT_IVPN}
    ${IPv6BIN} -A INPUT -j ${IN_IVPN}

    ### IPv4 ###
    # block everything by default
    ${IPv4BIN} -P INPUT DROP
    ${IPv4BIN} -P OUTPUT DROP

    # define chains
    create_chain ${IPv4BIN} ${IN_IVPN}
    create_chain ${IPv4BIN} ${OUT_IVPN}

    create_chain ${IPv4BIN} ${IN_IVPN_IF}
    create_chain ${IPv4BIN} ${OUT_IVPN_IF}

    # allow  local (lo) interface
    ${IPv4BIN} -A ${OUT_IVPN} -o lo -j ACCEPT
    ${IPv4BIN} -A ${IN_IVPN} -i lo -j ACCEPT

    # allow DHCP port (67out 68in)
    ${IPv4BIN} -A ${OUT_IVPN} -p udp --dport 67 -j ACCEPT
    ${IPv4BIN} -A ${IN_IVPN} -p udp --dport 68 -j ACCEPT

    # enable all ICMP ping outgoing request (needed to be able to ping VPN servers)
    ${IPv4BIN} -A ${OUT_IVPN} -p icmp --icmp-type 8 -d 0/0 -m state --state NEW,ESTABLISHED,RELATED -j ACCEPT
    ${IPv4BIN} -A ${IN_IVPN} -p icmp --icmp-type 0 -s 0/0 -m state --state ESTABLISHED,RELATED -j ACCEPT

    # assign our chains to global (global -> IVPN_CHAIN -> IVPN_VPN_CHAIN)
    ${IPv4BIN} -A OUTPUT -j ${OUT_IVPN}
    ${IPv4BIN} -A INPUT -j ${IN_IVPN}
    ${IPv4BIN} -A ${OUT_IVPN} -j ${OUT_IVPN_IF}
    ${IPv4BIN} -A ${IN_IVPN} -j ${IN_IVPN_IF}

    set +e

    echo "IVPN Firewall enabled"
}


# Remove all rules
function disable_firewall {
    # Flush rules and delete custom chains

    ### IPv4 ###
    ${IPv4BIN} -D OUTPUT -j ${OUT_IVPN}
    ${IPv4BIN} -D INPUT -j ${IN_IVPN}
    ${IPv4BIN} -D ${OUT_IVPN} -j ${OUT_IVPN_IF}
    ${IPv4BIN} -D ${IN_IVPN} -j ${IN_IVPN_IF}

    ${IPv4BIN} -F ${OUT_IVPN_IF}
    ${IPv4BIN} -F ${IN_IVPN_IF}
    ${IPv4BIN} -F ${OUT_IVPN}
    ${IPv4BIN} -F ${IN_IVPN}

    ${IPv4BIN} -X ${OUT_IVPN_IF}
    ${IPv4BIN} -X ${IN_IVPN_IF}
    ${IPv4BIN} -X ${OUT_IVPN}
    ${IPv4BIN} -X ${IN_IVPN}

    ### IPv6 ###
    ${IPv6BIN} -D OUTPUT -j ${OUT_IVPN}
    ${IPv6BIN} -D INPUT -j ${IN_IVPN}
    ${IPv6BIN} -F ${OUT_IVPN}
    ${IPv6BIN} -F ${IN_IVPN}
    ${IPv6BIN} -X ${OUT_IVPN}
    ${IPv6BIN} -X ${IN_IVPN}

    ### allow everything by default ###
    ${IPv4BIN} -P INPUT ACCEPT
    ${IPv4BIN} -P OUTPUT ACCEPT
    ${IPv6BIN} -P INPUT ACCEPT
    ${IPv6BIN} -P OUTPUT ACCEPT

    echo "IVPN Firewall disabled"
}

function client_connected {
  IFACE=$1
  ${IPv4BIN} -A ${OUT_IVPN_IF} -o ${IFACE} -j ACCEPT
  ${IPv4BIN} -A ${IN_IVPN_IF} -i ${IFACE} -j ACCEPT
}

function client_disconnected {
  ${IPv4BIN} -F ${OUT_IVPN_IF}
  ${IPv4BIN} -F ${IN_IVPN_IF}
}

function main {

    if [[ $1 = "-enable" ]] ; then

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
      create_chain ${IPv4BIN} ${IN_IVPN_IF}
      create_chain ${IPv4BIN} ${OUT_IVPN_IF}
      ${IPv4BIN} -A ${OUT_IVPN_IF} -d $@ -j ACCEPT
      ${IPv4BIN} -A ${OUT_IVPN_IF} -s $@ -j ACCEPT
      ${IPv4BIN} -A ${IN_IVPN_IF} -d $@ -j ACCEPT
      ${IPv4BIN} -A ${IN_IVPN_IF} -s $@ -j ACCEPT

    elif [[ $1 = "-remove_exceptions" ]]; then

      shift
      ${IPv4BIN} -D ${OUT_IVPN_IF} -d $@ -j ACCEPT
      ${IPv4BIN} -D ${OUT_IVPN_IF} -s $@ -j ACCEPT
      ${IPv4BIN} -D ${IN_IVPN_IF} -d $@ -j ACCEPT
      ${IPv4BIN} -D ${IN_IVPN_IF} -s $@ -j ACCEPT

    elif [[ $1 = "-connected" ]]; then
        client_connected $2
    elif [[ $1 = "-disconnected" ]]; then
        shift
        client_disconnected
    else
        echo "Unknown command"
        return 2
    fi
}

main $@
