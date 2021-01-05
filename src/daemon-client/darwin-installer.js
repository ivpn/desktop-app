import { Platform, PlatformEnum } from "@/platform/platform";

function InstallDaemon(onInstallationStarted, done) {
  let spawn = require("child_process").spawn;

  let logStringPrefix = "[IVPN Installer --install_helper]";

  try {
    let cmd = spawn(
      "/Applications/IVPN.app/Contents/MacOS/IVPN Installer.app/Contents/MacOS/IVPN Installer",
      ["--install_helper"]
    );

    if (onInstallationStarted != null) onInstallationStarted();

    cmd.stdout.on("data", data => {
      console.log(`${logStringPrefix}: ${data}`);
    });
    cmd.stderr.on("data", err => {
      console.log(`[ERROR] ${logStringPrefix}: ${err}`);
    });

    cmd.on("exit", code => {
      if (done) done(code);
    });
  } catch (e) {
    console.log(`Failed to run ${logStringPrefix}: ${e}`);
  }
}

function InstallDaemonIfRequired(onInstallationStarted, done) {
  if (Platform() !== PlatformEnum.macOS) return;

  let logStringPrefix = "[IVPN Installer --is_helper_installation_required]";

  let spawn = require("child_process").spawn;
  try {
    let cmd = spawn(
      "/Applications/IVPN.app/Contents/MacOS/IVPN Installer.app/Contents/MacOS/IVPN Installer",
      ["--is_helper_installation_required"]
    );

    cmd.on("exit", code => {
      // if exitCode == 0 - the daemon must be installed
      if (code == 0) InstallDaemon(onInstallationStarted, done);
      else if (done) done(code);
    });
  } catch (e) {
    console.log(`Failed to run ${logStringPrefix}: ${e}`);
    if (done) done(-1);
  }
}

export default {
  InstallDaemonIfRequired
};
