# Info

As of macOS 14 (Darwin v23.x.x), it is no longer possible to obtain WiFi SSID for background daemons.

In modern macOS versions, to retrieve WiFi network information, the application must have Location Services privileges. However, these privileges are not available for privileged LaunchDaemons. 

To overcome this limitation, we use a separate **LaunchAgent** that is installed in the user environment by the UI app.

Refer to: ../../../../daemon/wifiNotifier/darwin/agent_xpc/readme.md
