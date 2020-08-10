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

// TODO: just for testing
const electron = window.require("electron");
const { ipcRenderer } = electron;
ipcRenderer.on("change-view-request", (event, arg) => {
  router.push(arg);
});
/*
ipcRenderer.on("change-ui-style", (event, platform) => {
  changeUIStyle(platform);
});*/
