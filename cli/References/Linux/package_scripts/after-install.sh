#!/bin/sh

echo "[*] After install (<%= version %> : <%= pkg %> : $1)"

NEED_TO_SAVE_INSTRUCTIONS=true
IVPN_OPT="/opt/ivpn"
IVPN_ETC="/opt/ivpn/etc"
IVPN_TMP="/etc/opt/ivpn/mutable"
IVPN_LOG="/var/log/ivpn"

SERVERS_FILE_BUNDLED="$IVPN_ETC/servers.json"
SERVERS_FILE_DEST="$IVPN_TMP/servers.json"

mkdir -p $IVPN_LOG
mkdir -p $IVPN_TMP

INSTRUCTIONS_FILE="$IVPN_TMP/service_install.txt"
[ -e $INSTRUCTIONS_FILE ] && rm $INSTRUCTIONS_FILE

silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent command -v systemctl
}

try_systemd_install() {
    if has_systemd ; then
        echo "[ ] systemd detected. Trying to start service ..."
        echo "[+] Stopping old service (if exists)"
        systemctl stop ivpn-service
        echo "[+] Enabling service"
        systemctl enable ivpn-service || return 1
        echo "[+] Starting service"
        systemctl start ivpn-service || return 1

        NEED_TO_SAVE_INSTRUCTIONS=false
        return 0
    else
        echo "[-] Unable to start service automatically"
    fi
}

install_bash_completion() {
    # get bash completion folder (according to https://github.com/scop/bash-completion)
    bash_competion_folder=$(pkg-config --variable=completionsdir bash-completion 2>&1) 
    if [ $? -eq 0 ] && [ ! -z $bash_competion_folder ] ; then
      completion_file=${bash_competion_folder}/ivpn
      echo "[+] Installing bash completion (into '${completion_file}')"
      silent cp "$IVPN_ETC/ivpn.bash-completion" "${completion_file}"
      silent chmod 644 "${completion_file}"
    else
      echo "[ ] Installing bash completion - SKIPPED"
    fi
}

# Install AppArmor local override rules for OpenVPN to allow access to IVPN files.
# This function adds minimal rules to permit OpenVPN to read IVPN configuration 
# files and execute IVPN connection scripts when AppArmor profiles are active.
# Rules are added to local override files to avoid modifying main profiles.
install_apparmor_rules() {
  if ! silent command -v apparmor_parser; then
    return 0
  fi

  # AppArmor profile names to check
  local PROFILE_NAMES="
    openvpn
    usr.sbin.openvpn
  "
  local APPARMOR_DIR="/etc/apparmor.d"
  local APPARMOR_LOCAL_DIR="$APPARMOR_DIR/local"
  
  # Find the first existing OpenVPN AppArmor profile
  local PROFILE_NAME=""
  for name in $PROFILE_NAMES; do
    if [ -f "$APPARMOR_DIR/$name" ]; then
      PROFILE_NAME="$name"
      break
    fi
  done
  
  if [ -z "$PROFILE_NAME" ]; then
    return 0
  fi
  
  echo "[i] Found OpenVPN AppArmor profile: $APPARMOR_DIR/$PROFILE_NAME"
  
  # Create local override directory if it doesn't exist
  silent mkdir -p "$APPARMOR_LOCAL_DIR"
  
  local LOCAL_OVERRIDE="$APPARMOR_LOCAL_DIR/$PROFILE_NAME"
  
  # Define the IVPN rules to add
  local IVPN_RULES="# IVPN rules - START
  /etc/opt/ivpn/mutable/openvpn.cfg r,
  /opt/ivpn/etc/ta.key r,
  /opt/ivpn/etc/ca.crt r,
  /opt/ivpn/etc/client.up rix,
  /opt/ivpn/etc/client.down rix,
# IVPN rules - END"
  
  # Remove existing IVPN rules if present (to allow updates)
  if [ -f "$LOCAL_OVERRIDE" ] && grep -q "# IVPN rules - START" "$LOCAL_OVERRIDE"; then
    echo "[+] Updating existing IVPN AppArmor local rules in $LOCAL_OVERRIDE"
    sed -i '/# IVPN rules - START/,/# IVPN rules - END/d' "$LOCAL_OVERRIDE"
  else
    echo "[+] Adding new IVPN AppArmor local rules to $LOCAL_OVERRIDE"
  fi
  
  # Add our rules to the local override file
  echo "$IVPN_RULES" >> "$LOCAL_OVERRIDE"
  
  # Reload the AppArmor profile
  if silent apparmor_parser -r "$APPARMOR_DIR/$PROFILE_NAME"; then
    echo "[+] Successfully reloaded OpenVPN AppArmor profile"
  else
    echo "[!] Warning: Failed to reload OpenVPN AppArmor profile"
  fi
}

echo "[+] Defining access rights for files ..."
silent chmod 0400 $IVPN_ETC/*             # can read only owner (root)
silent chmod 0600 $IVPN_ETC/servers.json  # can read/wrire only owner (root)
silent chmod 0700 $IVPN_ETC/*.sh          # can execute only owner (root)
silent chmod 0700 $IVPN_ETC/*.up          # can execute only owner (root)
silent chmod 0700 $IVPN_ETC/*.down        # can execute only owner (root)
silent chmod 0755 /usr/bin/ivpn           # can change only owner (root)
silent chmod 0755 /usr/bin/ivpn-service   # can change only owner (root)
silent chmod 0755 $IVPN_OPT/obfsproxy/obfs4proxy          # can change only owner (root)
silent chmod 0755 $IVPN_OPT/v2ray/v2ray                   # can change only owner (root)
silent chmod 0755 $IVPN_OPT/wireguard-tools/wg-quick      # can change only owner (root)
silent chmod 0755 $IVPN_OPT/wireguard-tools/wg            # can change only owner (root)
silent chmod 0755 $IVPN_OPT/dnscrypt-proxy/dnscrypt-proxy # can change only owner (root)
silent chmod 0755 $IVPN_OPT/kem/kem-helper                # can change only owner (root)

if [ -f "${SERVERS_FILE_BUNDLED}" ] && [ -f "${SERVERS_FILE_DEST}" ]; then 
  # New service version may use new format of 'servers.json'. 
  # We must be sure that new format is in use.
  echo "[+] Overwriting servers information by the data from the bundle ..."
  silent cp "${SERVERS_FILE_BUNDLED}" "${SERVERS_FILE_DEST}"  
fi

echo "[+] Service install start (pleaserun) ..."
INSTALL_OUTPUT=$(sh /usr/share/pleaserun/ivpn-service/install.sh)
if [ $? -eq 0 ]; then
    # Print output of the install script
    echo $INSTALL_OUTPUT
    try_systemd_install
else
    # Print output of the install script
    echo $INSTALL_OUTPUT
    echo "[-] Service install FAILED!"
fi

if $NEED_TO_SAVE_INSTRUCTIONS == true ; then
    echo $INSTALL_OUTPUT > $INSTRUCTIONS_FILE
    echo "[!] Service start instructions saved into file: '$INSTRUCTIONS_FILE'"
fi

install_apparmor_rules

install_bash_completion

# ########################################################################################
#
# Next lines is in use only for compatibility with old package versions (v3.8.20 and older)
#
# DEB: 'before-remove' script (old versions) saving account credentials into 'toUpgradeID.tmp' and doing logout,
# here we have to re-login
#
# ########################################################################################
FILE_ACCID_TO_UPGRADE="/opt/ivpn/mutable/toUpgradeID.tmp"
if [ -f $FILE_ACCID_TO_UPGRADE ]; then
  echo "[ ] Upgrade detected (after-install: old-style)"
  # It is an upgrade.
  # We need to re-login after installation finished.
  # Read account ID
  ACCID=$(cat $FILE_ACCID_TO_UPGRADE) || echo "[-] Finishing installation: Failed to read accountID to re-login"

  # do not forget to remove temporary file
  silent rm $FILE_ACCID_TO_UPGRADE

  echo "[+] Disabling firewall (after-install: old-style) ..."
  /usr/bin/ivpn firewall -off || echo "[-] Failed to disable firewall"

  if [ ! -z "$ACCID" ]; then
    # giving a chance for a daemon to fully start
    sleep 1
    echo "[+] Logging in (after-install: old-style) ..."
    /usr/bin/ivpn login $ACCID #||  echo "[-] Finishing installation: Failed to to re-login (try#1)"
    if [ ! $? -eq 0 ]; then
      echo "[-] Finishing installation: Failed to to re-login (try#1)"
      echo "[ ] Retry ..."
      sleep 3
      /usr/bin/ivpn login $ACCID ||  echo "[-] Finishing installation: Failed to to re-login (try#2)"
    fi
  fi
fi
