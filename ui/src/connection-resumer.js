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

let resumerTimer = null;
export function InitConnectionResumer() {
  store.subscribe((mutation) => {
    try {
      if (
        mutation.type === "uiState/pauseConnectionTill" ||
        mutation.type === "vpnState/pauseState"
      ) {
        if (resumerTimer != null) clearTimeout(resumerTimer);
        resumerTimer = null;

        if (store.state.vpnState.pauseState == PauseStateEnum.Resumed) {
          return;
        }

        const pauseTill = store.state.uiState.pauseConnectionTill;
        if (pauseTill != null) {
          const timeDiff = pauseTill - new Date();
          if (timeDiff > 0) {
            resumerTimer = setTimeout(() => {
              resumeConnection();
            }, timeDiff);
          }
        }
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
