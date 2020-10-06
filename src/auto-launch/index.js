import { Platform, PlatformEnum } from "@/platform/platform";

// initialize application auto-launcher
var AutoLaunch = require("auto-launch");
let launcherOptions = { name: "IVPN" };
var autoLauncher = null;

if (Platform() === PlatformEnum.Linux) {
  const fs = require("fs");
  const binaryPath = "/opt/ivpn/ui/ivpn-ui.AppImage";
  if (fs.existsSync(binaryPath)) launcherOptions.path = binaryPath;
  else launcherOptions = null;
}

if (launcherOptions != null) autoLauncher = new AutoLaunch(launcherOptions);

function AutoLaunchIsInitialized() {
  return autoLauncher != null;
}

export async function AutoLaunchIsEnabled() {
  if (!AutoLaunchIsInitialized()) return null;
  try {
    return await autoLauncher.isEnabled();
  } catch (err) {
    console.error("Error obtaining 'LaunchAtLogin' value: ", err);
    return null;
  }
}

export async function AutoLaunchSet(isEnabled) {
  if (!AutoLaunchIsInitialized()) return;
  if (isEnabled) await autoLauncher.enable();
  else await autoLauncher.disable();
}
