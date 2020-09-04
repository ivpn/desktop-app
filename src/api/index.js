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
      let isRealLocation = isRealGeoLocation();

      const request = net.request("https://api.ivpn.net/v4/geo-lookup");
      log.debug("API: 'geo-lookup' ...");
      store.commit("isRequestingLocation", true);

      request.on("response", response => {
        if (response.statusCode != API_SUCCESS) {
          // save result in store
          store.commit("location", null);
          const error = new Error(
            `API 'geo-lookup' error (code:${response.statusCode})`
          );
          log.error(`API ERROR: 'geo-lookup' ${error}`);
          store.commit("isRequestingLocation", false);
          reject(error);
        }

        response.on("data", chunk => {
          const location = JSON.parse(`${chunk}`);

          if (isRealLocation != isRealGeoLocation()) {
            const error = new Error(
              "Unable to save geo-lookup result (connected state changed)"
            );
            log.error(`API ERROR: 'geo-lookup' ${error}`);
            store.commit("isRequestingLocation", false);
            reject(error);
            return;
          }

          location.isRealLocation = isRealLocation;

          // save result in store
          store.commit("location", location);
          log.debug("API: 'geo-lookup' success.");
          store.commit("isRequestingLocation", false);
          resolve(location);
        });
      });

      request.on("error", error => {
        // save result in store
        store.commit("location", null);
        log.error(`API ERROR: 'geo-lookup' ${error}`);
        store.commit("isRequestingLocation", false);
        reject(new Error(error));
      });

      request.end();
    });

    return retPromise;
  }
};

function isRealGeoLocation() {
  return (
    store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED ||
    store.state.vpnState.pauseState === PauseStateEnum.Paused
  );
}
