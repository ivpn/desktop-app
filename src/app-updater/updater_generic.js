//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui2
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

import client from "@/daemon-client";
import config from "@/config";
import store from "@/store";
import { AppUpdateStage } from "@/store/types";
import { IsNewVersion } from "./helper";
import { Platform, PlatformEnum } from "@/platform/platform";
import {
  ValidateDataOpenSSLCertificate,
  ValidateFileOpenSSLCertificate
} from "@/helpers/main_signature";

const os = require("os");

let DownloadUpdateCancelled = false;

export async function CheckUpdates() {
  try {
    let updatesInfoData = await client.GetAppUpdateInfo();

    if (
      !updatesInfoData ||
      !updatesInfoData.updateInfoRespRaw ||
      !updatesInfoData.updateInfoSignRespRaw
    )
      return null;

    if (
      (await ValidateDataOpenSSLCertificate(
        updatesInfoData.updateInfoRespRaw,
        updatesInfoData.updateInfoSignRespRaw,
        "IVPN_UpdateInfo"
      )) !== true
    ) {
      console.error("Failed to validate application update info signature");
      return null;
    }

    let updatesInfo = JSON.parse(`${updatesInfoData.updateInfoRespRaw}`);

    if (!updatesInfo) return null;
    if (!updatesInfo.generic || !updatesInfo.generic.version) return null;
    return updatesInfo;
  } catch (e) {
    if (e instanceof SyntaxError)
      console.error("[updater] parsing update file info error: ", e.message);
    else console.error("[updater] error: ", e);

    return null;
  }
}

export function IsNewerVersion(updatesInfo, currDaemonVer, currUiVer) {
  if (updatesInfo && updatesInfo.generic) {
    return (
      IsNewVersion(currDaemonVer, updatesInfo.generic.version) ||
      IsNewVersion(currUiVer, updatesInfo.generic.version)
    );
  }
  return false;
}

function setState(updateState) {
  store.commit("uiState/appUpdateProgress", updateState);
}

export async function CancelDownload() {
  DownloadUpdateCancelled = true;
}

export async function Install() {
  let updateProgress = store.state.uiState.appUpdateProgress;
  if (
    !updateProgress ||
    !updateProgress.readyToInstallBinary ||
    updateProgress.state != AppUpdateStage.ReadyToInstall
  ) {
    setState(null);
    return;
  }

  console.log(
    "INSTALLING :",
    updateProgress.readyToInstallBinary,
    updateProgress.readyToInstallSignatureFile
  );

  setState({
    state: AppUpdateStage.Installing
  });

  try {
    // validating certificate before start
    if (
      (await ValidateFileOpenSSLCertificate(
        updateProgress.readyToInstallBinary,
        updateProgress.readyToInstallSignatureFile
      )) !== true
    ) {
      setState({
        state: AppUpdateStage.Error,
        error: "Unable to start update: signature verification error"
      });
      return;
    }

    // START INSTALL
    if (Platform() === PlatformEnum.Windows) {
      let spawn = require("child_process").spawn;
      spawn(updateProgress.readyToInstallBinary);
    } else {
      throw new Error(
        "Automatic updates installation is not supported for this platform"
      );
    }
  } catch (err) {
    setState({
      state: AppUpdateStage.Error,
      error: "Unable to start update: " + err
    });
  }
  return true;
}

export async function Upgrade(latestVersionInfo) {
  if (!latestVersionInfo) {
    console.error("Upgrade skipped: no information about latest version");
    return null;
  }

  DownloadUpdateCancelled = false;

  try {
    if (!latestVersionInfo.generic || !latestVersionInfo.generic.downloadLink) {
      // if not enough information about update - just open website (applications download page)
      require("electron").shell.openExternal(config.URLApps);
    } else {
      // Start downloading an update binary
      let onDownloadProgress = function(contentLength, received) {
        setState({
          state: AppUpdateStage.Downloading,
          downloadStatus: {
            contentLength: contentLength,
            downloaded: received
          }
        });
      };

      setState({
        state: AppUpdateStage.Downloading
      });
      // DOWNLOAD SIGNATURE
      let downloadedSignatureFile = null;
      try {
        downloadedSignatureFile = await Download(
          latestVersionInfo.generic.signature
        );
      } catch (error) {
        // failed to download (or failed or save)
        setState({
          state: AppUpdateStage.Error,
          error: "Failed to download update signature: " + error
        });
      }

      if (DownloadUpdateCancelled) {
        setState({
          updateState: AppUpdateStage.CancelledDownload
        });
        return;
      }

      // DOWNLOAD BINARY
      let downloadedFile = null;
      try {
        downloadedFile = await Download(
          latestVersionInfo.generic.downloadLink,
          onDownloadProgress
        );
      } catch (error) {
        // failed to download (or failed or save) binary
        setState({
          state: AppUpdateStage.Error,
          error: "Failed to download update: " + error
        });
      }

      if (DownloadUpdateCancelled) {
        setState({
          updateState: AppUpdateStage.CancelledDownload
        });
        return;
      }

      // checking downloaded binary signature
      setState({
        state: AppUpdateStage.CheckingSignature
      });
      if (
        await ValidateFileOpenSSLCertificate(
          downloadedFile,
          downloadedSignatureFile
        )
      ) {
        // signature ok - ready to install
        setState({
          state: AppUpdateStage.ReadyToInstall,
          readyToInstallBinary: downloadedFile,
          readyToInstallSignatureFile: downloadedSignatureFile
        });
      } else {
        // signature check error
        setState({
          state: AppUpdateStage.Error,
          error: "Update failed: signature verification error"
        });
      }
    }
  } catch (e) {
    console.error(e);
  }
}

async function Download(link, onProgress) {
  return await new Promise((resolve, reject) => {
    try {
      var path = require("path");
      var fs = require("fs");
      var https = require("https");

      let filename = link.substring(link.lastIndexOf("/") + 1);
      let outFilePath = path.join(os.tmpdir(), filename);

      var file = fs.createWriteStream(outFilePath);

      let request = https
        .get(link, res => {
          if (res.statusCode != 200) {
            throw new Error(`StatusCode: ${res.statusCode}`);
          }
          // pipe to file
          res.pipe(file);

          let contentLength = res.headers["content-length"];
          let received = 0;

          // listening for progress event
          res.on("data", d => {
            if (DownloadUpdateCancelled) {
              request.abort();
              return;
            }
            received += d.length;
            if (onProgress) onProgress(contentLength, received);
          });

          // finished
          res.on("end", () => {
            resolve(outFilePath);
          });
        })
        .on("error", e => {
          reject(e);
        });
    } catch (e) {
      reject(e);
    }
  });
}
