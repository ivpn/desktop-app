# WiFi Information Retrieval on macOS

As of macOS 14 (Darwin v23.x.x), it is no longer possible to obtain WiFi SSID for background daemons.

In modern macOS versions, to retrieve WiFi network information, the application must have Location Services privileges. However, these privileges are not available for privileged LaunchDaemons. 

To overcome this limitation, we use a separate **LaunchAgent** *(ui/References/macOS/HelperProjects/launchAgent)* that is installed in the user environment by the UI app.

We employ XPC communication between the Agent and the Daemon. The Daemon acts as a "server", waiting for connections from Agents. Agents provide the Daemon with WiFi information upon request.

The Electron UI app uses a custom **NAPI module** *(ui/addons/wifi-info-macos)* to install/uninstall the LaunchAgent and to request Location Services permission from the OS.