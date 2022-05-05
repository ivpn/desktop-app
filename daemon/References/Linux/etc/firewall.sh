#!/bin/bash

#
#  Daemon for IVPN Client Desktop
#  https://github.com/ivpn/desktop-app/daemon
#
#  Created by Stelnykovych Alexandr.
#  Copyright (c) 2020 Privatus Limited.
#
#  This file is part of the Daemon for IVPN Client Desktop.
#
#  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
#  modify it under the terms of the GNU General Public License as published by the Free
#  Software Foundation, either version 3 of the License, or (at your option) any later version.
#
#  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
#  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
#  details.
#
#  You should have received a copy of the GNU General Public License
#  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
#

# Useful commands
#   List all rules:
#     sudo iptables -L -v
#     or
#     sudo iptables -S

IPv4BIN=iptables
IPv6BIN=ip6tables

LOCKWAITTIME=2

# main chains for IVPN firewall
IN_IVPN=IVPN-IN
OUT_IVPN=IVPN-OUT
# chain for DNS rules
OUT_IVPN_DNS=IVPN-OUT-DNS
# IVPN chains for VPN dependend rules (applicable when VPN enabled)
IN_IVPN_IF=IVPN-IN-VPN
OUT_IVPN_IF=IVPN-OUT-VPN
# chain for non-VPN depended exceptios (applicable all time when firewall enabled)
# can be used, for example, for 'allow LAN' functionality
IN_IVPN_STAT_EXP=IVPN-IN-STAT-EXP
OUT_IVPN_STAT_EXP=IVPN-OUT-STAT-EXP
# chain for user-defined exceptios (applicable all time when firewall enabled)
IN_IVPN_STAT_USER_EXP=IVPN-IN-STAT-USER-EXP
OUT_IVPN_STAT_USER_EXP=IVPN-OUT-STAT-USER-EXP
# chain for non-VPN depended exceptios: only for ICMP protocol (ping)
IN_IVPN_ICMP_EXP=IVPN-IN-ICMP-EXP
OUT_IVPN_ICMP_EXP=IVPN-OUT-ICMP-EXP

# returns 0 if chain exists
function chain_exists()
{
    local bin=$1
    local chain_name=$2
    ${bin} -w ${LOCKWAITTIME} -n -L ${chain_name} >/dev/null 2>&1
}

function create_chain()
{
  local bin=$1
  local chain_name=$2
  chain_exists ${bin} ${chain_name} || ${bin} -w ${LOCKWAITTIME} -N ${chain_name}
}

# erase rules in a chain
function clean_chain() {
  BIN=$1
  CH=$2
  ${BIN} -w ${LOCKWAITTIME} -F ${CH}
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
      return 0
    fi

    set -e

    if [ -f /proc/net/if_inet6 ]; then
      ### IPv6 ###

      # IPv6: define chains
      create_chain ${IPv6BIN} ${IN_IVPN}
      create_chain ${IPv6BIN} ${OUT_IVPN}
      create_chain ${IPv6BIN} ${OUT_IVPN_DNS}
      create_chain ${IPv6BIN} ${IN_IVPN_IF}
      create_chain ${IPv6BIN} ${OUT_IVPN_IF}
      create_chain ${IPv6BIN} ${IN_IVPN_STAT_USER_EXP}
      create_chain ${IPv6BIN} ${OUT_IVPN_STAT_USER_EXP}

      # IPv6: allow  local (lo) interface
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -o lo -j ACCEPT
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -i lo -j ACCEPT

      # allow link-local addresses
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -s FE80::/10 -j ACCEPT
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -d FE80::/10 -j ACCEPT

      # allow unique-local addresses
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -s FD00::/8 -j ACCEPT
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -d FD00::/8 -j ACCEPT

      # allow DHCP port (547out 546in)
      # ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -p udp --dport 547 -j ACCEPT
      # ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -p udp --dport 546 -j ACCEPT

      # IPv6: assign our chains to global (global -> IVPN_CHAIN -> IVPN_VPN_CHAIN)
      ${IPv6BIN} -w ${LOCKWAITTIME} -A OUTPUT -j ${OUT_IVPN}
      ${IPv6BIN} -w ${LOCKWAITTIME} -A INPUT -j ${IN_IVPN}

      # block DNS for IPv6
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_DNS}
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p udp --dport 53 -j DROP
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p tcp --dport 53 -j DROP

      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_IF}
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_IF}
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_STAT_USER_EXP}
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_STAT_USER_EXP}

      # IPv6: block everything by default
      ${IPv6BIN} -w ${LOCKWAITTIME} -P INPUT DROP
      ${IPv6BIN} -w ${LOCKWAITTIME} -P OUTPUT DROP

    else
      echo "IPv6 disabled: skipping IPv6 rules"
    fi

    ### IPv4 ###

    # define chains
    create_chain ${IPv4BIN} ${IN_IVPN}
    create_chain ${IPv4BIN} ${OUT_IVPN}

    create_chain ${IPv4BIN} ${OUT_IVPN_DNS}

    create_chain ${IPv4BIN} ${IN_IVPN_IF}
    create_chain ${IPv4BIN} ${OUT_IVPN_IF}

    create_chain ${IPv4BIN} ${IN_IVPN_STAT_EXP}
    create_chain ${IPv4BIN} ${OUT_IVPN_STAT_EXP}

    create_chain ${IPv4BIN} ${IN_IVPN_STAT_USER_EXP}
    create_chain ${IPv4BIN} ${OUT_IVPN_STAT_USER_EXP}

    create_chain ${IPv4BIN} ${IN_IVPN_ICMP_EXP}
    create_chain ${IPv4BIN} ${OUT_IVPN_ICMP_EXP}

    # allow  local (lo) interface
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -o lo -j ACCEPT
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -i lo -j ACCEPT

    # allow DHCP port (67out 68in)
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -p udp --dport 67 -j ACCEPT
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -p udp --dport 68 -j ACCEPT

    # enable all ICMP ping outgoing request (needed to be able to ping VPN servers)
    #${IPv4BIN} -A ${OUT_IVPN} -p icmp --icmp-type 8 -d 0/0 -m state --state NEW,ESTABLISHED,RELATED -j ACCEPT
    #${IPv4BIN} -A ${IN_IVPN} -p icmp --icmp-type 0 -s 0/0 -m state --state ESTABLISHED,RELATED -j ACCEPT

    # assign our chains to global
    # (global -> IVPN_CHAIN -> IVPN_VPN_CHAIN)
    # (global -> IVPN_CHAIN -> IN_IVPN_STAT_EXP)
    ${IPv4BIN} -w ${LOCKWAITTIME} -A OUTPUT -j ${OUT_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A INPUT -j ${IN_IVPN}

    # block DNS by default
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_DNS}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p udp --dport 53 -j DROP
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p tcp --dport 53 -j DROP

    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_IF}

    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN} -j ${OUT_IVPN_ICMP_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN} -j ${IN_IVPN_ICMP_EXP}

    # block everything by default
    ${IPv4BIN} -w ${LOCKWAITTIME} -P INPUT DROP
    ${IPv4BIN} -w ${LOCKWAITTIME} -P OUTPUT DROP

    set +e

    echo "IVPN Firewall enabled"
}

# Remove all rules
function disable_firewall {
    # Flush rules and delete custom chains

    ### allow everything by default ###
    ${IPv4BIN} -w ${LOCKWAITTIME} -P INPUT ACCEPT
    ${IPv4BIN} -w ${LOCKWAITTIME} -P OUTPUT ACCEPT
    ${IPv6BIN} -w ${LOCKWAITTIME} -P INPUT ACCEPT
    ${IPv6BIN} -w ${LOCKWAITTIME} -P OUTPUT ACCEPT

    ### IPv4 ###
    # '-D' Delete matching rule from chain
    ${IPv4BIN} -w ${LOCKWAITTIME} -D OUTPUT -j ${OUT_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D INPUT -j ${IN_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_DNS}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_ICMP_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_ICMP_EXP}
    # '-F' Delete all rules in  chain or all chains
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_DNS}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_ICMP_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_ICMP_EXP}
    # '-X' Delete a user-defined chain
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_DNS}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_IF}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_ICMP_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_ICMP_EXP}

    ### IPv6 ###
    ${IPv6BIN} -w ${LOCKWAITTIME} -D OUTPUT -j ${OUT_IVPN}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D INPUT -j ${IN_IVPN}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_DNS}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_STAT_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_STAT_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${OUT_IVPN} -j ${OUT_IVPN_STAT_USER_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -D ${IN_IVPN} -j ${IN_IVPN_STAT_USER_EXP}

    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_DNS}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_STAT_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_STAT_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_STAT_USER_EXP}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_STAT_USER_EXP}

    ${IPv6BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_DNS}
    ${IPv6BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN}
    ${IPv6BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_STAT_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${OUT_IVPN_STAT_USER_EXP}
    ${IPv4BIN} -w ${LOCKWAITTIME} -X ${IN_IVPN_STAT_USER_EXP}
    echo "IVPN Firewall disabled"
}

function client_connected {
  IFACE=$1

  # allow all packets to VPN interface
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_IF} -o ${IFACE} -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN_IF} -i ${IFACE} -j ACCEPT

  if [ -f /proc/net/if_inet6 ]; then
      ### IPv6 ###

      # allow all packets to VPN interface
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_IF} -o ${IFACE} -j ACCEPT
      ${IPv6BIN} -w ${LOCKWAITTIME} -A ${IN_IVPN_IF} -i ${IFACE} -j ACCEPT
    fi
}

function client_disconnected {
  ${IPv4BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_IF}
  ${IPv4BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_IF}

  if [ -f /proc/net/if_inet6 ]; then
    ### IPv6 ###
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${OUT_IVPN_IF}
    ${IPv6BIN} -w ${LOCKWAITTIME} -F ${IN_IVPN_IF}
  fi
}

function add_exceptions {
  BIN=$1
  IN_CH=$2
  OUT_CH=$3
  shift 3
  EXP=$@

  create_chain ${BIN} ${IN_CH}
  create_chain ${BIN} ${OUT_CH}

  #add new rule
  # '-C' option is checking if the rule already exists (needed to avoid duplicates)
  ${BIN} -w ${LOCKWAITTIME} -C ${IN_CH} -s $@ -j ACCEPT || ${BIN} -w ${LOCKWAITTIME} -A ${IN_CH} -s $@ -j ACCEPT
  ${BIN} -w ${LOCKWAITTIME} -C ${OUT_CH} -d $@ -j ACCEPT || ${BIN} -w ${LOCKWAITTIME} -A ${OUT_CH} -d $@ -j ACCEPT
}

function add_direction_exception {
  IN_CH=$1
  OUT_CH=$2

  #SRC_PORT=$3
  DST_ADDR=$4
  DST_PORT=$5
  PROTOCOL=$6

  create_chain ${IPv4BIN} ${IN_CH}
  create_chain ${IPv4BIN} ${OUT_CH}

  #add new rule
  # ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_CH}  -s ${DST_ADDR} -p ${PROTOCOL} --sport ${DST_PORT} --dport ${SRC_PORT} -j ACCEPT
  # ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_CH} -d ${DST_ADDR} -p ${PROTOCOL} --sport ${SRC_PORT} --dport ${DST_PORT} -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_CH}  -s ${DST_ADDR} -p ${PROTOCOL} --sport ${DST_PORT} -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_CH} -d ${DST_ADDR} -p ${PROTOCOL} --dport ${DST_PORT} -j ACCEPT
}

function remove_exceptions {
  BIN=$1
  IN_CH=$2
  OUT_CH=$3
  shift 3
  EXP=$@

  ${BIN} -w ${LOCKWAITTIME} -D ${IN_CH} -s $@ -j ACCEPT
  ${BIN} -w ${LOCKWAITTIME} -D ${OUT_CH} -d $@ -j ACCEPT
}

function remove_exceptions_icmp {
  IN_CH=$1
  OUT_CH=$2
  shift 2
  EXP=$@

  ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_CH} -p icmp --icmp-type 0 -s $@ -m state --state ESTABLISHED,RELATED -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_CH} -p icmp --icmp-type 8 -d $@ -m state --state NEW,ESTABLISHED,RELATED -j ACCEPT
}

function add_exceptions_icmp {
  IN_CH=$1
  OUT_CH=$2
  shift 2
  EXP=$@

  create_chain ${IPv4BIN} ${IN_CH}
  create_chain ${IPv4BIN} ${OUT_CH}

  # remove same rule if exists (just to avoid duplicates)
  ${IPv4BIN} -w ${LOCKWAITTIME} -D ${IN_CH} -p icmp --icmp-type 0 -s $@ -m state --state ESTABLISHED,RELATED -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -D ${OUT_CH} -p icmp --icmp-type 8 -d $@ -m state --state NEW,ESTABLISHED,RELATED -j ACCEPT

  #add new rule
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${IN_CH} -p icmp --icmp-type 0 -s $@ -m state --state ESTABLISHED,RELATED -j ACCEPT
  ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_CH} -p icmp --icmp-type 8 -d $@ -m state --state NEW,ESTABLISHED,RELATED -j ACCEPT
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
      add_exceptions ${IPv4BIN} ${IN_IVPN_IF} ${OUT_IVPN_IF} $@

    elif [[ $1 = "-remove_exceptions" ]]; then

      shift
      remove_exceptions ${IPv4BIN} ${IN_IVPN_IF} ${OUT_IVPN_IF} $@

    elif [[ $1 = "-add_exceptions_static" ]]; then

      shift
      add_exceptions ${IPv4BIN} ${IN_IVPN_STAT_EXP} ${OUT_IVPN_STAT_EXP} $@

    elif [[ $1 = "-remove_exceptions_static" ]]; then

      shift
      remove_exceptions ${IPv4BIN} ${IN_IVPN_STAT_EXP} ${OUT_IVPN_STAT_EXP} $@

    # User exceptions
    elif [[ $1 = "-set_user_exceptions_static" ]]; then

      shift
      clean_chain ${IPv4BIN} ${IN_IVPN_STAT_USER_EXP}
      clean_chain ${IPv4BIN} ${OUT_IVPN_STAT_USER_EXP}

      [ -z "$@" ] && return
      add_exceptions ${IPv4BIN} ${IN_IVPN_STAT_USER_EXP} ${OUT_IVPN_STAT_USER_EXP} $@

    elif [[ $1 = "-set_user_exceptions_static_ipv6" ]]; then

      if [ -f /proc/net/if_inet6 ]; then
        shift
        clean_chain ${IPv6BIN} ${IN_IVPN_STAT_USER_EXP}
        clean_chain ${IPv6BIN} ${OUT_IVPN_STAT_USER_EXP}

        [ -z "$@" ] && return
        add_exceptions ${IPv6BIN} ${IN_IVPN_STAT_USER_EXP} ${OUT_IVPN_STAT_USER_EXP} $@
      fi
    
    # DNS rules
    elif [[ $1 = "-set_dns" ]]; then

      shift
      clean_chain ${IPv4BIN} ${OUT_IVPN_DNS}

      if [[ -z "$@" ]] ; then
        # block DNS
        ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p udp --dport 53 -j DROP
        ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} -p tcp --dport 53 -j DROP
      else
        # block everything except defined address
        ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} ! -d $@ -p udp --dport 53 -j DROP
        ${IPv4BIN} -w ${LOCKWAITTIME} -A ${OUT_IVPN_DNS} ! -d $@ -p tcp --dport 53 -j DROP
      fi

    # icmp exceptions
    elif [[ $1 = "-add_exceptions_icmp" ]]; then

      shift
      add_exceptions_icmp ${IN_IVPN_ICMP_EXP} ${OUT_IVPN_ICMP_EXP} $@

    elif [[ $1 = "-remove_exceptions_icmp" ]]; then

      shift
      remove_exceptions_icmp ${IN_IVPN_ICMP_EXP} ${OUT_IVPN_ICMP_EXP} $@

    elif [[ $1 = "-connected" ]]; then
        IFACE=$2
        #SRC_ADDR=$3
        SRC_PORT=$4
        DST_ADDR=$5
        DST_PORT=$6
        PROTOCOL=$7

        # allow all communication trough vpn interface
        client_connected ${IFACE}

        # allow communication with host only srcPort <=> host.dstsPort
        add_direction_exception ${IN_IVPN_IF} ${OUT_IVPN_IF} ${SRC_PORT} ${DST_ADDR} ${DST_PORT} ${PROTOCOL}
    elif [[ $1 = "-disconnected" ]]; then
        shift
        client_disconnected
    else
        echo "Unknown command"
        return 2
    fi
}

main $@
