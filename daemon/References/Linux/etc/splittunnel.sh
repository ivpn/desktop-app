#!/bin/bash

_CGROUP_NAME=ivpnsplittun
_CGROUP_CLASSID=0x4956504e      # Anything from 0x00000001 to 0xFFFFFFFF
_PACKETS_MARK_VALUE=1230393422  # Anything from 1 to 2147483647
_ROUTING_TABLE_WEIGHT=17        # Anything from 1 to 252

_LOCKWAITTIME=2
_CGROUP_FOLDER=/sys/fs/cgroup/net_cls/${_CGROUP_NAME}

_use_ipv6=0 #1
_iptables=/sbin/iptables
_ip6tables=/sbin/ip6tables
_ip=/sbin/ip

function cgroup_create()
{
    # check is cgroup exists
    if [ ! -f ${_CGROUP_FOLDER}/net_cls.classid ]; then
       mkdir ${_CGROUP_FOLDER}
    fi
    
    echo ${_CGROUP_CLASSID} > ${_CGROUP_FOLDER}/net_cls.classid
}

function cgroup_remove() 
{
    # TODO: remove cgroup folders
    cgcreate -g net_cls:${_CGROUP_NAME}

    #    echo ${_CGROUP_FOLDER}
    #    # check is cgroup exists
    #    if [ ! -d ${_CGROUP_FOLDER} ]; then
    #       echo "cgroup not exists"
    #       return 0;
    #    fi
    #    rm -rf ${_CGROUP_FOLDER}
}

function routing_table_create()
{
    if grep -E "^[0-9]+\s+${_CGROUP_NAME}\s*$" /etc/iproute2/rt_tables &>/dev/null; then
       # table already defined
       return 0
    fi

    echo "${_ROUTING_TABLE_WEIGHT}      ${_CGROUP_NAME}" >> /etc/iproute2/rt_tables
}
function routing_table_remove()
{
    sed -i "/${_CGROUP_NAME}\s*$/d" /etc/iproute2/rt_tables
}

function rules_create()
{
    if [ -z $1 ]; then
        echo "[!] Default network interface is not defined. Trying to determine it automatically..."
        _def_interface_name=$(${_ip} route | awk '/default/ { print $5 }')
        echo "[+] Default network interface: '${_def_interface_name}'"
    else
        _def_interface_name=$1
    fi
        
    rules_remove $@ > /dev/null 2>&1

    # Important! allow DNS request before setting mark rule (DNS request should not be marked)
    # TODO: necessary to test for DNS leaks!
    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p tcp --dport 53 -j ACCEPT
    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p udp --dport 53 -j ACCEPT
    # Add mark on packets of classid ${_CGROUP_CLASSID}
    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}
    # Force the packets to exit through default interface (eg. eth0, enp0s3 ...) with NAT
    ${_iptables} -w ${_LOCKWAITTIME} -t nat -A POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE
    # Packets with mark will use splittun table
    ${_ip} rule add fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}

    # do the same for IPv6
    if [ ${_use_ipv6} == 1 ] && [ -f /proc/net/if_inet6 ]; then
        # Important! allow DNS request before setting mark rule (DNS request should not be marked)
        # TODO: necessary to test for DNS leaks!
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p tcp --dport 53 -j ACCEPT
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p udp --dport 53 -j ACCEPT
        # Add mark on packets of classid ${_CGROUP_CLASSID}
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}
        # Force the packets to exit through default interface (eg. eth0, enp0s3 ...) with NAT
        ${_ip6tables} -w ${_LOCKWAITTIME} -t nat -A POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE
        # Packets with mark will use splittun table
        ${_ip} -6 rule add fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}
    fi
}
function rules_remove()
{
    if [ -z $1 ]; then
        echo "[!] Default network interface is not defined. Trying to determine it automatically..."
        _def_interface_name=$(${_ip} route | awk '/default/ { print $5 }')
        echo "[+] Default network interface: '${_def_interface_name}'"
    else
        _def_interface_name=$1
    fi

    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p tcp --dport 53 -j ACCEPT
    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p udp --dport 53 -j ACCEPT
    ${_iptables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}
    ${_iptables} -w ${_LOCKWAITTIME} -t nat -D POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE
    ${_ip} rule del fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}

    # do the same for IPv6
    if [ ${_use_ipv6} == 1 ]; then
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p tcp --dport 53 -j ACCEPT
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p udp --dport 53 -j ACCEPT
        ${_ip6tables} -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}
        ${_ip6tables} -w ${_LOCKWAITTIME} -t nat -D POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE
        ${_ip} -6 rule del fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}    
    fi
}

function routes_create()
{
    _def_gateway=$1
    _def_gateway_ipv6=$2

    if [ -z ${_def_gateway} ]; then
        echo "[!] Default gateway is not defined. Trying to determine it automatically..."
        _def_gateway=$(${_ip} route | awk '/default/ { print $3 }')
        echo "[+] Default gateway: '${_def_gateway}'"
    fi

    routes_remove ${_def_gateway} ${_def_gateway_ipv6} > /dev/null 2>&1

    # splittun table has a default gateway to the default interface
    ${_ip} route add default via ${_def_gateway} table ${_CGROUP_NAME}


    if [ ${_use_ipv6} == 1 ] && [ -f /proc/net/if_inet6 ]; then
        if [ -z ${_def_gateway_ipv6} ]; then
            echo "[!] Default IPv6 gateway is not defined. Trying to determine it automatically..."
            _def_gateway_ipv6=$(${_ip} -6 route | awk '/default/ { print $3 }')
            echo "[+] Default IPv6 gateway: '${_def_gateway_ipv6}'"
        fi
        # splittun table has a default gateway to the default interface
        echo "${_ip} -6 route add default via ${_def_gateway_ipv6} table ${_CGROUP_NAME}"
        ${_ip} -6 route add default via ${_def_gateway_ipv6} table ${_CGROUP_NAME}
    fi
}
function routes_remove()
{
    _def_gateway=$1
    _def_gateway_ipv6=$2

    if [ -z ${_def_gateway} ]; then
        echo "[!] Default gateway is not defined. Trying to determine it automatically..."
        _def_gateway=$(${_ip} route | awk '/default/ { print $3 }')
        echo "[+] Default gateway: '${_def_gateway}'"
    fi

    # splittun table has a default gateway to the default interface
    ${_ip} route del default via ${_def_gateway} table ${_CGROUP_NAME}

    if [ ${_use_ipv6} == 1 ] && [ -f /proc/net/if_inet6 ]; then
        if [ -z ${_def_gateway_ipv6} ]; then
            echo "[!] Default IPv6 gateway is not defined. Trying to determine it automatically..."
            _def_gateway_ipv6=$(${_ip} -6 route | awk '/default/ { print $3 }')
            echo "[+] Default IPv6 gateway: '${_def_gateway_ipv6}'"
        fi

        # splittun table has a default gateway to the default interface
        ${_ip} -6 route del default via ${_def_gateway_ipv6} table ${_CGROUP_NAME}
    fi
}

function debug_show_status()
{
    echo "[ ] /proc/sys/net/ipv4/conf/*/rp_filter:"
    for i in /proc/sys/net/ipv4/conf/*/rp_filter; do 
        echo $i:
        cat $i 
    done
    echo     

    if [ ! -d ${_CGROUP_FOLDER} ]; then
        echo "[!] cgroup folder NOT exists: '${_CGROUP_FOLDER}'"
    else
        echo "[ ] cgroup folder exists: '${_CGROUP_FOLDER}'"
        echo "[ ] File '${_CGROUP_FOLDER}/net_cls.classid':"
        cat ${_CGROUP_FOLDER}/net_cls.classid
        echo 
    fi

    echo "[ ] File '/etc/iproute2/rt_tables':"
    cat /etc/iproute2/rt_tables
    echo 

    echo "[ ] iptables -t mangle -S:"
    ${_iptables} -t mangle -S
    echo 

    echo "[ ] iptables -t nat -S:"
    ${_iptables} -t nat -S
    echo 

    echo "[ ] ip rule:"
    ${_ip} rule
    echo 

    echo "[ ] ip route show table ${_CGROUP_NAME}"
    ${_ip} route show table ${_CGROUP_NAME}
    echo

    if [ ${_use_ipv6} == 1 ]; then
        echo "--- IPv6 status: ---"

        echo "[ ] ip6tables -t mangle -S:"
        ${_ip6tables} -t mangle -S
        echo 

        echo "[ ] ip6tables -t nat -S:"
        ${_ip6tables} -t nat -S
        echo 

        echo "[ ] ip -6 rule:"
        ${_ip} -6 rule
        echo 

        echo "[ ] ip -6 route show table ${_CGROUP_NAME}"
        ${_ip} -6 route show table ${_CGROUP_NAME}
        echo
    fi
}

function execute()
{    
    _app="$1"
    _user="$2"

    if [ -z ${_app} ]; then
        echo "[!] ERROR: Application not defined"
        exit 1
    fi   

    # Obtaining information about user running the script
    # (script can be executed with 'sudo', but we should get real user)
    if [ -z ${_user} ]; then
        _user="${SUDO_USER:-$USER}"
    fi    
    
    echo "[+] Starting '${_app}' for a user '${_user}'..."

    /usr/sbin/runuser -u ${_user} -- cgexec -g net_cls:${_CGROUP_NAME} "${_app}" &

    # NOTE: this command should be executed under the original user account (not root)    
    #cgexec -g net_cls:${_CGROUP_NAME} ${_app} &
}

function init()
{
    _user=$1
    _def_interface_name=$2
    _def_gateway_ipv4=$3
    _def_gateway_ipv6=$4
    
    cgroup_create
    routing_table_create    
    rules_create ${_def_interface_name}
    routes_create ${_def_gateway_ipv4} ${_def_gateway_ipv6}

    # Obtaining information about user running the script
    # (script can be executed with 'sudo', but we should get real user)
    if [ -z ${_user} ]; then
        _user="${SUDO_USER:-$USER}"
    fi    
    echo "[+] Creating cgroup for user '${_user}'"
    cgcreate -t ${_user}:${_user} -a ${_user}:${_user} -g net_cls:${_CGROUP_NAME}
    # sudo chown stenya:stenya ivpnsplittun/*
    # sudo chown stenya:stenya ivpnsplittun
}

function clean() 
{
    _def_interface_name=$1
    _def_gateway_ipv4=$2
    _def_gateway_ipv6=$3
    
    routes_remove ${_def_gateway_ipv4} ${_def_gateway_ipv6}
    rules_remove ${_def_interface_name}
    routing_table_remove
    cgroup_remove    
}

if [[ $1 = "-init" ]] ; then    
    shift 
    init  $@
elif [[ $1 = "-execute" ]] ; then    
    shift 
    execute $@    
elif [[ $1 = "-clean" ]] ; then    
    shift 
    clean $@
elif [[ $1 = "-manual" ]] ; then
    _FUNCNAME=$2
    shift 
    shift
    echo "Running manual command: ${_FUNCNAME}($@) "
    ${_FUNCNAME} $@
else
    echo "[!] Unknown comand!"
fi