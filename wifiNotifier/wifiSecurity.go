package wifiNotifier

type WiFiSecurity int

const (
	WiFiSecurityNone WiFiSecurity = 0
	WiFiSecurityWEP  WiFiSecurity = 1
	/*WiFiSecurityWPAPersonal        WiFiSecurity = 2
	WiFiSecurityWPAPersonalMixed   WiFiSecurity = 3
	WiFiSecurityWPA2Personal       WiFiSecurity = 4
	WiFiSecurityPersonal           WiFiSecurity = 5
	WiFiSecurityDynamicWEP         WiFiSecurity = 6
	WiFiSecurityWPAEnterprise      WiFiSecurity = 7
	WiFiSecurityWPAEnterpriseMixed WiFiSecurity = 8
	WiFiSecurityWPA2Enterprise     WiFiSecurity = 9
	WiFiSecurityEnterprise         WiFiSecurity = 10*/
	WiFiSecurityUnknown WiFiSecurity = 0xFFFFFFFF
)
