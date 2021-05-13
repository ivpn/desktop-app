// Author: Alexander Stelnykovych
// Date: 2017.09.02
//
// Functionality to detect changes of power status
#include "power_change_notifications.h"

#include <pthread.h>

#include <ctype.h>
#include <stdlib.h>
#include <stdio.h>

#include <mach/mach_port.h>
#include <mach/mach_interface.h>
#include <mach/mach_init.h>

#include <IOKit/pwr_mgt/IOPMLib.h>
#include <IOKit/IOMessage.h>

//////////////////////////////////////////////////
// Internal functionality
//////////////////////////////////////////////////
CFRunLoopRef _runLoop = NULL;
PowerChangeCallback _callback = NULL;
pthread_mutex_t _mutex = PTHREAD_MUTEX_INITIALIZER;

io_connect_t  _root_port; // a reference to the Root Power Domain IOService

void NotifyPowerChange(enum PowerStatus status)
{
	pthread_mutex_lock( &_mutex );
	if (_callback != NULL)
		_callback(status);
	pthread_mutex_unlock( &_mutex );
}

void PowerChangeCallBack( void * refCon, io_service_t service, natural_t messageType, void * messageArgument )
{
	//printf( "messageType %08lx, arg %08lx\n",
    //    (long unsigned int)messageType,
    //    (long unsigned int)messageArgument );

    switch ( messageType )
    {

        case kIOMessageCanSystemSleep:
            // Idle sleep is about to kick in. This message will not be sent for forced sleep.
            //    Applications have a chance to prevent sleep by calling IOCancelPowerChange.
            //    Most applications should not prevent idle sleep.
        	//
            //    Power Management waits up to 30 seconds for you to either allow or deny idle
            //    sleep. If you don't acknowledge this power change by calling either
            //    IOAllowPowerChange or IOCancelPowerChange, the system will wait 30
            //    seconds then go to sleep.

            //Uncomment to cancel idle sleep
            //IOCancelPowerChange( _root_port, (long)messageArgument );

            // we will allow idle sleep
            IOAllowPowerChange( _root_port, (long)messageArgument );
            break;

        case kIOMessageSystemWillSleep:
            // The system WILL go to sleep. If you do not call IOAllowPowerChange or
            //    IOCancelPowerChange to acknowledge this message, sleep will be
            //    delayed by 30 seconds.
        	//
            //    NOTE: If you call IOCancelPowerChange to deny sleep it returns
            //    kIOReturnSuccess, however the system WILL still go to sleep.

            IOAllowPowerChange( _root_port, (long)messageArgument );
            NotifyPowerChange(SystemWillSleep);
            break;

        case kIOMessageSystemWillPowerOn:
            //System has started the wake up process...
        	NotifyPowerChange(SystemWillPowerOn);
            break;

        case kIOMessageSystemHasPoweredOn:
        	//System has finished waking up...
        	NotifyPowerChange(SystemHasPoweredOn);
        	break;

        default:
            break;

    }
}

void* PowerManagmentChangeDetectorThread(void* data)
{
    // notification port allocated by IORegisterForSystemPower
    IONotificationPortRef  notifyPortRef;

    // notifier object, used to deregister later
    io_object_t            notifierObject;
   // this parameter is passed to the callback
    void*                  refCon;

    // register to receive system sleep notifications

    _root_port = IORegisterForSystemPower( refCon, &notifyPortRef, PowerChangeCallBack, &notifierObject );
    if ( _root_port == 0 )
    {
        printf("IORegisterForSystemPower failed\n");
        return NULL;
    }

    // add the notification port to the application runloop
    CFRunLoopAddSource( CFRunLoopGetCurrent(),
            IONotificationPortGetRunLoopSource(notifyPortRef), kCFRunLoopCommonModes );

    // Start the run loop to receive sleep notifications. Don't call CFRunLoopRun if this code
    //    is running on the main thread of a Cocoa or Carbon application. Cocoa and Carbon
    //    manage the main thread's run loop for you as part of their event handling
    //    mechanisms.

    printf("Started power detection\n");

    // save reference of RunLoop for current thread
    _runLoop = CFRunLoopGetCurrent();

    // Start loop
    CFRunLoopRun();

    printf("Stopped power detection\n");

    //Not reached, CFRunLoopRun doesn't return in this case.
    return NULL;
}

int LaunchNotificationsThread()
{
	if (_runLoop!=NULL)
		return 0;

    // Create the thread using POSIX routines.
    pthread_attr_t  attr;
    pthread_t       posixThreadID;
    int             returnVal;

    returnVal = pthread_attr_init(&attr);
    if (returnVal != 0)
    	return returnVal;

    returnVal = pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
    if (returnVal != 0)
    	return returnVal;

    int     threadError = pthread_create(&posixThreadID, &attr, &PowerManagmentChangeDetectorThread, NULL);

    returnVal = pthread_attr_destroy(&attr);
    if (returnVal != 0)
    	return returnVal;

    if (threadError != 0)
    	return threadError;

    return 0;
}

void StopNotificationsThread()
{
	if (_runLoop!=NULL)
		CFRunLoopStop(_runLoop);
	_runLoop = NULL;
}

//////////////////////////////////////////////////
// Public functions
//////////////////////////////////////////////////
int PowerChangeInitializeNotifications()
{
	pthread_mutex_lock( &_mutex );
	int ret = LaunchNotificationsThread();
	pthread_mutex_unlock( &_mutex );

	return ret;
};

void PowerChangeRegisterCallback(PowerChangeCallback callback)
{
	pthread_mutex_lock( &_mutex );
	_callback = callback;
	pthread_mutex_unlock( &_mutex );
};

void PowerChangeUnRegister()
{
	pthread_mutex_lock( &_mutex );
	_callback = NULL;
	pthread_mutex_unlock( &_mutex );
};

void PowerChangeUnInitializeNotifications()
{
	pthread_mutex_lock( &_mutex );
	_callback = NULL;
	StopNotificationsThread();
	pthread_mutex_unlock( &_mutex );
};

