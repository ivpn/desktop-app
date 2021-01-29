#include <ServiceManagement/ServiceManagement.h>
#include <Security/Security.h>
#include <syslog.h>
#include <string.h>

#define HELPER_LABEL "net.ivpn.client.Helper"
#define HELPER_TO_REMOVE_PLIST_PATH "/Library/LaunchDaemons/net.ivpn.client.Helper.plist"
#define HELPER_TO_REMOVE_BIN_PATH "/Library/PrivilegedHelperTools/net.ivpn.client.Helper"

// #define IS_INSTALLER 0   //  IS_INSTALLER should be passed by compiler
//  Makefile example: cc -D IS_INSTALLER='1' ...

/*
#define LOG_EMERG       0       // system is unusable
#define LOG_ALERT       1       // action must be taken immediately
#define LOG_CRIT        2       // critical conditions
#define LOG_ERR         3       // error conditions
#define LOG_WARNING     4       // warning conditions
#define LOG_NOTICE      5       // normal but significant condition
#define LOG_INFO        6       // informational
#define LOG_DEBUG       7       // debug-level messages
*/
void logmes(int mesType, const char* text) {
    syslog(mesType, "%s", text);
    if (IS_INSTALLER!=0)
      printf("[mestype:%d] IVPN Installer: %s\n", mesType, text);
    else
      printf("[mestype:%d] IVPN UnInstaller: %s\n", mesType, text);
}

void logmesError(CFErrorRef error) {
	if (error == NULL)
		return;

    CFStringRef errorText = CFErrorCopyDescription(error);
    if (errorText==NULL)
        return;

	const char  *ptr      = CFStringGetCStringPtr(errorText, kCFStringEncodingUTF8);
    if (ptr==NULL)
    {
        const CFIndex bufSize = 1024;
        char buff[bufSize];
        // if CFStringGetCStringPtr returns null - trying to use CFStringGetCString
        if (CFStringGetCString(errorText, buff, bufSize, kCFStringEncodingUTF8))
            logmes(LOG_ERR, buff);
        return;
    }

    logmes(LOG_ERR, ptr);
}

CFDictionaryRef get_bundle_dictionary(const char *bundlePath) {
    CFStringRef bundleString = CFStringCreateWithCString(kCFAllocatorDefault, bundlePath, kCFStringEncodingMacRoman);
    CFStringRef bundleStringEscaped = CFURLCreateStringByAddingPercentEscapes(NULL, bundleString, NULL, NULL, kCFStringEncodingUTF8);
    CFURLRef url = CFURLCreateWithString(NULL, bundleStringEscaped, NULL);
    CFDictionaryRef dictionary = CFBundleCopyInfoDictionaryForURL(url);

    if (url!=NULL)
      CFRelease(url);
    if (bundleString!=NULL)
      CFRelease(bundleString);

    return dictionary;
}

int getBundleVer(const char* bundlePath, char* retBuff, int buffSize) {
    CFDictionaryRef retDict = get_bundle_dictionary(bundlePath);
    if (retDict == NULL)
      return 1;

    int retVal = 0;
    CFStringRef key = CFStringCreateWithCString(kCFAllocatorDefault, "CFBundleVersion", kCFStringEncodingMacRoman);
    if (retDict != NULL && CFDictionaryContainsKey(retDict, key))
    {
        CFStringRef ver = CFDictionaryGetValue(retDict, key);
        if (ver==NULL)
          retVal = 2;
        else
        {
          if (!CFStringGetCString(ver, retBuff, buffSize, kCFStringEncodingMacRoman))
            retVal = 3;
          CFRelease(ver);
        }
    }
    else
        retVal = 4;

    if (key!=NULL)
      CFRelease(key);
    if (retDict!=NULL)
      CFRelease(retDict);

    return retVal;
}

int getInstalledHelperBundlePath(char* retBuff, int buffSize) {
    CFStringRef helperString = CFStringCreateWithCString(kCFAllocatorDefault, HELPER_LABEL, kCFStringEncodingMacRoman);
    CFDictionaryRef retDict = SMJobCopyDictionary(kSMDomainSystemLaunchd, helperString);
    if (helperString!=NULL)
      CFRelease(helperString);

    if (retDict == NULL)
      return 1;

    int retVal = 0;
    CFStringRef key = CFStringCreateWithCString(kCFAllocatorDefault, "ProgramArguments", kCFStringEncodingMacRoman);
    if (CFDictionaryContainsKey(retDict, key))
    {
        CFArrayRef program_arguments = CFDictionaryGetValue(retDict, key);
        if (program_arguments!=NULL)
        {
          CFStringRef helperBundlePath = CFArrayGetValueAtIndex(program_arguments, 0);
          if (!CFStringGetCString(helperBundlePath, retBuff, buffSize, kCFStringEncodingMacRoman))
            retVal = 2;
        }
        else
          retVal = 3;
    }
    else
        retVal = 4;

    if (key!=NULL)
      CFRelease(key);
    if (retDict!=NULL)
      CFRelease(retDict);

    return retVal;
}

void get_versions(char* retInstalledVer, char* retCurrentVer, int buffersSize) {
    // check if the helper of current version is already installed
    char installedBundlePath[256]={0};
    getBundleVer("/Applications/IVPN.app/Contents/MacOS/IVPN Installer.app/Contents/Library/LaunchServices/net.ivpn.client.Helper", retCurrentVer, buffersSize);

    if (getInstalledHelperBundlePath(installedBundlePath, 256)==0)
    {
      // helper is installed
      getBundleVer(installedBundlePath, retInstalledVer, buffersSize);;
    }
}

// returns 0 in case if helper must (and can) be installed
int is_helper_installation_required() {
    // check if the helper of current version is already installed
    char installedVer[128] = {0};
    char currentVer[128] = {0};

    get_versions(installedVer, currentVer, 128);
    if (currentVer[0]==0)
      return 1; // Unable to install IVPN Helper. Please, copy 'IVPN.app' to '/Applications'

    if (installedVer[0]!=0)
    {
      if (strcmp(installedVer, currentVer)==0)
        return 1; // Required version of IVPN Helper is already installed. No installation needed

      return 0; // Another version is installed. Upgrade required
    }

    return 0; // helper not installed. Installation required
}


int remove_helper_with_auth(AuthorizationRef authRef) {
  CFErrorRef error = NULL;
  int ret = 0;

  bool isSuccess = SMJobRemove(kSMDomainSystemLaunchd, CFStringCreateWithCString(kCFAllocatorDefault, HELPER_LABEL, kCFStringEncodingMacRoman), authRef, true, &error);
  if (!isSuccess)
  {
    logmesError(error);
    logmes(LOG_ERR, "ERROR: Cannot remove helper");
    if (error != NULL) CFRelease(error);
    return 1;
  }

  const char *filesToRemove[] = {HELPER_TO_REMOVE_PLIST_PATH, HELPER_TO_REMOVE_BIN_PATH, NULL};
  OSStatus err = AuthorizationExecuteWithPrivileges(authRef, (const char*) "/bin/rm", kAuthorizationFlagDefaults, filesToRemove, NULL);
  if (err)
  {
    char messageBuff[256] = {0};
    snprintf(messageBuff, 256, "ERROR (%d): Error removing files: '%s' and/or '%s'", err, HELPER_TO_REMOVE_PLIST_PATH, HELPER_TO_REMOVE_BIN_PATH);
    logmes(LOG_ERR, messageBuff);
    ret=2;
  }

  if (ret==0)
	  logmes(LOG_INFO, "Success (IVPN Helper removed)");
  else
    logmes(LOG_ERR, "IVPN helper removal not complete successfully.");

  return ret;
}

int remove_helper() {
    logmes(LOG_INFO, "Removing IVPN helper...");

    CFErrorRef error = NULL;

    AuthorizationItem authItem = { kSMRightModifySystemDaemons, 0, NULL, 0 };
    AuthorizationRights authRights = { 1, &authItem };
    AuthorizationFlags flags = kAuthorizationFlagDefaults |
                               kAuthorizationFlagInteractionAllowed |
                               kAuthorizationFlagPreAuthorize |
                               kAuthorizationFlagExtendRights;
    AuthorizationRef authRef = NULL;

    const char *prompt = "This will remove the previously installed IVPN helper.\n\n";
    AuthorizationItem envItems = {kAuthorizationEnvironmentPrompt, strlen(prompt), (void *)prompt, 0};
    AuthorizationEnvironment env = { 1, &envItems };

    OSStatus err = AuthorizationCreate(&authRights, &env, flags, &authRef);
    if(err == errAuthorizationSuccess)
    {
      int ret = remove_helper_with_auth(authRef);
      AuthorizationFree(authRef, kAuthorizationFlagDefaults);
      return ret;
    }

    logmes(LOG_ERR, "ERROR: Getting authorization failed (IVPN helper NOT removed)");
    return err;
}

int install_helper() {
    logmes(LOG_INFO, "Installing IVPN helper...");

    bool isUpgrade = false;

    char messageBuff[256] = {0};

    // check if the helper of current version is already installed
    char installedVer[128] = {0};
    char currentVer[128] = {0};

    get_versions(installedVer, currentVer, 128);
    if (currentVer[0]==0)
    {
      logmes(LOG_ERR, "Unable to install IVPN Helper. Please, copy 'IVPN.app' to '/Applications'");
      return 1;
    }

    if (installedVer[0]!=0)
    {
      if (strcmp(installedVer, currentVer)==0)
      {
        snprintf(messageBuff, 256, "Required version of IVPN Helper (v%s) is already installed. IVPN Helper installation skipped.", installedVer);
        logmes(LOG_NOTICE, messageBuff);
        return 1;
      }

      isUpgrade = true;
      snprintf(messageBuff, 256, "Upgrading IVPN helper v%s (already installed version v%s) ...", currentVer, installedVer);
      logmes(LOG_INFO, messageBuff);
    }
    else
    {
      // helper not installed
      snprintf(messageBuff, 256, "Installing IVPN helper v%s ...", currentVer);
      logmes(LOG_INFO, messageBuff);
    }

    CFErrorRef error = NULL;
    AuthorizationItem authItem = { kSMRightBlessPrivilegedHelper, 0, NULL, 0 };
    AuthorizationRights authRights = { 1, &authItem };
    AuthorizationFlags flags = kAuthorizationFlagDefaults |
                               kAuthorizationFlagInteractionAllowed |
                               kAuthorizationFlagPreAuthorize |
                               kAuthorizationFlagExtendRights;

    const char *promptUpgrade = "A new version of IVPN has been installed and the privileged helper must be upgraded too.\n\n";
    const char *prompt = "A privileged helper must be installed to use the IVPN client.\n\n";
    if (isUpgrade)
      prompt = promptUpgrade;

    AuthorizationItem envItems = {kAuthorizationEnvironmentPrompt, strlen(prompt), (void *)prompt, 0};
    AuthorizationEnvironment env = { 1, &envItems };

    AuthorizationRef  authRef = NULL;
    OSStatus err = AuthorizationCreate(&authRights, &env, flags, &authRef);
    if(err == errAuthorizationSuccess)
    {
        if (isUpgrade) {
          remove_helper_with_auth(authRef);
        }

        bool isSuccess = SMJobBless(kSMDomainSystemLaunchd,
                      CFStringCreateWithCString(kCFAllocatorDefault, HELPER_LABEL, kCFStringEncodingMacRoman),
                      (AuthorizationRef) authRef,
                      &error);

        AuthorizationFree(authRef, kAuthorizationFlagDefaults);

        if (isSuccess)
        {
			      logmes(LOG_INFO, "IVPN helper installed.");
            return 0;
        }
        else
        {
            logmesError(error);
            logmes(LOG_ERR, "ERROR: SMJobBless failed (IVPN helper NOT installed)");
            if (error != NULL) CFRelease(error);
            return 1;
        }
    }

	logmes(LOG_ERR, "ERROR: Getting authorization failed (IVPN helper NOT installed)");
    return err;
}

int uninstall() {
      logmes(LOG_INFO, "Uninstalling IVPN ...");
      const char *homeDir = getenv("HOME");

      CFErrorRef error = NULL;
      AuthorizationRef authRef = NULL;

      OSStatus err = AuthorizationCreate(NULL, NULL, kAuthorizationFlagDefaults, &authRef);
      if(err != errAuthorizationSuccess)
      {
        logmes(LOG_ERR, "ERROR: Creating authorization failed");
        return 1;
      }

      const char *args0[] = {NULL};
      // Hack to force "authentication required" window to pop-up;
      if (AuthorizationExecuteWithPrivileges(authRef, (const char*) "/bin/echo", kAuthorizationFlagDefaults, args0, NULL))
      {
        logmes(LOG_ERR, "FAILED to get privileges");
        return 2;
      }

      bool hasErrors = false;

      printf("[ ] Disabling IVPN firewall ...\n");
      system("/Applications/IVPN.app/Contents/MacOS/cli/ivpn firewall -off");

      printf("[ ] Disconnecting IVPN ...\n");
      system("/Applications/IVPN.app/Contents/MacOS/cli/ivpn disconnect");

      printf("[ ] Closing IVPN app...\n");
      if (system("/usr/bin/osascript -e 'quit app \"ivpn-ui\"'"))
      {
        logmes(LOG_ERR, "ERROR: Unable to close application (ivpn-ui). IVPN not uninstalled.");
        system( "/usr/bin/osascript -e 'display alert \"IVPN Uninstaller\" message \"Please, close IVPN application and try again.\"'");
        return 3;
      }
      if (system("/usr/bin/osascript -e 'quit app \"IVPN\"'"))
      {
        logmes(LOG_ERR, "ERROR: Unable to close application (IVPN). IVPN not uninstalled.");
        system( "/usr/bin/osascript -e 'display alert \"IVPN Uninstaller\" message \"Please, close IVPN application and try again.\"'");
        return 4;
      }

      printf("[ ] Logout ...\n");
      system("/Applications/IVPN.app/Contents/MacOS/cli/ivpn logout");

      printf("[ ] Removing apps defaults...\n");
      system("/usr/bin/defaults delete net.ivpn.client.IVPN"); // old UI bundleID
      system("/usr/bin/defaults delete com.electron.ivpn-ui");

      printf("[ ] Removing helper ...\n");
      remove_helper_with_auth(authRef);

      char relFile1[128], relFile2[128];
      snprintf(relFile1, 128, "%s/Library/Preferences/net.ivpn.client.IVPN.plist", homeDir);
      snprintf(relFile2, 128, "%s/Library/Preferences/com.electron.ivpn-ui.plist", homeDir);

      const char *filesToRemove[] = {
        "/Library/Logs/IVPN Agent.log",
        "/Library/Logs/IVPN Agent.log.0",
        "/Library/Logs/IVPN Agent CrashInfo.log",
        "/Library/Logs/IVPN Agent CrashInfo.log.0",
        "/Library/Application Support/net.ivpn.client.Agent/last-btime", // seems, the file created by OS,
        relFile1,
        relFile2
      };

      for (int i=0;i<7;i++)
      {
        char* fname = filesToRemove[i];
        printf("[ ] Removing file: %s ...\n", fname);
        char *args[] = {fname, NULL};
        OSStatus ret = AuthorizationExecuteWithPrivileges(authRef, (const char*) "/bin/rm", kAuthorizationFlagDefaults, args, NULL);
        if (ret)
        {
          printf("    error removing file (%d): '%s'\n", ret, fname);
          hasErrors = true;
        }
      }

      char relDir1[128], relDir2[128];
      snprintf(relDir1, 128, "%s/Library/Application Support/IVPN", homeDir);  // old UI
      snprintf(relDir2, 128, "%s/Library/Application Support/ivpn-ui", homeDir);

      const char *foldersToRemove[] = {
        "/Applications/IVPN.app",
        "/Library/Application Support/IVPN/OpenVPN",
        "/Library/Application Support/IVPN",
        "/Library/Application Support/net.ivpn.client.Agent/LocalMachine", // seems, the folder created by OS
        "/Library/Application Support/net.ivpn.client.Agent", // seems, the folder created by OS
        relDir1,
        relDir2
      };

     for (int i=0;i<7;i++)
     {
        char* fname = foldersToRemove[i];
        printf("[] Removing folder: %s ...\n", fname);
        char *args[] = {"-R",  fname, NULL};
        OSStatus ret = AuthorizationExecuteWithPrivileges(authRef, (const char*) "/bin/rm", kAuthorizationFlagDefaults, args, NULL);
        if (ret)
        {
          printf("    error removing folder (%d): '%s'\n", ret, fname);
          hasErrors = true;
        }
      }

      AuthorizationFree(authRef, kAuthorizationFlagDefaults);

      if (hasErrors)
        system( "/usr/bin/osascript -e 'display alert \"IVPN Uninstaller\" message \"IVPN removed with errors!\"'");
      else
        system( "/usr/bin/osascript -e 'display alert \"IVPN Uninstaller\" message \"IVPN removed!\"'");

      return hasErrors;
}

int start_helper() {
  // This instructions is creating 'fake' connection to service.
  // Benefints of this operations - OS will restart helper if it will crash.
  // Additionally, (already installed) helper will be started (if it not started)
  xpc_connection_t connection = xpc_connection_create_mach_service(HELPER_LABEL, NULL, XPC_CONNECTION_MACH_SERVICE_PRIVILEGED);
  xpc_connection_set_event_handler(connection, ^(xpc_object_t server)
  {
      // event received
  });
  xpc_connection_resume(connection);

  // send a start request
  xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
  xpc_dictionary_set_int64(message, "test", 0);
  xpc_connection_send_message_with_reply(connection, message, NULL, ^(xpc_object_t reply)
  {
    // reply received
  });
  xpc_release(message);

  return 0;
}

int main(int argc, char **argv) {
    if (argc >= 2)
    {
        if (strcmp(argv[1], "--is_helper_installation_required")==0)
            return is_helper_installation_required();
        if (strcmp(argv[1], "--install_helper")==0) {
            int ret = install_helper();
            if (ret==0)
            {
              // Helper must be already started
              // Just making it to be able automatically restart on crash
              start_helper();
            }
            return ret;
        }
        if (strcmp(argv[1], "--uninstall_helper")==0)
            return remove_helper();
        if (strcmp(argv[1], "--start_helper")==0)
            return start_helper();
        if (strcmp(argv[1], "--uninstall")==0)
            return uninstall();
    }

    if (IS_INSTALLER != 0)
    {
      printf("No arguments provided.\n");
      printf(" arguments:\n");
      printf("    --install_helper\n");
      printf("    --uninstall_helper\n");
      printf("    --start_helper\n");
      printf("    --uninstall\n");
      printf("    --is_helper_installation_required (returns exit code: 0 -> helper have to be installed)\n");
      return 1;
    }

    return uninstall();
}
