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

import Vue from "vue";
import App from "./App.vue";
import router from "./router";
import store from "./store";

Vue.config.productionTip = false;

import { Platform, PlatformEnum } from "@/platform/platform";
changeUIStyle(Platform());

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");

function changeUIStyle(platform) {
  switch (platform) {
    case PlatformEnum.Windows:
      console.log("Windows UI"); // TODO: comment this lines when building for non-Windows platform
      require("@/assets/fonts/fonts_windows.scss");
      require("@/components/scss/platform/windows.scss");
      break;
    case PlatformEnum.macOS:
      console.log("macOS UI"); // TODO: comment this lines when building for non-macOS platform
      require("@/assets/fonts/fonts_macos.scss");
      require("@/components/scss/platform/macos.scss");
      break;
    default:
      console.log("Linux UI"); // TODO: comment this lines when building for non-Linux platform
      require("@/assets/fonts/fonts_linux.scss");
      require("@/components/scss/platform/linux.scss");
  }
}

const electron = window.require("electron");
const { ipcRenderer } = electron;
ipcRenderer.on("change-view-request", (event, arg) => {
  router.push(arg);
});

// After initialized, ask main thread about initial route
async function getInitRouteArgs() {
  return await ipcRenderer.invoke("renderer-request-ui-initial-route-args");
}
setTimeout(async () => {
  let initRouteArgs = await getInitRouteArgs();
  if (initRouteArgs != null) router.push(initRouteArgs);
}, 0);

/*
ipcRenderer.on("change-ui-style", (event, platform) => {
  changeUIStyle(platform);
});*/
