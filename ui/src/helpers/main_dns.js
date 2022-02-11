import { DnsEncryption } from "@/store/types";
import { Platform, PlatformEnum } from "@/platform/platform";
const log = require("electron-log");

async function windowsGetSystemDohConfigurations() {
  var output = "";

  let retCode = await new Promise((resolve, reject) => {
    let logStringPrefix = "Checking system DOH configurations";

    try {
      let spawn = require("child_process").spawn;
      let cmd = spawn("powershell", ["Get-DnsClientDohServerAddress"]);

      cmd.stdout.on("data", function (data) {
        output += data.toString();
      });

      cmd.on("error", (err) => {
        console.log(`[ERROR] ${logStringPrefix}: ${err}`);
        reject(err);
      });

      cmd.on("exit", (code) => {
        resolve(code);
      });
    } catch (e) {
      console.log(`Failed to run ${logStringPrefix}: ${e}`);
      reject(e);
    }
  });

  if (retCode != 0) return [];

  const reColumns = /\s*(?:\s|$)\s*/;
  const reLines = /\n*(?:\n|$)\n*/;
  var lines = output.split(reLines);

  let ret = []; //  { DnsHost: "", Encryption: DnsEncryption.None, DohTemplate: "", }
  lines.forEach((line) => {
    if (!line) return;

    var cols = line.split(reColumns);
    if (cols.length < 4) return;
    if (!cols[3].startsWith("https://")) return;

    ret.push({
      DnsHost: cols[0],
      Encryption: DnsEncryption.DnsOverHttps,
      DohTemplate: cols[3],
    });
  });

  return ret;
}

export async function GetSystemDohConfigurations() {
  let ret = [];

  if (Platform() === PlatformEnum.Windows) {
    log.debug(`windowsGetSystemDohConfigurations (enter)`);
    ret = await windowsGetSystemDohConfigurations();
    log.debug(`windowsGetSystemDohConfigurations (exit)`);
  }
  return ret;
}
