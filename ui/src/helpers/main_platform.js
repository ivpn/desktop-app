import { Platform, PlatformEnum } from "@/platform/platform";
import { app } from "electron";
import path from "path";

const os = require("os");

async function winInstallFolder() {
  return await new Promise((resolve, reject) => {
    let Registry = require("winreg");
    let regKey = new Registry({
      hive: Registry.HKLM,
      key: "\\Software\\IVPN Client",
    });

    regKey.get(Registry.DEFAULT_VALUE, function (err, item) {
      if (err) reject(`Error reading installation path (registry):${err}`);
      else resolve(item.value);
    });
  });
}

// Returns: null - if we are not running in Snap environment,
// otherwise it returns values of Snap sandbox environment variables
export function GetLinuxSnapEnvVars() {
  if (process.env.SNAP && process.env.SNAP_COMMON && process.env.SNAP_DATA) {
    if (app.getAppPath().startsWith(process.env.SNAP)) {
      return {
        SNAP: process.env.SNAP,
        SNAP_COMMON: process.env.SNAP_COMMON,
        SNAP_DATA: process.env.SNAP_DATA,
      };
    }
  }
}

export async function GetPortInfoFilePath() {
  switch (Platform()) {
    case PlatformEnum.macOS:
      return "/Library/Application Support/IVPN/port.txt";
    case PlatformEnum.Linux: {
      const snapVars = GetLinuxSnapEnvVars();
      if (snapVars != null) {
        console.log("SNAP environment detected!");
        return path.join(snapVars.SNAP_COMMON, "/opt/ivpn/mutable/port.txt");
      }
      return "/opt/ivpn/mutable/port.txt";
    }
    case PlatformEnum.Windows: {
      let dir = await winInstallFolder();
      return `${dir}\\etc\\port.txt`;
    }
    default:
      throw new Error(`Not supported platform: '${os.platform()}'`);
  }
}

export async function GetOpenSSLBinaryPath() {
  switch (Platform()) {
    case PlatformEnum.macOS:
      return "/usr/bin/openssl";
    case PlatformEnum.Linux:
      return "/usr/bin/openssl";
    case PlatformEnum.Windows: {
      if (os.arch() === "x64") {
        let dir = await winInstallFolder();
        return `${dir}\\OpenVPN\\x86_64\\openssl.exe`;
      } else throw new Error(`Not supported architecture: '${os.arch()}'`);
    }
    default:
      throw new Error(`Not supported platform: '${os.platform()}'`);
  }
}
