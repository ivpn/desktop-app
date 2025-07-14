#!/bin/sh

echo "[*] After remove (<%= version %> : <%= pkg %> : $1)"

# Obtaining information about user running the script
# (script can be executed with 'sudo', but we should get real user)
USER="${SUDO_USER:-$USER}"

# When removing package: $1==0 for RPM; $1 == "remove" for DEB
_IS_REMOVE=0
if [ "$1" = "remove" -o "$1" = "0" ]; then
  _IS_REMOVE=1
fi

# ########################################################################################
#
# COMPATIBILITY BLOCK (BEGIN)
#
# Next block is in use only for compatibility with old package versions (v3.8.20 and older)
#
# ########################################################################################
# RPM: do not forget to remove file '/opt/ivpn/mutable/rpm_upgrade.lock' (if exists)
if [ "<%= pkg %>" = "rpm" ]; then
    if [ -f /opt/ivpn/mutable/rpm_upgrade.lock ]; then
        rm /opt/ivpn/mutable/rpm_upgrade.lock || echo "[-] Failed to remove rpm_upgrade.lock"
    fi
fi
# ########################################################################################
# COMPATIBILITY BLOCK (END)
# ########################################################################################

if [ $_IS_REMOVE = 0 ]; then
  echo "[ ] Upgrade detected. After-remove operations skipped"
  exit 0
fi

silent() {
  "$@" > /dev/null 2>&1
}

has_systemd() {
  # Some OS vendors put systemd in ... different places ...
  [ -d "/lib/systemd/system/" -o -d "/usr/lib/systemd/system" ] && silent command -v systemctl
}

try_systemd_stop() {
    if has_systemd ; then
        echo "[ ] systemd detected. Trying to stop service ..."

        echo "[+] Stopping service"
        silent systemctl stop ivpn-service

        echo "[+] Disabling service"
        silent systemctl disable ivpn-service

        if [ -f "/etc/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            rm /etc/systemd/system/ivpn-service.service
        fi
        if [ -f "/usr/lib/systemd/system/ivpn-service.service" ]; then
            echo "[+] Removing service"
            rm /usr/lib/systemd/system/ivpn-service.service
        fi
    fi
}

# Remove AppArmor local override rules that were added for IVPN.
# This function cleans up IVPN-specific AppArmor rules from local override files
# and removes empty override files. The main AppArmor profiles remain untouched.
uninstall_apparmor_rules() {
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
  
  local LOCAL_OVERRIDE="$APPARMOR_LOCAL_DIR/$PROFILE_NAME"
  
  # Check if our rules are present
  if [ ! -f "$LOCAL_OVERRIDE" ] || ! grep -q "# IVPN rules - START" "$LOCAL_OVERRIDE"; then
    return 0
  fi
  
  echo "[i] Found OpenVPN AppArmor profile: $APPARMOR_DIR/$PROFILE_NAME"
  
  # Remove our rules using sed (delete from START to END markers, inclusive)
  sed -i '/# IVPN rules - START/,/# IVPN rules - END/d' "$LOCAL_OVERRIDE"
  echo "[+] Removed IVPN rules from $LOCAL_OVERRIDE"
  
  # Remove the file if it's empty (ignoring whitespace)
  if [ -f "$LOCAL_OVERRIDE" ] && [ ! -s "$LOCAL_OVERRIDE" ] || [ "$(grep -v '^[[:space:]]*$' "$LOCAL_OVERRIDE" | wc -l)" -eq 0 ]; then
    rm "$LOCAL_OVERRIDE"
    echo "[+] Removed empty local override file $LOCAL_OVERRIDE"
  fi
  
  # Reload the AppArmor profile
  if silent apparmor_parser -r "$APPARMOR_DIR/$PROFILE_NAME"; then
    echo "[+] Successfully reloaded OpenVPN AppArmor profile"
  else
    echo "[!] Warning: Failed to reload OpenVPN AppArmor profile"
  fi
}

uninstall_bash_completion() {
    # get bash completion folder (according to https://github.com/scop/bash-completion)
    bash_competion_folder=$(pkg-config --variable=completionsdir bash-completion 2>&1) 
    if [ $? -eq 0 ] && [ ! -z $bash_competion_folder ] ; then
      completion_file=${bash_competion_folder}/ivpn
      if [ -f ${completion_file} ] ; then
        echo "[+] Uninstalling bash completion ('${completion_file}')"
        rm "${completion_file}"    
      fi
    fi
}

# stop & disable service
try_systemd_stop

uninstall_apparmor_rules

uninstall_bash_completion

PLEASERUN_DIR="/usr/share/pleaserun/ivpn-service"
if [ -d $PLEASERUN_DIR ] ; then
  echo "[+] Service cleanup (pleaserun) ..."
  silent sh $PLEASERUN_DIR/cleanup.sh 
  rm -fr $PLEASERUN_DIR
fi 

#IVPN_DIR="/opt/ivpn"
#if [ -d $IVPN_DIR ] ; then
#  echo "[+] Removing other files ..."
#  # Normally, all files which were installed, deleted automatically.
#  # But ivpn-service also writing to 'mutable' additional temporary files (uninstaller know nothing about them).
#  # Therefore, we are completely removing all content of '/opt/ivpn/mutable'.
#  # Also, there could stay empty dirs which were not deleted automatically.
#  rm -rf $IVPN_DIR || echo "[-] Removing '$IVPN_DIR' folder failed"
#fi

echo "[+] Removing mutable data ..."
IVPN_TMP="/etc/opt/ivpn"
rm -rf $IVPN_TMP

echo "[+] Removing logs ..."
IVPN_LOG="/var/log/ivpn" 
rm -rf $IVPN_LOG


