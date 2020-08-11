//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui-beta
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

const { net } = require("electron");
const log = require("electron-log");

import { API_SUCCESS } from "@/api/statuscode";
import store from "@/store";
import { VpnStateEnum, PauseStateEnum } from "@/store/types";

export default {
  GeoLookup: async function() {
    const retPromise = new Promise((resolve, reject) => {
      if (
        store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED &&
        store.state.vpnState.pauseState !== PauseStateEnum.Paused
      ) {
        reject(new Error("Unable to request geo-lookup in connected state"));
        return;
      }
      const request = net.request("https://api.ivpn.net/v4/geo-lookup");
      log.debug("API: 'geo-lookup' ...");
      request.on("response", response => {
        if (response.statusCode != API_SUCCESS) {
          // save result in store
          store.commit("location", null);
          const error = new Error(
            `API 'geo-lookup' error (code:${response.statusCode})`
          );
          log.error(`API ERROR: 'geo-lookup' ${error}`);
          reject(error);
        }
        response.on("data", chunk => {
          const location = JSON.parse(`${chunk}`);

          if (
            store.state.vpnState.connectionState !==
              VpnStateEnum.DISCONNECTED &&
            store.state.vpnState.pauseState !== PauseStateEnum.Paused
          ) {
            const error = new Error(
              "Unable to save geo-lookup result in connected state"
            );
            log.error(`API ERROR: 'geo-lookup' ${error}`);
            reject(error);
            return;
          }
          // save result in store
          store.commit("location", location);
          log.debug("API: 'geo-lookup' success.");
          resolve(location);
        });
      });
      request.on("error", error => {
        // save result in store
        store.commit("location", null);
        log.error(`API ERROR: 'geo-lookup' ${error}`);
        reject(new Error(error));
      });
      request.end();
    });

    return retPromise;
  }
};
