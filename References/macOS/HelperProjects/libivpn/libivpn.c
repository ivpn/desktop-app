#include <net/route.h>
#include <err.h>
#include <errno.h>
#include <stdio.h>
#include <sysexits.h>
#include <unistd.h>

#include <syslog.h>
#include <dispatch/dispatch.h>


#include "libivpn.h"

#include "power_change_notifications.h"

#define EXPORT __attribute__((visibility("default")))

// IMPLEMENTATION
dispatch_queue_t queue;
xpc_connection_t connection;
CFRunLoopSourceRef netChangeRLSource = NULL;

// INTERNAL FUNCTIONS ---
const int   BUFSIZE = 1024;

void syslogSaveError(CFErrorRef error, const char* prefix)
{
	if (error == NULL && prefix == NULL)
		return;

	const CFStringEncoding  encoding  = kCFStringEncodingMacRoman;

	char        buffer[BUFSIZE];
	const char  *ptr = NULL;

	if (error!=NULL)
	{
		CFStringRef errorText = CFErrorCopyDescription(error);
		ptr      = CFStringGetCStringPtr(errorText, encoding);

		if (ptr == NULL)
		{
			if (CFStringGetCString(errorText, buffer, BUFSIZE, encoding))
				ptr = buffer;
		}

		if (ptr!=NULL)
		{
			if (prefix!=NULL)
				syslog(LOG_ALERT, "libivpn: %s %s", prefix, ptr);
			else
				syslog(LOG_ALERT, "libivpn: %s", ptr);
		}
		else
		{
			if (prefix!=NULL)
				syslog(LOG_ALERT, "libivpn: %s", prefix);
		}
	}
	else
		syslog(LOG_ALERT, "libivpn: %s", prefix);
}

void syslogSaveXpcObject(xpc_object_t object, const char* prefix)
{
    if (object == NULL && prefix == NULL)
        return;

    if (object != NULL)
    {
        char *text = xpc_copy_description(object);
        if (text!=NULL)
        {
            if (prefix!=NULL)
                syslog(LOG_ALERT, "libivpn: %s %s", prefix, text);
            else
                syslog(LOG_ALERT, "libivpn: %s", text);

            free(text);
        }
    }
    else
        syslog(LOG_ALERT, "libivpn: %s", prefix);
}
//-----------------------

dispatch_queue_t init_queue(char *name) {
    char qname[128];
    qname[0] = '\0';
    strncat(qname, name, 120);
    strcat(qname, ".queue");

    return dispatch_queue_create(qname, NULL);
}

EXPORT
void start_xpc_listener(char *name, int serviceTcpPort, uint64_t serviceSecret) {
    queue = init_queue(name);

    puts("libivpn: Starting listener");
		syslog(LOG_ALERT, "libivpn: Starting listener");

    connection = xpc_connection_create_mach_service(name, queue, XPC_CONNECTION_MACH_SERVICE_LISTENER);

    xpc_connection_set_event_handler(connection, ^(xpc_object_t client)
		{
        if(xpc_get_type(client) == XPC_TYPE_ERROR)
				{
            if(client == XPC_ERROR_CONNECTION_INTERRUPTED)
						{
								puts("libivpn: INTERRUPTED");
                syslog(LOG_ALERT, "libivpn: INTERRUPTED");
            }
						else if(client == XPC_ERROR_CONNECTION_INVALID)
						{
								puts("libivpn: INVALID");
                syslog(LOG_ALERT, "libivpn: INVALID");
            }

            syslogSaveXpcObject(client, NULL);

            char *error = (char *) xpc_dictionary_get_string(client, XPC_ERROR_KEY_DESCRIPTION);
            if(error)
						{
                printf("libivpn: error: %s\n", error);
								syslog(LOG_ALERT, "libivpn: error: %s", error);
                return;
            }
        }

        syslogSaveXpcObject(client, NULL);

        xpc_connection_set_event_handler(client, ^(xpc_object_t event)
				{
            syslogSaveXpcObject(event, "****: %s");
            fflush(stdout);

            if(xpc_get_type(event) != XPC_TYPE_DICTIONARY)
                return;

            if(xpc_dictionary_get_int64(event, "type") == LIBIVPN_XPC_MESSAGE_TYPE_START_REQUEST)
						{
                syslog(LOG_ALERT, "libivpn: **************** START REQUEST");
								puts( "libivpn: **************** START REQUEST");

                //xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
                xpc_object_t message = xpc_dictionary_create_reply(event);
                xpc_dictionary_set_int64(message, "type", LIBIVPN_XPC_MESSAGE_TYPE_STARTED_REPLY);
                xpc_dictionary_set_int64(message, "port", serviceTcpPort);
								xpc_dictionary_set_uint64(message, "secret", serviceSecret);

                xpc_connection_t remote = xpc_dictionary_get_remote_connection(event);

                xpc_connection_send_message(remote, message);
                xpc_release(message);

								puts("libivpn: SENT REPLY");
                syslog(LOG_ALERT, "libivpn: SENT REPLY");
            }
        });

        xpc_connection_resume(client);
    });

    xpc_connection_resume(connection);
}

EXPORT
void connect_to_agent(char *name, AgentConnectedHandler handler) {
    queue = init_queue(name);

    connection = xpc_connection_create_mach_service(name, queue, XPC_CONNECTION_MACH_SERVICE_PRIVILEGED);

    xpc_connection_set_event_handler(connection, ^(xpc_object_t server)
		{
        puts("HERE");
        fflush(stdout);
    });

    xpc_connection_resume(connection);

    // send a start request
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, "type", LIBIVPN_XPC_MESSAGE_TYPE_START_REQUEST);
    xpc_connection_send_message_with_reply(connection, message, NULL, ^(xpc_object_t reply)
		{
        if(xpc_get_type(reply) != XPC_TYPE_DICTIONARY) {
            handler(-1, 0);
            syslog(LOG_ALERT, "libivpn: Received reply in connect_to_agent");
            syslogSaveXpcObject(reply, NULL);
            return;
          }

        handler(xpc_dictionary_get_int64(reply, "port"), xpc_dictionary_get_uint64(reply, "secret"));
    });
    xpc_release(message);
}

EXPORT
void close_connection() {
    xpc_connection_cancel(connection);
}

EXPORT CFDictionaryRef get_bundle_dictionary(char *helperLabel)
{
    CFStringRef helperString = CFStringCreateWithCString(kCFAllocatorDefault,
      helperLabel,
      kCFStringEncodingMacRoman);

    CFURLRef url = CFURLCreateWithString(NULL, helperString, NULL);

    CFDictionaryRef dictionary = CFBundleCopyInfoDictionaryForURL(url);

    CFRelease(url);
    CFRelease(helperString);
    return dictionary;
}

EXPORT CFDictionaryRef get_smjob_dictionary(char *helperLabel)
{
    CFStringRef helperString = CFStringCreateWithCString(kCFAllocatorDefault,
      helperLabel,
      kCFStringEncodingMacRoman);

    CFDictionaryRef dictionary = SMJobCopyDictionary(
            kSMDomainSystemLaunchd,
            helperString);

    CFRelease(helperString);
    return dictionary;
}

EXPORT
int install_helper_with_auth(char *label, AuthorizationRef authRef) {
    CFErrorRef error;

    if(SMJobBless(kSMDomainSystemLaunchd,
                  CFStringCreateWithCString(kCFAllocatorDefault, label, kCFStringEncodingMacRoman),
                  (AuthorizationRef) authRef,
                  &error))
    {
				syslog(LOG_ALERT, "libivpn: helper installed (install_helper_with_auth)");
        return 1;
    }
    else
    {
        syslogSaveError(error, "SMJobBless failed. (install_helper_with_auth)");
        if (error != NULL)
            CFRelease(error);

        return 0;
    }
}

EXPORT
int install_helper(char *label) {
    CFErrorRef error;

    syslog(LOG_ALERT, "libivpn: Installing helper...");

    AuthorizationRef  authRef = NULL;
    OSStatus err = AuthorizationCreate(NULL, NULL, 0, &authRef);
    if(err == errAuthorizationSuccess) {
        if(SMJobBless(kSMDomainSystemLaunchd,
                      CFStringCreateWithCString(kCFAllocatorDefault, label, kCFStringEncodingMacRoman),
                      (AuthorizationRef) authRef,
                      &error))
        {
						syslog(LOG_ALERT, "libivpn: helper installed");
            return 1;
        }
        else
        {
            syslogSaveError(error, "SMJobBless failed. ");
            if (error != NULL)
              CFRelease(error);

            return 0;
        }
    } else {
				syslog(LOG_ALERT, "libivpn: ERROR GETTING AUTHORIZATION");
        puts("ERROR GETTING AUTHORIZATION");

        return 0;
    }
}

EXPORT
int remove_helper(char *label) {
    CFErrorRef error;

    AuthorizationItem authItem = { kSMRightModifySystemDaemons, 0, NULL, 0 };
    AuthorizationRights authRights = { 1, &authItem };
    AuthorizationFlags flags = kAuthorizationFlagDefaults |
                               kAuthorizationFlagInteractionAllowed |
                               kAuthorizationFlagPreAuthorize |
                               kAuthorizationFlagExtendRights;
    AuthorizationRef authRef = NULL;

    const char *prompt = "This will remove the previously installed helper.";

    AuthorizationItem envItems = {kAuthorizationEnvironmentPrompt, strlen(prompt), (void *)prompt, 0};
    AuthorizationEnvironment env = { 1, &envItems };

    OSStatus err = AuthorizationCreate(&authRights, &env, flags, &authRef);
    if(err == errAuthorizationSuccess) {
        if(SMJobRemove(kSMDomainSystemLaunchd, CFStringCreateWithCString(kCFAllocatorDefault, label, kCFStringEncodingMacRoman), (AuthorizationRef) authRef, true, &error)) {
            puts("REMOVED!");
						syslog(LOG_ALERT, "libivpn: Helper REMOVED!");
            return 1;
        } else {
            puts("ERROR");
						syslog(LOG_ALERT, "libivpn: ERROR (remove_helper)");

            puts( CFStringGetCStringPtr(CFErrorCopyDescription(error), kCFStringEncodingMacRoman) );
            //puts( [(__bridge NSError *)error description] );
            CFRelease(error);

            return 0;
        }
    } else {
        puts("ERROR GETTING AUTHORIZATION");
				syslog(LOG_ALERT, "libivpn: ERROR GETTING AUTHORIZATION (remove_helper)");

        return 0;
    }
}

EXPORT
int remove_helper_with_auth(char *label, AuthorizationRef authRef)
{
    CFErrorRef error;

    if (SMJobRemove(
        kSMDomainSystemLaunchd,
        CFStringCreateWithCString(kCFAllocatorDefault, label, kCFStringEncodingMacRoman),
        (AuthorizationRef) authRef,
        true,
        &error))
    {
        puts("REMOVED!");
				syslog(LOG_ALERT, "libivpn: Helper REMOVED! (remove_helper_with_auth)");
        return 1;
    }
    else
    {
        syslogSaveError(error, "SMJobRemove failed. ");
        if (error != NULL)
            CFRelease(error);
        return 0;
    }
}

void onDynamicStoreChanged(SCDynamicStoreRef store, CFArrayRef changedKeys, void *info) {
  CFNotificationCenterRef centerRef = CFNotificationCenterGetLocalCenter();
  CFNotificationCenterPostNotification(centerRef, CFSTR("net.ivpn.client.IVPN.NetworkConfigurationChangedNotification"), centerRef, NULL, true);
}

EXPORT
int register_network_change_monitor() {
  SCDynamicStoreRef storeRef;
  SCDynamicStoreContext context = {0, NULL, NULL, NULL, NULL};

  storeRef = SCDynamicStoreCreate(kCFAllocatorDefault, CFBundleGetIdentifier(CFBundleGetMainBundle()), onDynamicStoreChanged, &context);

  const CFStringRef keys[4] = {
    CFSTR("State:/Network/Global/DNS"),
    CFSTR("State:/Network/Global/IPv4"),
		CFSTR("State:/Network/Service/.*/DNS")
  };

  CFArrayRef watchedKeys = CFArrayCreate(kCFAllocatorDefault, (const void **)keys, 2, &kCFTypeArrayCallBacks); // 2 - means, only 2 kesy should be used from array
  if(!SCDynamicStoreSetNotificationKeys(storeRef, NULL, watchedKeys)) {
    CFRelease(watchedKeys);
    fprintf(stderr, "SCDynamicStoreSetNotificationKeys() failed: %s", SCErrorString(SCError()));
    CFRelease(storeRef);
    storeRef = NULL;
    return -1;
  }
  CFRelease(watchedKeys);
  netChangeRLSource = SCDynamicStoreCreateRunLoopSource(kCFAllocatorDefault, storeRef, 0);
  CFRunLoopAddSource(CFRunLoopGetCurrent(), netChangeRLSource, kCFRunLoopDefaultMode);

	syslog(LOG_ALERT, "libivpn: network_change_monitor STARTED");
	CFRunLoopRun(); // Start asynchronously: loop for detectiong changes
	syslog(LOG_ALERT, "libivpn: network_change_monitor STOPPED");

	return 0;
}

EXPORT
void remove_network_change_monitor() {
	CFRunLoopSourceRef tmp_netChangeRLSource = netChangeRLSource;
	netChangeRLSource = NULL;
	if (tmp_netChangeRLSource == NULL)
		return;

	CFRunLoopRemoveSource(CFRunLoopGetCurrent(), tmp_netChangeRLSource, kCFRunLoopDefaultMode);
	CFRelease(tmp_netChangeRLSource);

	CFRunLoopStop(CFRunLoopGetCurrent());
}

EXPORT
int wait_for_route_change() {
  int retval = 1;
  int s = socket(PF_ROUTE, SOCK_RAW, 0);

  if (s < 0)
    return 2;

  int n;
  char msg[2048];

  for(;;) {
    n = read(s, msg, 2048);
    struct rt_msghdr *rtm = (struct rt_msghdr *)msg;

    if (rtm->rtm_version != RTM_VERSION) {
      (void) printf("routing message version %d not understood\n",
          rtm->rtm_version);
      break;
    }

    if (rtm->rtm_errno)
      continue;

    if (rtm->rtm_type == RTM_DELETE || rtm->rtm_type == RTM_CHANGE) {
      retval = 0;
      break;
    }
  }

  close(s);
  return retval;
}

EXPORT
int get_interface_statistics(char *interface, u_int32_t *bytesReceived, u_int32_t *bytesSent)
{
    int     mib[6];
    char    *buf = NULL, *lim, *next;
    size_t  len;
    struct  if_msghdr *ifm;
    unsigned int ifindex = 0;

		if (!interface)
				return -1;

		ifindex = if_nametoindex(interface);
		if (!ifindex)
				return -1;

    mib[0] = CTL_NET;
    mib[1] = PF_ROUTE;
    mib[2] = 0;
    mib[3] = 0;
    mib[4] = NET_RT_IFLIST2;
    mib[5] = 0;

    if (sysctl(mib, 6, NULL, &len, NULL, 0) != 0)
        return -1;

    if ((buf = malloc(len)) == NULL)
	  		return -1;

    if (sysctl(mib, 6, buf, &len, NULL, 0) != 0)
    {
      	free(buf);
        return -1;
    }

    lim = buf + len;
    for (next = buf; next < lim;)
    {
        ifm = (struct if_msghdr *)next;
        next += ifm->ifm_msglen;

        if (ifm->ifm_type == RTM_IFINFO2)
        {
            struct if_msghdr2 *if2m = (struct if_msghdr2 *)ifm;

            if (if2m->ifm_index != ifindex)
                continue;

            /*
            * Get the interface stats.  These may get overriden
            * below on a per-interface basis.
            */
            // opackets = if2m->ifm_data.ifi_opackets;
            // ipackets = if2m->ifm_data.ifi_ipackets;
            *bytesSent = if2m->ifm_data.ifi_obytes;
            *bytesReceived = if2m->ifm_data.ifi_ibytes;
						return 0;
        }
    }

    free(buf);

    return 1;
}

EXPORT
int  power_change_initialize_notifications(PowerChangeCallback callback)
{
	int ret = PowerChangeInitializeNotifications();
	if (ret!=0)
		return ret;

	PowerChangeRegisterCallback(callback);
	return 0;
};

EXPORT
void power_change_uninitialize_notifications()
{
	PowerChangeUnInitializeNotifications();
};
