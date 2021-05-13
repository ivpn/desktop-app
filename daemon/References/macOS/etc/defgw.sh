PATH=/sbin:/usr/sbin:$PATH

set -e

if [ "$1" = "-restore" ] ; then
    PRI_IFACE=`echo 'show State:/Network/Global/IPv4' | scutil | grep PrimaryInterface | sed -e 's/.*PrimaryInterface : //'`
    ROUTER=`echo 'show State:/Network/Global/IPv4' | scutil | grep Router | sed -e 's/.*Router : //'`

    route -n add -net 0.0.0.0 "${ROUTER}"
else
    echo "usage: $0 <-restore>"
    exit 1
fi
