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
