#!/bin/sh

echo "[+] Files installed"

NEED_TO_SAVE_INSTRUCTIONS=true
INSTRUCTIONS_FILE="/opt/ivpn/service_install.txt"
[ -e $INSTRUCTIONS_FILE ] && rm $INSTRUCTIONS_FILE

echo "[+] Service install start (pleaserun) ..."
INSTALL_OUTPUT=$(sh /usr/share/pleaserun/ivpn-service/install.sh) 

if [ $? -eq 0 ]; then 
    # Print output of the install script
    echo $INSTALL_OUTPUT

    # Trying to start service automatically
    # Dirty hack: trying to parse the output of 'install.sh'
    # which contains instructions to start service manually.
    # E.g.:
    #       Platform systemd (default) detected. Installing service.
    #       To start this service, use: systemctl start ivpn-service
    # Here we trying to run this instructions in automatic way
    PREFIX_TEXT_TO_DETECT="To start this service, use:"

    
# need this to divide output by new line symbol    
IFS="
"
    for line in $INSTALL_OUTPUT
    do
        if echo ${line} | grep ${PREFIX_TEXT_TO_DETECT}; then
            cmd=${line#"$PREFIX_TEXT_TO_DETECT"}
            
            echo "[+] Trying to start service by command: '$cmd'"
            eval $cmd

            if [ $? -eq 0 ]; then 
                echo "[+] Service started"
                NEED_TO_SAVE_INSTRUCTIONS=false
            else
                echo "[-] Service start FAILED"
            fi

            break
        fi
    done
else
    # Print output of the install script
    echo $INSTALL_OUTPUT
    echo "[+] Service install FAILED!"
fi

if [ "$NEED_TO_SAVE_INSTRUCTIONS" == true ]; then
    echo $INSTALL_OUTPUT > $INSTRUCTIONS_FILE
    echo "[!] Service start instructions saved into file: '$INSTRUCTIONS_FILE'"
fi 