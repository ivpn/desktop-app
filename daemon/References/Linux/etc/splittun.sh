#!/bin/bash

#
#  Script to control the Split-Tunneling functionality for Linux.
#  It is a part of Daemon for IVPN Client Desktop.
#  https://github.com/ivpn/desktop-app/daemon
#
#  Created by Stelnykovych Alexandr.
#  Copyright (c) 2021 Privatus Limited.
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

# Split-Tunneling namespace name
_namespace=ivpnst
# Virtual interfaces to link connectivity between main and ST namespace
_link_out=ivpnstout
_link_in=ivpnstin
# IP addresses and mask for virtual interfaces
_link_out_ipv4=10.17.5.1
_link_in_ipv4=10.17.5.2
_link_mask_bits=24

# DNS IP for ST namespace if it was not defined from command line argument
_the_fallback_dns=1.1.1.1

# Paths to standard binaries
_bin_ip=/sbin/ip
_bin_runuser=/usr/sbin/runuser
_bin_sudo=/usr/bin/sudo
_bin_iptables=/sbin/iptables
_bin_awk=/usr/bin/awk
_bin_dirname=/usr/bin/dirname
_bin_pwd=/usr/bin/pwd

# Routing tabel configuration for packets coming from Split-Tunneling environment
_routing_table_name=ivpnstrt
_routing_table_weight=17            # Anything from 1 to 252

# Additional parameters
_iptables_locktime=2

# Info: The 'mark' value for packets coming from the Split-Tunneling environment.
# Using here value 0xca6c. It is the same as WireGuard marking packets which were processed.
# That allows us not to be aware of changes in the routing policy database on each new connection of WireGuard.
# Otherwise, it would be necessary to call this script with parameter "update" each time of new WG connection.
# Extended description:
# The WG is updating its routing policy rule (ip rule) on every new connection:
#   32761:	not from all fwmark 0xca6c lookup 51820
# The problem is that each time this rule appears with the highest priority.
# So, this rule absorbs all packets which are not marked as 0xca6c
_packets_fwmark_value=0xca6c        # Anything from 1 to 2147483647

# Returns 0 in case if split tunneling enabled
function status()
{
    ${_bin_sudo} ${_bin_ip} netns exec ${_namespace} ${_bin_ip} netns identify > /dev/null 2>&1
    _ret=$?
    if [ ${_ret} == 0 ]; then
      echo "Split Tunneling: ENABLED"
      return 0
    fi
    echo "Split Tunneling: DISABLED"
    return ${_ret}
}

# Print detailed information about current configuration
function info()
{
    status
    echo

    echo "[*] Interfaces (${_bin_ip} link):"
    ${_bin_ip} link
    echo

    _val=`cat /proc/sys/net/ipv4/ip_forward`
    echo "[*] /proc/sys/net/ipv4/ip_forward: ${_val}"
    echo

    echo "[*] /proc/sys/net/ipv4/conf/*/rp_filter:"
    for i in /proc/sys/net/ipv4/conf/*/rp_filter; do
        _val=`cat $i`
        echo $i: ${_val}
    done
    echo

    echo "[*] Default rules (${_bin_iptables} -S):"
    ${_bin_iptables} -S
    echo

    echo "[*] Namespaces (${_bin_ip} netns list):"
    ${_bin_ip} netns list
    echo

    echo "[*] File '/etc/iproute2/rt_tables':"
    cat /etc/iproute2/rt_tables
    echo

    echo "[*] iptables -t mangle -S:"
    ${_bin_iptables} -t mangle -S
    echo

    echo "[*] iptables -t nat -S:"
    ${_bin_iptables} -t nat -S
    echo

    echo "[*] ip rule:"
    ${_bin_ip} rule
    echo

    echo "[*] ip route show table ${_routing_table_name}"
    ${_bin_ip} route show table ${_routing_table_name}
    echo
}

function backupUserConfig()
{
    # Directory where current script is located
    _script_dir="$( cd "$(${_bin_dirname} "$0")" >/dev/null 2>&1 ; ${_bin_pwd} -P )"

    if [ -z "${_script_dir}" ]; then
        return 1
    fi

    _tempDir="${_script_dir}/splittun_tmp"
    if [ -d "${_script_dir}/../mutable" ]; then
        _tempDir="${_script_dir}/../mutable/splittun_tmp"
    fi

    if [ -f ${_tempDir}/ip_forward ]; then
      # the backup is already exists
      return
    fi

    mkdir -p ${_tempDir}
    cp /proc/sys/net/ipv4/ip_forward ${_tempDir}/ip_forward
}

function restoreUserConfig()
{
    # Directory where current script is located
    _script_dir="$( cd "$(${_bin_dirname} "$0")" >/dev/null 2>&1 ; ${_bin_pwd} -P )"

    if [ -z "${_script_dir}" ]; then
        return 1
    fi

    _tempDir="${_script_dir}/splittun_tmp"
    if [ -d "${_script_dir}/../mutable" ]; then
        _tempDir="${_script_dir}/../mutable/splittun_tmp"
    fi

    if [ ! -d "${_tempDir}" ]; then
        return 2
    fi

    if [ -f "${_tempDir}/ip_forward" ]; then
        cp ${_tempDir}/ip_forward /proc/sys/net/ipv4/ip_forward
        rm ${_tempDir}/ip_forward
    fi

    rm -fr ${_tempDir}

}

# Initialize split tunneling
function init()
{
    # default interface name
    _def_interface_name=$1
    # default gateway IP
    _def_gateway=$2
    # default DNS IP
    _def_dns=$3

    # Ensure the input parameters not empty
    if [ -z ${_def_interface_name} ]; then
        echo "[i] Default network interface is not defined. Trying to determine it automatically..."
        _def_interface_name=$(${_bin_ip} route | ${_bin_awk} '/default/ { print $5 }')
        echo "[+] Default network interface: '${_def_interface_name}'"
    fi
    if [ -z ${_def_gateway} ]; then
        echo "[i] Default gateway is not defined. Trying to determine it automatically..."
        _def_gateway=$(${_bin_ip} route | ${_bin_awk} '/default/ { print $3 }')
        echo "[+] Default gateway: '${_def_gateway}'"
    fi
    if [ -z ${_def_interface_name} ]; then
        echo "[!] Default network interface is not defined."
        return 2
    fi
    if [ -z ${_def_gateway} ]; then
        echo "[!] Default gateway is not defined."
        return 3
    fi

    ##############################################
    # Ensure previous configuration erased
    ##############################################
    clean $@  > /dev/null 2>&1

    ##############################################
    # Create namespace
    ##############################################
    ${_bin_ip} netns add ${_namespace}

    ##############################################
    # Create a veth virtual-interface pair and initialize interfaces
    ##############################################

    # Create a veth virtual-interface pair
    # ${_link_out} - live in default namespace
    # ${_link_in} - live in namespace for splittuneling (${_namespace})
    ${_bin_ip} link add ${_link_out} type veth peer name ${_link_in} netns ${_namespace}

    # Assign an address to each interface
    ${_bin_ip} addr add ${_link_out_ipv4}/${_link_mask_bits} dev ${_link_out}
    ${_bin_ip} netns exec ${_namespace} ${_bin_ip} addr add ${_link_in_ipv4}/${_link_mask_bits} dev ${_link_in}

    # UP the interfaces
    ${_bin_ip} link set ${_link_out} up
    ${_bin_ip} netns exec ${_namespace} ${_bin_ip} link set 'lo' up
    ${_bin_ip} netns exec ${_namespace} ${_bin_ip} link set ${_link_in} up

    # configure routing in namespace via default interface ${_link_out_ipv4}
    ${_bin_ip} netns exec ${_namespace} ${_bin_ip} route add default via ${_link_out_ipv4} dev ${_link_in}

    ##############################################
    # Setup IP forwarding
    ##############################################

    # Reverse path filtering
    echo 2 > /proc/sys/net/ipv4/conf/${_link_out}/rp_filter

    # backup the original value of "/proc/sys/net/ipv4/ip_forward"
    backupUserConfig
    # Activate router functions (/proc/sys/net/ipv4/ip_forward
    # Has side effects: e.g. net.ipv4.conf.all.accept_redirects=0,secure_redirects=1
    # Resets ipv4 kernel interface 'all' config values to default for HOST or ROUTER
    # https://www.kernel.org/doc/Documentation/networking/ip-sysctl.txt
    echo 1 > /proc/sys/net/ipv4/ip_forward

    # Enable masquerading. Force the packets to exit through default interface (eg. eth0, enp0s3 ...) with NAT
    ${_bin_iptables} -w ${_iptables_locktime} -t nat -A POSTROUTING -s ${_link_out_ipv4}/${_link_mask_bits} -o ${_def_interface_name} -j MASQUERADE
    ${_bin_iptables} -w ${_iptables_locktime} -A FORWARD -i ${_def_interface_name} -o ${_link_out} -j ACCEPT
    ${_bin_iptables} -w ${_iptables_locktime} -A FORWARD -o ${_def_interface_name} -i ${_link_out} -j ACCEPT
    # OR
    #${_bin_iptables} -t nat -A POSTROUTING -s ${_link_in_ipv4}/${_link_mask_bits} -j SNAT --to-source 192.168.1.167

    ##############################################
    # Setup DNS for 'splitted' namespace
    ##############################################

    # set DNS for the namespace
    # TODO: set correct DNS
    if [ -z ${_def_dns} ]; then
        _def_dns=${_the_fallback_dns}
        echo "[!] DNS IP not defined. Using default '${_def_dns}'"
    fi
    mkdir -p /etc/netns/${_namespace} && echo "nameserver ${_def_dns}" > /etc/netns/${_namespace}/resolv.conf
    # (optional) copy hosts file
    mkdir -p /etc/netns/${_namespace} && cp /etc/hosts /etc/netns/${_namespace}/hosts

    ##############################################
    # When 'firewalld' is in use - add specific firewalld rules
    ##############################################
    firewall-cmd --state > /dev/null 2>&1
    if [ $? == 0 ]; then
        firewall-cmd --zone=trusted --add-interface=${_link_out}
    fi

    ##############################################
    # INFO: Some Linux configurations blocks the ping packets from the namespace
    ##############################################
    # Example how to allow ping:
    # ip netns exec sysctl net.ipv4.ping_group_range="0 2147483647"

    ##############################################
    # Use different routing for packets coming from the 'splitted' namespaces
    ##############################################
    # create routing table for splitunneling
    echo "${_routing_table_weight}      ${_routing_table_name}" >> /etc/iproute2/rt_tables
    # Set default gateway for the 'splittun' table
    ${_bin_ip} route add default via ${_def_gateway} table ${_routing_table_name}

    # add mark to all packets coming from namespace (from ${_link_out})
    ${_bin_iptables} -w ${_iptables_locktime} -t mangle -A PREROUTING -i ${_link_out} -j MARK --set-mark ${_packets_fwmark_value}
    # Packets with mark will use splittun table
    ${_bin_ip} rule add fwmark ${_packets_fwmark_value} table ${_routing_table_name}

    # do required operations for compatibility with WireGuard
    applyWireGuardCompatibility
}

function applyWireGuardCompatibility()
{
    # Check iw WG connected
    _ret=$(${_bin_ip} rule list not from all fwmark 0xca6c) # WG rule
    if [ ! -z "${_ret}" ]; then
        # Only for WireGuard connection:
        # Ensure rule 'from all fwmark 0xca6c lookup ivpnstrt' has higher priority
        ${_bin_ip} rule del from all lookup main suppress_prefixlength 0 > /dev/null 2>&1
        ${_bin_ip} rule add from all lookup main suppress_prefixlength 0
    fi
}

# Update (restore) routing policy rule
# It can be useful if there were changes in the routing policy database (e.g. new WireGuard connection established)
function update()
{
    # Check if split tunneling enabled
    status > /dev/null 2>&1
    if [ $? != 0 ]; then
        echo "ERROR: split tunneling DISABLED."
        exit 1
    fi

    # Remove our routing policy rule
    # and add the same rule again rule will be added with higher priority (smaller weight) than other rules
    ${_bin_ip} rule del fwmark ${_packets_fwmark_value} table ${_routing_table_name}
    ${_bin_ip} rule add fwmark ${_packets_fwmark_value} table ${_routing_table_name}

    # do required operations for compatibility with WireGuard
    applyWireGuardCompatibility
}

# UnInitialize split tunneling
function clean()
{
    # default interface name
    _def_interface_name=$1

    # Ensure the input parameters not empty
    if [ -z ${_def_interface_name} ]; then
        echo "[i] Default network interface is not defined. Trying to determine it automatically..."
        _def_interface_name=$(${_bin_ip} route | ${_bin_awk} '/default/ { print $5 }')
        echo "[+] Default network interface: '${_def_interface_name}'"
    fi

    ##############################################
    # Delete namespace
    ##############################################

    # The pair of 'veth' interfaces (${_link_in},  ${_link_out}) will be deleted automatically
    ${_bin_ip} netns del ${_namespace}
    #just to ensure that everything cleaned up
    ${_bin_ip} link delete ${_link_out} > /dev/null 2>&1

    ##############################################
    # IP forwarding
    ##############################################
    # restore the original value of "/proc/sys/net/ipv4/ip_forward"
    restoreUserConfig

    # Erase forward rules
    ${_bin_iptables} -w ${_iptables_locktime} -t nat -D POSTROUTING -s ${_link_out_ipv4}/${_link_mask_bits} -o ${_def_interface_name} -j MASQUERADE
    ${_bin_iptables} -w ${_iptables_locktime} -D FORWARD -i ${_def_interface_name} -o ${_link_out} -j ACCEPT
    ${_bin_iptables} -w ${_iptables_locktime} -D FORWARD -o ${_def_interface_name} -i ${_link_out} -j ACCEPT

    ##############################################
    # Remove namespace leftovers (DNS configuration; hosts ...)
    ##############################################
    rm -fr /etc/netns/${_namespace}

    ##############################################
    # When 'firewalld' is in use - add specific firewalld rules
    ##############################################
    firewall-cmd --state > /dev/null 2>&1
    if [ $? == 0 ]; then
        firewall-cmd --zone=trusted --remove-interface=${_link_out}
    fi

    ##############################################
    # Remove routing for packets coming from the 'splitted' namespaces
    ##############################################

    # remove rule: packets with mark will use splittun table
    # ${_bin_ip} rule del from all lookup main suppress_prefixlength 0 #do not remove this rulle (it can be used by active WireGuard connection)
    ${_bin_ip} rule del fwmark ${_packets_fwmark_value} table ${_routing_table_name}
    # remove: add mark to all packets coming from namespace (from ${_link_out})
    ${_bin_iptables} -w ${_iptables_locktime} -t mangle -D PREROUTING -i ${_link_out} -j MARK --set-mark ${_packets_fwmark_value}

    # remove: splittun table has a default gateway to the default interface
    ${_bin_ip} route del default table ${_routing_table_name}
    # remove routing table for splitunneling
    sed -i "/${_routing_table_name}\s*$/d" /etc/iproute2/rt_tables
}

# Execute command n split tunneling environment
# (note: the '-init' command must be started before)
function execute()
{
    _user="$1"
    _app="$2"

    #echo "App: ${_app}"
    #echo "User: ${_user}"

    if [ -z "${_app}" ]; then
        echo "[!] ERROR: Application not defined"
        exit 1
    fi

    # Check if split tunneling enabled
    status > /dev/null 2>&1
    if [ $? != 0 ]; then
        echo "ERROR: split tunneling DISABLED. Please call '-init' command first"
        exit 1
    fi

    # Obtaining information about user running the script
    # (script must be executed with 'sudo', but we should get real user)
    if [ -z ${_user} ]; then
        _user="${SUDO_USER:-$USER}"
    fi

    echo "[+] Starting '${_app}' for a user '${_user}'..."
    ${_bin_ip} netns exec ${_namespace} ${_bin_runuser} -u ${_user} -- ${_app}
}

if [[ $1 = "start" ]] ; then
    # TODO: implement return value (0 when succes)
    _interface_name=""
    _gateway_ip=""
    _dns_ip=""
    shift

    while getopts ":i:g:d:" opt; do
        case $opt in
            i) _interface_name="$OPTARG"   ;;
            g) _gateway_ip="$OPTARG"    ;;
            d) _dns_ip="$OPTARG"    ;;
        esac
    done

    init  ${_interface_name} ${_gateway_ip} ${_dns_ip}

elif [[ $1 = "run" ]] ; then
    # TODO: implement return value (0 when succes)
    _command=""
    _user=""

    shift
    while getopts ":u:" opt; do
        case $opt in
            u) _user="$OPTARG"   ;;
        esac
    done

    if [ ! -z ${_user} ]; then
        shift
        shift
    fi
    _command=$@
    execute "${_user}" "${_command}"
elif [[ $1 = "stop" ]]; then
    # TODO: implement return value (0 when succes)
    _interface_name=""
    shift

    while getopts ":i:" opt; do
        case $opt in
            i) _interface_name="$OPTARG"   ;;
        esac
    done
    clean ${_interface_name}
elif [[ $1 = "update" ]] || [[ $1 = "restart" ]] ; then
    # TODO: implement return value (0 when succes)
    shift
    update $@
elif [[ $1 = "status" ]] ; then
    shift
    status $@
elif [[ $1 = "info" ]] ; then
    shift
    info $@
elif [[ $1 = "manual" ]] ; then
    _FUNCNAME=$2
    shift
    shift
    echo "Running manual command: ${_FUNCNAME}($@) "
    ${_FUNCNAME} $@
else
    echo "Script to control the Split-Tunneling functionality for Linux."
    echo "It is a part of Daemon for IVPN Client Desktop."
    echo "https://github.com/ivpn/desktop-app/daemon"
    echo "Created by Stelnykovych Alexandr."
    echo "Copyright (c) 2021 Privatus Limited."
    echo ""
    echo "Usage:"
    echo "Note! The script have to be started under privilaged user (sudo $0 ...)"
    echo "    $0 <command> [parameters]"
    echo "Parameters:"
    echo "    start [-i <interface_name>] [-g <gateway_ip>] [-d <dns>]"
    echo "        Initialize split-tunneling functionality"
    echo "        - interface_name - (optional) name of network interface to be used for ST environment"
    echo "        - gateway_ip     - (optional) gateway IP to be used for ST environment"
    echo "        - dns            - (optional) DNS IP to be used for ST environment"
    echo "    stop [-i <interface_name>]"
    echo "        Uninitialize split-tunneling functionality"
    echo "        - interface_name - (optional) name of network interface which was previously used for '-init' command"
    echo "    run [-u <username>] <command>"
    echo "        Start commands in split-tunneling environment"
    echo "        - command        - the command or path to binary to be executed"
    echo "        - username       - (optional) the account under which the command have to be executed"
    #echo "    update"
    #echo "        Update (restore) routing policy rule"
    #echo "        It can be useful if there were changes in the routing policy database (e.g. new WireGuard connection established)"
    echo "    status"
    echo "        Check split-tunneling status"
    echo "Examples:"
    echo "    Initialize split-tunneling functionality:"
    echo "        $0 start"
    echo "        $0 start -i wlp3s0 -g 192.168.1.1 -d 1.1.1.1"
    echo "    Start commands in split-tunneling environment:"
    echo "        $0 run firefox"
    echo "        $0 run '/usr/bin/firefox'"
    echo "        $0 run 'ping 8.8.8.8'"
    echo "    Uninitialize split-tunneling functionality:"
    echo "        $0 stop"
    echo "        $0 stop -i wlp3s0"
    echo "    Check split-tunneling status:"
    echo "        $0 status"
fi
