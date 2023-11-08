#!/bin/bash

#
#  Script to control the Split-Tunneling functionality for Linux.
#  It is a part of Daemon for IVPN Client Desktop.
#  https://github.com/ivpn/desktop-app/daemon
#
#  Created by Stelnykovych Alexandr.
#  Copyright (c) 2023 IVPN Limited.
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

# Split Tunneling cgroup parameters
_cgroup_name=ivpn-exclude
_cgroup_classid=0x4956504e      # Anything from 0x00000001 to 0xFFFFFFFF
_cgroup_folder=/sys/fs/cgroup/net_cls/${_cgroup_name}

# Routing tabel configuration for packets coming from Split-Tunneling environment
_routing_table_name=ivpn-exclude-tbl
_routing_table_weight=17            # Anything from 1 to 252

# iptables chains
POSTROUTING_mangle="IVPN_ST_POSTROUTING -t mangle"
OUTPUT_mangle="IVPN_ST_OUTPUT -t mangle"
PREROUTING_mangle="IVPN_ST_PREROUTING -t mangle"
POSTROUTING_nat="IVPN_ST_POSTROUTING -t nat"
OUTPUT="IVPN_ST_OUTPUT"
INPUT="IVPN_ST_INPUT"

# Additional parameters
_iptables_locktime=2

# Backup folder name.
# This folder contains temporary data to be able to clean everything correctly 
_backup_folder_name=ivpn-exclude-tmp
_mutable_folder_default=/etc/opt/ivpn/mutable   # default location of 'mutable' folder
_mutable_folder_fallback=/opt/ivpn/mutable      # alternate location of 'mutable' folder (needed for backward compatibility and snap environment)

# Info: The 'mark' value for packets coming from the Split-Tunneling environment.
# Using here value 0xca6c. It is the same as WireGuard marking packets which were processed.
# That allows us not to be aware of changes in the routing policy database on each new connection of WireGuard.
# Extended description:
# The WG is updating its routing policy rule (ip rule) on every new connection:
#   32761:	not from all fwmark 0xca6c lookup 51820
# The problem is that each time this rule appears with the highest priority.
# So, this rule absorbs all packets which are not marked as 0xca6c
_packets_fwmark_value=0xca6c        # Anything from 1 to 2147483647

# Paths to standard binaries
_bin_iptables=iptables
_bin_ip6tables=ip6tables
_bin_runuser=runuser
_bin_ip=ip
_bin_awk=awk
_bin_grep=grep
_bin_dirname=dirname
_bin_sed=sed

#
#Variables vill be initialized later:
#
_def_interface_name=""
_def_interface_nameIPv6=""
_def_gateway=""
_def_gatewayIPv6=""
# When inversed - only apps added to ST will use VPN connection, 
# all other apps will use direct unencrypted connection
_is_inversed=0 
# Applicable for Inverse mode only: block/allow communication for 'splitted' apps 
#   -   "1" means the communication for splitted apps will be blocked (for example, when VPN not connected)
#   -   "0" means the communication for splitted apps will not be blocked
_is_inversed_blocked=0
_is_inversed_blocked_ipv6=0

vercomp () {
    if [[ $1 == $2 ]]
    then
        return 0
    fi
    local IFS=.
    local i ver1=($1) ver2=($2)
    # fill empty fields in ver1 with zeros
    for ((i=${#ver1[@]}; i<${#ver2[@]}; i++))
    do
        ver1[i]=0
    done
    for ((i=0; i<${#ver1[@]}; i++))
    do
        if [[ -z ${ver2[i]} ]]
        then
            # fill empty fields in ver2 with zeros
            ver2[i]=0
        fi
        if ((10#${ver1[i]} > 10#${ver2[i]}))
        then
            return 1
        fi
        if ((10#${ver1[i]} < 10#${ver2[i]}))
        then
            return 2
        fi
    done
    return 0
}

function test()
{
    # TODO: the real mount path have to be taken from /proc/mounts
    # It has format: <devtype> <mount path> <fstype> <options>
    # Example: cgroup /sys/fs/cgroup/net_cls,net_prio cgroup rw,nosuid,nodev,noexec,relatime,net_cls,net_prio 0 0
    # We have to check <fstype>=='cgroup'; <options> contain 'net_cls'

    if [ ! -d /sys/fs/cgroup/net_cls ]; then
        echo "Creating '/sys/fs/cgroup/net_cls' folder ..."
        if ! mkdir -p /sys/fs/cgroup/net_cls;   then 
            echo "ERROR: Failed to create CGROUP folder Not Found (/sys/fs/cgroup/net_cls)" 1>&2
            return 1; 
        fi
    fi
    if ! mount | grep "/sys/fs/cgroup/net_cls" &>/dev/null ; then
        echo "Mounting CGROUP subsystem '/sys/fs/cgroup/net_cls'..."
        if ! mount -t cgroup -o net_cls net_cls /sys/fs/cgroup/net_cls ; then
            echo "ERROR: Failed to mount CGROUP subsystem (net_cls)" 1>&2
            return 2; 
        fi
    fi

    if ! command -v ${_bin_iptables} &>/dev/null ;   then echo "ERROR: Binary Not Found (${_bin_iptables})" 1>&2; return 1; fi
    if ! command -v ${_bin_ip} &>/dev/null ;         then echo "ERROR: Binary Not Found (${_bin_ip})" 1>&2; return 1; fi    
    if ! command -v ${_bin_grep} &>/dev/null ;       then echo "ERROR: Binary Not Found (${_bin_grep})" 1>&2; return 1; fi
    if ! command -v ${_bin_dirname} &>/dev/null ;    then echo "ERROR: Binary Not Found (${_bin_dirname})" 1>&2; return 1; fi
    if ! command -v ${_bin_sed} &>/dev/null ;        then echo "ERROR: Binary Not Found (${_bin_sed})" 1>&2; return 1; fi

    if ! command -v ${_bin_ip6tables} &>/dev/null ;  then echo "WARNING: Binary Not Found (${_bin_ip6tables})" 1>&2; fi
    if ! command -v ${_bin_awk} &>/dev/null ;        then echo "WARNING: Binary Not Found (${_bin_awk})" 1>&2; fi
    if ! command -v ${_bin_runuser} &>/dev/null ;    then echo "WARNING: Binary Not Found (${_bin_runuser})" 1>&2; fi


    # ###
    # -= Compare minimum required iptables version for Inverse Split Tunneling =-
    # ###
    local min_required_ver="1.8.7"
    
    local iptables_version=$(${_bin_iptables} --version 2>&1 | ${_bin_awk} '{print $2}') # Get iptables version
    local iptables_version=${iptables_version#v} # remove "v" prefix, if exists
    vercomp $iptables_version $min_required_ver # compare versions
    if [[ $? -eq 2 ]]; then 
        # NOTE! Do not chnage the message below. It is used by daemon to detect the error.
        echo "Warning: Inverse mode for IVPN Split Tunnel functionality is not applicable. The minimum required version of 'iptables' is $min_required_ver, while your version is $iptables_version."
    fi
    
    return 0
}

function detectDefRouteVars() 
{
    if [ -z ${_def_gateway} ] || [ -z ${_def_interface_name} ]; then
        # Get both default gateway IP and interface name in one command
        read -r _def_gateway _def_interface_name <<< $(${_bin_ip} route | awk '/default/  {print $3, $5}')
        echo "[+] Detected default route     : gateway='${_def_gateway}' interface='${_def_interface_name}'"
    fi

    if [ -z ${_def_gatewayIPv6} ] || [ -z ${_def_interface_nameIPv6} ]; then
        if [ -f /proc/net/if_inet6 ]; then
            read -r _def_gatewayIPv6 _def_interface_nameIPv6 <<< $(${_bin_ip} -6 route | awk '/default/  {print $3, $5}')
            echo "[+] Detected default IPv6 route: gateway='${_def_gatewayIPv6}' interface='${_def_interface_nameIPv6}'"
        fi
    fi
}

function init_iptables() 
{    
    local bin_iptables=$1
    local def_inf_name=$2
    local inverse_block=$3

    # in Inverse mode - we are inversing firewall rules:
    # 'splitted' apps use only VPN connection, all the rest apps use default connection settings (bypassing VPN)
    local inverseOption=""
    if [ ${_is_inversed} -eq 1 ]; then
        inverseOption=" ! "
    fi
    ##############################################
    # Firewall rules for packets coming from cgroup
    ##############################################    
    # NOTE! All rules here added with "-I" parameter. "-I" means insert rule at the top.
    # So, the original rules sequence will be the reverse sequence to the list below.

    ${bin_iptables} -w ${LOCKWAITTIME} -N ${POSTROUTING_mangle}
    ${bin_iptables} -w ${LOCKWAITTIME} -N ${OUTPUT_mangle}
    ${bin_iptables} -w ${LOCKWAITTIME} -N ${PREROUTING_mangle}
    ${bin_iptables} -w ${LOCKWAITTIME} -N ${POSTROUTING_nat}
    ${bin_iptables} -w ${LOCKWAITTIME} -N ${OUTPUT}
    ${bin_iptables} -w ${LOCKWAITTIME} -N ${INPUT}

    # Save packets mark (to be able to restore mark for incoming packets of the same connection)
    ${bin_iptables} -w ${_iptables_locktime} -I ${POSTROUTING_mangle} -j CONNMARK --save-mark    
    # Change the source IP address of packets to the IP address of the interface they're going out on
    # Do this only if default interface is defined (for example: IPv6 interface may be empty when IPv6 not configured on the system)
    if [ ! -z ${def_inf_name} ]; then
        ${bin_iptables} -w ${_iptables_locktime} -I ${POSTROUTING_nat} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -o ${def_inf_name} -j MASQUERADE
    fi
    # Add mark on packets of classid ${_cgroup_classid}
    ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT_mangle} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -j MARK --set-mark ${_packets_fwmark_value}
    # Important! Process DNS request before setting mark rule (DNS request should not be marked)
    ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT_mangle} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -p tcp --dport 53 -j RETURN
    ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT_mangle} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -p udp --dport 53 -j RETURN

    # Allow packets from/to cgroup (bypass IVPN firewall)
    if [ ! -z ${def_inf_name} ]; then
        ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -j ACCEPT
        ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -j ACCEPT   # this rule is not effective, so we use 'mark' (see the next rule)
        ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -m mark --mark ${_packets_fwmark_value} -j ACCEPT
    else
        # If local interface not defined - block all packets from/to cgroup
        # (for example: IPv6 interface may be empty when IPv6 not configured on the system)
        ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -j DROP
        ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -m cgroup ${inverseOption} --cgroup ${_cgroup_classid} -j DROP   # this rule is not effective, so we use 'mark' (see the next rule)
        ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -m mark --mark ${_packets_fwmark_value} -j DROP
    fi
    
    # Inverse mode: only 'splitted' apps use only VPN connection
    if [ ${_is_inversed} -eq 1 ]; then
        # Important! Process DNS request first: do not drop DNS requests
        ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -p tcp --dport 53 -j RETURN
        ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -p udp --dport 53 -j RETURN
        
        # Allow or block communication for 'splitted' apps in inverse mode
        # E.g.: If we want to block 'splitted' apps when VPN not connected -  'inverse_block' must be '1'
        if [ ${inverse_block} -eq 1 ]; then
            ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -m cgroup --cgroup ${_cgroup_classid} -j DROP
            ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -m cgroup --cgroup ${_cgroup_classid} -j DROP
        fi 
    fi 

    # Just ensure that packets from/to localhost will not be blocked            
    ${bin_iptables} -w ${_iptables_locktime} -I ${OUTPUT} -o lo -j ACCEPT
    ${bin_iptables} -w ${_iptables_locktime} -I ${INPUT}  -i lo -j ACCEPT
    
    # Restore packets mark for incoming packets
    ${bin_iptables} -w ${_iptables_locktime} -I ${PREROUTING_mangle} -j CONNMARK --restore-mark

    ${bin_iptables} -w ${_iptables_locktime} -I POSTROUTING -t mangle  -j ${POSTROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -I OUTPUT -t mangle  -j ${OUTPUT_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -I PREROUTING -t mangle  -j ${PREROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -I POSTROUTING -t nat  -j ${POSTROUTING_nat}
    ${bin_iptables} -w ${_iptables_locktime} -I OUTPUT -j ${OUTPUT}
    ${bin_iptables} -w ${_iptables_locktime} -I INPUT -j ${INPUT}
}

function clear_iptables() 
{   
    local bin_iptables=$1    
    ##############################################
    # Remove firewall rules
    ##############################################
    # '-D' Delete matching rule from chain    
    ${bin_iptables} -w ${_iptables_locktime} -D POSTROUTING -t mangle  -j ${POSTROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -D OUTPUT -t mangle  -j ${OUTPUT_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -D PREROUTING -t mangle  -j ${PREROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -D POSTROUTING -t nat  -j ${POSTROUTING_nat}
    ${bin_iptables} -w ${_iptables_locktime} -D OUTPUT -j ${OUTPUT}
    ${bin_iptables} -w ${_iptables_locktime} -D INPUT -j ${INPUT}
    
    # '-F' Delete all rules in  chain 
    ${bin_iptables} -w ${_iptables_locktime} -F ${POSTROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -F ${OUTPUT_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -F ${PREROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -F ${POSTROUTING_nat}
    ${bin_iptables} -w ${_iptables_locktime} -F ${OUTPUT}
    ${bin_iptables} -w ${_iptables_locktime} -F ${INPUT}

    # '-X' Delete a user-defined chains
    ${bin_iptables} -w ${_iptables_locktime} -X ${POSTROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -X ${OUTPUT_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -X ${PREROUTING_mangle}
    ${bin_iptables} -w ${_iptables_locktime} -X ${POSTROUTING_nat}
    ${bin_iptables} -w ${_iptables_locktime} -X ${OUTPUT}
    ${bin_iptables} -w ${_iptables_locktime} -X ${INPUT}
}

function init()
{
    if [ -z ${_def_interface_name} ]; then
        echo "Default network interface is not defined. Please, check internet connectivity." 1>&2
        return 2
    fi
    if [ -z ${_def_gateway} ]; then
        echo "Default gateway is not defined. Please, check internet connectivity." 1>&2
        return 3
    fi

    if [ -f /proc/net/if_inet6 ]; then 
        if [ -z ${_def_gatewayIPv6} ]; then
            echo "Warning: Default IPv6 gateway is not defined." 1>&2
        fi
        if [ -z ${_def_interface_nameIPv6} ]; then
            echo "Warning: Default IPv6 interface is not defined." 1>&2
        fi
    fi

    ##############################################
    # Ensure previous configuration erased
    ##############################################
    clean $@  > /dev/null 2>&1

    set -e

    ##############################################
    # Backup some parameters for restore function (_def_interface_name, /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter )
    ##############################################
    backup
    # Set required reverse path filtering parameter
    if [ -f /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter ]; then
        echo 2 > /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter
    fi

    ##############################################
    # Create cgroup
    ##############################################
    if [ ! -d ${_cgroup_folder} ]; then
        mkdir -p ${_cgroup_folder}
        echo ${_cgroup_classid} > ${_cgroup_folder}/net_cls.classid
    fi
    
    ##############################################
    # Firewall rules for packets coming from cgroup
    ##############################################       
    init_iptables  ${_bin_iptables} ${_def_interface_name} ${_is_inversed_blocked}
    if [ -f /proc/net/if_inet6 ]; then
        block=0
        if [ ! ${_is_inversed_blocked} -eq 0 ] || [ ! ${_is_inversed_blocked_ipv6} -eq 0 ]; then 
            block=1
        fi
        init_iptables  ${_bin_ip6tables} ${_def_interface_nameIPv6} ${block}
    fi

    ##############################################
    # Initialize routing table for packets coming from cgroup   
    ##############################################    
    if ! ${_bin_grep} -E "^[0-9]+\s+${_routing_table_name}\s*$" /etc/iproute2/rt_tables &>/dev/null ; then
        # initialize new routing table
        mkdir -p /etc/iproute2
        echo "${_routing_table_weight}      ${_routing_table_name}" >> /etc/iproute2/rt_tables

        # Packets with mark will use splittun table
        ${_bin_ip} rule add fwmark ${_packets_fwmark_value} table ${_routing_table_name}

        if [ ! -z ${_def_gatewayIPv6} ]; then
            if [ -f /proc/net/if_inet6 ]; then
                # Packets with mark will use splittun table
                ${_bin_ip} -6 rule add fwmark ${_packets_fwmark_value} table ${_routing_table_name}
            fi
        fi

        # The splittun table has a default gateway route to the default interface
        #   ${_bin_ip} route add default via ${_def_gateway} table ${_routing_table_name}  
        #   ${_bin_ip} -6 route add default via ${_def_gatewayIPv6} table ${_routing_table_name}
        updateRoutes        
    fi

    ##############################################
    # Compatibility with WireGuard rules 
    ##############################################
    # Check iw WG connected
    _ret=$(${_bin_ip} rule list not from all fwmark 0xca6c) # WG rule
    if [ ! -z "${_ret}" ]; then
        # Only for WireGuard connection:
        # Ensure rule 'rule add from all lookup main suppress_prefixlength 0' has higher priority
        #
        # This wireguard rule respects the manually configured routes in the main table. 
        # (routing decision is ignored for routes with a prefix length of 0 (it is 'default' route: 0.0.0.0/0))
        #
        # Info:
        #   wireguard adds such rules:
        #   	from all lookup main suppress_prefixlength 0
        #   	not from all fwmark 0xca6c lookup 51820

        ${_bin_ip} rule del from all lookup main suppress_prefixlength 0 > /dev/null 2>&1
        ${_bin_ip} rule add from all lookup main suppress_prefixlength 0

        if [ -f /proc/net/if_inet6 ]; then
            _ret=$(${_bin_ip} -6 rule list not from all fwmark 0xca6c) # WG rule
            if [ ! -z "${_ret}" ]; then
                ${_bin_ip} -6 rule del from all lookup main suppress_prefixlength 0 > /dev/null 2>&1
                ${_bin_ip} -6 rule add from all lookup main suppress_prefixlength 0
            fi
        fi
    fi

    set +e

    echo "IVPN Split Tunneling enabled"
}

function updateRoutes() 
{ 
    # simple check if ST enabled
    if [ ! -d ${_cgroup_folder} ]; then
        return
    fi

    # splittun table has a default gateway to the default interface
    if [ ! -z ${_def_gateway} ] && [ ! -z ${_def_interface_name} ]; then        
        ${_bin_ip} route replace default via ${_def_gateway} dev ${_def_interface_name} table ${_routing_table_name}  
    fi
    if [ -f /proc/net/if_inet6 ] && [ ! -z ${_def_gatewayIPv6} ] && [ ! -z ${_def_interface_nameIPv6} ]; then
        ${_bin_ip} -6 route replace default via ${_def_gatewayIPv6} dev ${_def_interface_nameIPv6} table ${_routing_table_name}
    fi
}

function clean()
{
    ##############################################
    # Restore parameters
    ##############################################
    # read ${_def_interface_name} from backup
    restore 

    ##############################################
    # Move all processes from the IVPN cgroup to the main cgroup
    ##############################################    
    # removeAllPids

    ##############################################
    # Remove cgroup    
    ##############################################
    # check is cgroup exists
    if [ -d ${_cgroup_folder} ]; then
        # Note: the cgroup folder will be removed only in case
        # when no active process are in that cgroup
        rmdir ${_cgroup_folder}
    fi  

    ##############################################
    # Remove firewall rules
    ##############################################
    clear_iptables ${_bin_iptables}    
    if [ -f /proc/net/if_inet6 ]; then
        clear_iptables ${_bin_ip6tables} &>/dev/null 
    fi

    ##############################################
    # Remove routing
    ##############################################
    ${_bin_ip} rule del fwmark ${_packets_fwmark_value} table ${_routing_table_name}    
    ${_bin_ip} route flush table ${_routing_table_name}
    if [ -f /proc/net/if_inet6 ]; then
        ${_bin_ip} -6 rule del fwmark ${_packets_fwmark_value} table ${_routing_table_name}    &>/dev/null 
        ${_bin_ip} -6 route flush table ${_routing_table_name}    &>/dev/null 
    fi 

    ${_bin_sed} -i "/${_routing_table_name}\s*$/d" /etc/iproute2/rt_tables   
}

function getBackupFolderPath()
{
    # default location
    if [ -w "${_mutable_folder_default}" ]; then       
        echo "${_mutable_folder_default}/${_backup_folder_name}"  # return value in stdout        
        return 0
    fi
    # fallback location
    if [ -w "${_mutable_folder_fallback}" ]; then       
        echo "${_mutable_folder_fallback}/${_backup_folder_name}"  # return value in stdout
        return 0
    fi

    echo "${_mutable_folder_default}/${_backup_folder_name}"
    return 1
}

function backup()
{
    if [ -z ${_def_interface_name} ]; then
        return 1
    fi

    local _tempDir="$( getBackupFolderPath )"
    mkdir -p ${_tempDir}

    echo ${_def_interface_name} > ${_tempDir}/def_interface
    if [ -f /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter ]; then        
        cat /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter >  ${_tempDir}/${_def_interface_name}-rp_filter
    fi
}

function restore()
{
    local _tempDir="$( getBackupFolderPath )"
    if [ ! -f ${_tempDir}/def_interface ]; then 
        return 1
    fi

    _def_interface_name="$( cat ${_tempDir}/def_interface )"

    if [ -f ${_tempDir}/${_def_interface_name}-rp_filter ]; then
        cat ${_tempDir}/${_def_interface_name}-rp_filter > /proc/sys/net/ipv4/conf/${_def_interface_name}/rp_filter
    fi

    rm -fr ${_tempDir}
}

# Move all processes from the IVPN cgroup to the main cgroup
function removeAllPids() 
{    
    while IFS= read -r line
    do
        echo $line >> /sys/fs/cgroup/net_cls/cgroup.procs
    done < "${_cgroup_folder}/cgroup.procs"
}

function removepid()
{
    local _pid="$1"    
    if [ -z "${_pid}" ]; then
        echo "[!] ERROR: PID not defined" 1>&2
        exit 1
    fi   
    echo "[+] Removing PID ${_pid} from Split Tunneling group..."
    echo ${_pid} >> /sys/fs/cgroup/net_cls/cgroup.procs
}

function addpid()
{
    local _pid="$1"    
    if [ -z "${_pid}" ]; then
        echo "[!] ERROR: PID not defined"
        exit 1
    fi   
    echo "[+] Adding PID ${_pid} to Split Tunneling group..."
    echo ${_pid} >> ${_cgroup_folder}/cgroup.procs
}

function execute()
{    
    _user="$1"
    _app="$2"

    if [ -z "${_app}" ]; then
        echo "[!] ERROR: Application not defined" 1>&2
        exit 1
    fi   

    # Check if split tunneling enabled
    status > /dev/null 2>&1
    if [ $? != 0 ]; then
        echo "ERROR: split tunneling DISABLED. Please call 'start' command first" 1>&2
        exit 1
    fi

    # Obtaining information about user running the script
    # (script can be executed with 'sudo', but we should get real user)
    if [ -z "${_user}" ]; then
        _user="${SUDO_USER:-$USER}"
    fi    
    if [ -z "${_user}" ]; then
        echo "[!] User not defined" 1>&2
        exit 2
    fi

    addpid $$
    
    if [ $? != 0 ]; then
        echo "[!] Failed " 1>&2
        exit 3
    fi

    echo "[+] Starting '${_app}' for a user '${_user}'..."
    ${_bin_runuser} -u ${_user} -- ${_app}
}

function status()
{
    if [ -d ${_cgroup_folder} ]; then
         if ${_bin_grep} -E "^[0-9]+\s+${_routing_table_name}\s*$" /etc/iproute2/rt_tables &>/dev/null ; then
            echo "Split Tunneling: ENABLED"
            return 0
         fi
    fi
    echo "Split Tunneling: DISABLED"
    return 1
}

function info()
{
    #echo "[*] Interfaces (${_bin_ip} link):"
    #${_bin_ip} link
    #echo

    if [[ $1 != "-6" ]]; then
        _val=`cat /proc/sys/net/ipv4/ip_forward`
        echo "[*] /proc/sys/net/ipv4/ip_forward: ${_val}"
        
        echo ---------------------------------
        echo "[*] /proc/sys/net/ipv4/conf/*/rp_filter:"
        for i in /proc/sys/net/ipv4/conf/*/rp_filter; do
            _val=`cat $i`
            echo $i: ${_val}
        done
    fi
    
    echo ---------------------------------
    if [ ! -d ${_cgroup_folder} ]; then
        echo "[*] cgroup folder NOT exists: '${_cgroup_folder}'"
    else
        echo "[*] cgroup folder exists: '${_cgroup_folder}'"
        echo "[*] File '${_cgroup_folder}/net_cls.classid':"
        cat ${_cgroup_folder}/net_cls.classid
    fi
    
    echo ---------------------------------
    echo "[*] File '/etc/iproute2/rt_tables':"
    cat /etc/iproute2/rt_tables
        
    echo ---------------------------------
    if [[ $1 != "-4" ]]; then
        echo "[*] ip6tables -t mangle -S:"
        ${_bin_ip6tables} -t mangle -S
    fi
    if [[ $1 != "-6" ]]; then
        echo "[*] iptables -t mangle -S:"
        ${_bin_iptables} -t mangle -S
    fi

    echo ---------------------------------
    if [[ $1 != "-4" ]]; then
        echo "[*] ip6tables -t nat -S:"
        ${_bin_ip6tables} -t nat -S
    fi
    if [[ $1 != "-6" ]]; then
        echo "[*] iptables -t nat -S:"
        ${_bin_iptables} -t nat -S
    fi
    #echo ---------------------------------
    #echo "[*] ip6tables -S ${INPUT}:"
    #${_bin_ip6tables} -S ${INPUT}
    #echo "[*] iptables -S ${INPUT}:"
    #${_bin_iptables} -S ${INPUT}
    #
    #echo ---------------------------------
    #echo "[*] ip6tables -S ${OUTPUT}:"
    #${_bin_ip6tables} -S ${OUTPUT}
    #echo "[*] iptables -S ${OUTPUT}:"
    #${_bin_iptables} -S ${OUTPUT}
    
    echo ---------------------------------
    if [[ $1 != "-4" ]]; then
        echo "[*] ip6tables -S | grep IVPN:"
        ${_bin_ip6tables} -S  | grep IVPN
    fi
    if [[ $1 != "-6" ]]; then
        echo "[*] iptables -S | grep IVPN:"
        ${_bin_iptables} -S  | grep IVPN    
    fi
    echo ---------------------------------
    if [[ $1 != "-4" ]]; then
        echo "[*] ip -6 rule:"
        ${_bin_ip} -6 rule
    fi
    if [[ $1 != "-6" ]]; then
        echo "[*] ip rule:"
        ${_bin_ip} rule
    fi

    echo ---------------------------------
    if [[ $1 != "-4" ]]; then
        echo "[*] ip -6 route show table ${_routing_table_weight}"
        ${_bin_ip} -6 route show table ${_routing_table_weight}    
    fi
    if [[ $1 != "-6" ]]; then
        echo "[*] ip route show table ${_routing_table_weight}"
        ${_bin_ip} route show table ${_routing_table_weight} #${_routing_table_name}
    fi
    echo ---------------------------------

    detectDefRouteVars

    echo ---------------------------------
    status
}

function parseInputArgs()
{
    while [ $# -gt 0 ]; do
        # Check for empty parameter and shift if found
        if [ -z "$1" ]; then
            shift
            continue
        fi

        case "$1" in
            -interface) _def_interface_name="$2"; shift;;
            -gateway) _def_gateway="$2"; shift;;
            -interface6) _def_interface_nameIPv6="$2"; shift;;
            -gateway6) _def_gatewayIPv6="$2"; shift;;
            -inverse) _is_inversed=1; echo "'-inverse' flag defined!";;
            -inverse_block) _is_inversed_blocked=1; echo "'-inverse_block' flag defined!";;
            -inverse_block_ipv6) _is_inversed_blocked_ipv6=1; echo "'-inverse_block_ipv6' flag defined!";;
            *) echo "Unknown parameter: '$1'" 1>&2; exit 1;;
        esac
        shift
    done
}

if [[ $1 = "start" ]] ; then    
    shift    
    parseInputArgs "$@"    
    detectDefRouteVars # Ensure the input parameters not empty
    init

elif [[ $1 = "stop" ]] ; then    
    clean

elif [[ $1 = "reset" ]] ; then 
    removeAllPids

elif [[ $1 = "addpid" ]] ; then
    shift 
    addpid $@

elif [[ $1 = "removepid" ]] ; then
    shift 
    removepid $@

elif [[ $1 = "run" ]] ; then    
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

elif [[ $1 = "update-routes" ]] ; then
    # Linux is erasing ST routing rules when disable/enable default network interface, so we need to restore them back
    shift 
    detectDefRouteVars

    updateRoutes $@  

elif [[ $1 = "info" ]] ; then
    shift 
    info $@  

elif [[ $1 = "status" ]] ; then
    shift
    status $@

elif [[ $1 = "test" ]] ; then
    shift
    test $@

elif [[ $1 = "manual" ]] ; then
    _FUNCNAME=$2
    shift
    shift
    echo "Running manual command: ${_FUNCNAME}($@) "
    ${_FUNCNAME} $@
else
    echo "Script to control the Split-Tunneling functionality for Linux."
    echo "Applications running in the split tunnel environment do not use the VPN tunnel."
    echo "It is a part of Daemon for IVPN Client Desktop."
    echo "https://github.com/ivpn/desktop-app/daemon"
    echo "Created by Stelnykovych Alexandr."
    echo "Copyright (c) 2023 IVPN Limited."
    echo ""
    echo "Usage:"
    echo "Note! The script have to be started under privilaged user (sudo $0 ...)"
    echo "    $0 <command> [parameters]"
    echo "Parameters:"
    echo "    start [-interface <inf_name>] [-gateway <gateway>] [-interface6 <inf_name_IPv6>] [-gateway6 <gateway_IPv6>] [[-inverse] [-inverse_block]]"
    echo "        Initialize split-tunneling functionality"
    echo "        - interface         - (optional) name of IPv4 network interface to be used for ST environment"
    echo "        - gateway           - (optional) IPv4 gateway IP to be used for ST environment"
    echo "        - interface6        - (optional) name of IPv6 network interface to be used for ST environment"
    echo "        - gateway6          - (optional) IPv6 gateway IP to be used for ST environment"
    echo "        - inverse           - (optional) When defined - route specified applications exclusively through the VPN."
    echo "                                         All other traffic will bypass the VPN and use the default connection."
    echo "        - inverse_block     - (optional) Block connectivity for specified apps."
    echo "                                         For example, to block connectivity when VPN not enabled."
    echo "                                         Note: This option applicable only with '-inverse' option."
    echo "        - inverse_block_ipv6- (optional) Block IPv6 connectivity for specified apps."
    echo "                                         For example, to block IPv6 connectivity when VPN does not support IPv6."
    echo "                                         Note: This option applicable only with '-inverse' option."    
    echo "    stop"
    echo "        Uninitialize split-tunneling functionality"
    echo "    run [-u <username>] <command>"
    echo "        Start commands in split-tunneling environment"
    echo "        - command        - the command or path to binary to be executed"
    echo "        - username       - (optional) the account under which the command have to be executed"
    echo "    addpid <PID>"
    echo "        Add process to Split Tunneling environment"
    echo "        - PID             - process ID"
    echo "    removepid <PID>"
    echo "        Remove process from Split Tunneling environment"
    echo "        - PID             - process ID"
    echo "    update-routes"
    echo "        Update the routing table for packets within the split tunnel."
    echo "        Linux erases split-tunnel routing rules when the default network interface is disabled/enabled. This command restores those rules."
    echo "    reset"
    echo "        Remove all processes from Split Tunneling environment"
    echo "    status"
    echo "        Check split-tunneling status"
    echo "Examples:"
    echo "    Initialize split-tunneling functionality:"
    echo "        $0 start"
    echo "        $0 start -interface wlp3s0 -gateway 192.168.1.1"
    echo "    Start commands in split-tunneling environment:"
    echo "        $0 run firefox"
    echo "        $0 run /usr/bin/firefox"
    echo "        $0 run ping 8.8.8.8"
    echo "    Uninitialize split-tunneling functionality:"
    echo "        $0 stop"
    echo "    Check split-tunneling status:"
    echo "        $0 status"
fi
