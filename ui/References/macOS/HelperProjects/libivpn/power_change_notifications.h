// Author: Alexander Stelnykovych
// Date: 2017.09.02
//
// Functionality to detect changes of power status

enum PowerStatus
{
	SystemWillSleep = 0x280,
	SystemWillPowerOn = 0x320,
	SystemHasPoweredOn = 0x300,
};

typedef void (* PowerChangeCallback)(enum PowerStatus status);

int  PowerChangeInitializeNotifications();
void PowerChangeRegisterCallback(PowerChangeCallback callback);
void PowerChangeUnRegisterCallback();
void PowerChangeUnInitializeNotifications();
