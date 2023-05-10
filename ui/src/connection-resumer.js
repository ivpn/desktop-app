//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

import store from "@/store";
import daemonClient from "./daemon-client";
import { PauseStateEnum } from "@/store/types";

let resumeCheckerInterval = null;
export function InitConnectionResumer() {
  store.subscribe((mutation) => {
    try {
      if (
        mutation.type === "uiState/pauseConnectionTill" ||
        mutation.type === "vpnState/pauseState"
      ) {
        if (resumeCheckerInterval != null) clearInterval(resumeCheckerInterval);
        resumeCheckerInterval = null;

        if (store.state.vpnState.pauseState == PauseStateEnum.Resumed) {
          return;
        }

        const pauseTill = store.state.uiState.pauseConnectionTill;
        // We don't use 'setTimeout()' here because it doesn't function correctly when the computer goes to sleep on some systems.
        // Instead, we are checking the time every second and resuming the connection when the time is up.
        resumeCheckerInterval = setInterval(() => {
          if (
            store.state.vpnState.pauseState !== PauseStateEnum.Paused ||
            !pauseTill ||
            new Date() >= pauseTill
          ) {
            clearInterval(resumeCheckerInterval);
            resumeCheckerInterval = null;
            resumeConnection();
          }
        }, 1000);
      }
    } catch (e) {
      console.error("Error in InitConnectionResumer handler", e);
    }
  });
}

function resumeConnection() {
  try {
    console.log("Resuming connection");
    daemonClient.ResumeConnection();
  } catch (e) {
    console.error("Failed to resume connection:", e);
  }
}
