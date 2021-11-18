#!/bin/bash

_CGROUP_NAME=ivpnsplittun
_CGROUP_CLASSID=0x4956504e      # Anything from 0x00000001 to 0xFFFFFFFF
_PACKETS_MARK_VALUE=1230393422  # Anything from 1 to 2147483647
_ROUTING_TABLE_WEIGHT=17        # Anything from 1 to 252

_LOCKWAITTIME=2
_CGROUP_FOLDER=/sys/fs/cgroup/net_cls/${_CGROUP_NAME}

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
    echo 
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
    sudo sed -i "/${_CGROUP_NAME}\s*$/d" /etc/iproute2/rt_tables
}

function rules_create()
{
    if [ -z $1 ]; then
        echo "[!] Default network interface is not defoned. Trying to determine it automatically..."
        _def_interface_name=$(route | grep '^default' | grep -o '[^ ]*$')
        echo "[i] Default network interface: '${_def_interface_name}'"
    else
        _def_interface_name=$1
    fi
    
    #rules_remove ${_def_interface_name}
    
    # Important! allow DNS request before setting mark rule (DNS request should not be marked)
    # TODO: necessary to test for DNS leaks!
    iptables -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p tcp --dport 53 -j ACCEPT
    iptables -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -p udp --dport 53 -j ACCEPT

    # Add mark on packets of classid ${_CGROUP_CLASSID}
    iptables -w ${_LOCKWAITTIME} -t mangle -A OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}

    # Force the packets to exit through default interface (eg. eth0, enp0s3 ...) with NAT
    iptables -w ${_LOCKWAITTIME} -t nat -A POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE

    # Packets with mark will use splittun table
    echo ip rule add fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}
    ip rule add fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}
}
function rules_remove()
{
    if [ -z $1 ]; then
        echo "[!] Default network interface is not defoned. Trying to determine it automatically..."
        _def_interface_name=$(route | grep '^default' | grep -o '[^ ]*$')
        echo "[i] Default network interface: '${_def_interface_name}'"
    else
        _def_interface_name=$1
    fi

    iptables -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p tcp --dport 53 -j ACCEPT
    iptables -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -p udp --dport 53 -j ACCEPT
    iptables -w ${_LOCKWAITTIME} -t mangle -D OUTPUT -m cgroup --cgroup ${_CGROUP_CLASSID} -j MARK --set-mark ${_PACKETS_MARK_VALUE}
    iptables -w ${_LOCKWAITTIME} -t nat -D POSTROUTING -m cgroup --cgroup ${_CGROUP_CLASSID} -o ${_def_interface_name} -j MASQUERADE
    ip rule del fwmark ${_PACKETS_MARK_VALUE} table ${_CGROUP_NAME}
}

function routes_create()
{
    if [ -z $1 ]; then
        echo "[!] Default gateway is not defoned. Trying to determine it automatically..."
        _def_gateway=$(ip route | awk '/default/ { print $3 }')
        echo "[i] Default gateway: '${_def_gateway}'"
    else
        _def_gateway=$1
    fi

    # splittun table has a default gateway to the default interface
    ip route add default via ${_def_gateway} table ${_CGROUP_NAME}
}
function routes_remove()
{
    if [ -z $1 ]; then
        echo "[!] Default gateway is not defoned. Trying to determine it automatically..."
        _def_gateway=$(ip route | awk '/default/ { print $3 }')
        echo "[i] Default gateway: '${_def_gateway}'"
    else
        _def_gateway=$1
    fi

    # splittun table has a default gateway to the default interface
    ip route del default via ${_def_gateway} table ${_CGROUP_NAME}
}

function debug_show_status()
{
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
    iptables -t mangle -S
    echo 

    echo "[ ] iptables -t nat -S:"
    iptables -t nat -S
    echo 

    echo "[ ] ip rule:"
    ip rule
    echo 

    echo "[ ] ip route show table ${_CGROUP_NAME}"
    ip route show table ${_CGROUP_NAME}
    echo

    echo "[ ] /proc/sys/net/ipv4/conf/*/rp_filter:"
    for i in /proc/sys/net/ipv4/conf/*/rp_filter; do 
        echo $i:
        cat $i 
    done
    echo     
}

function init()
{
    _def_interface_name=$1
    _def_gateway=$2

    cgroup_create
    routing_table_create    
    rules_create ${_def_interface_name}
    routes_create ${_def_gateway}
}

function clean() 
{
    _def_interface_name=$1
    _def_gateway=$2
    
    routes_remove ${_def_gateway}
    rules_remove ${_def_interface_name}
    routing_table_remove
    cgroup_remove    
}

if [[ $1 = "-manual" ]] ; then
    _FUNCNAME=$2
    shift 
    shift
    echo "Running manual command: ${_FUNCNAME}($@) "
    ${_FUNCNAME} $@
fi