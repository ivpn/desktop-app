<template>
  <div class="flexColumn">
    <div class="settingsTitle flexRow">SPLIT TUNNEL SETTINGS</div>

    <div class="param">
      <input
        ref="checkboxIsSTEnabled"
        type="checkbox"
        id="isSTEnabled"
        v-model="isSTEnabled"
      />
      <label class="defColor" for="isSTEnabled">Split Tunnel (Beta) </label>
    </div>
    <div class="fwDescription" style="margin-bottom: 0px">
      Exclude traffic from specific applications from being routed through the
      VPN
    </div>

    <div>
      <!-- functionality description: LINUX -->
      <div
        v-if="isLinux"
        class="fwDescription"
        style="margin-top: 0px; margin-bottom: 0px"
      >
        <span class="settingsGrayLongDescriptionFont" style="font-weight: bold"
          >Warning:</span
        >
        Already running applications can not use Split Tunneling.
        <br />
        <span class="settingsGrayLongDescriptionFont" style="font-weight: bold"
          >Warning:</span
        >
        Some applications (e.g. Web browsers) need to be closed before launching
        them in the Split Tunneling environment. Otherwise, it might not be
        excluded from the VPN tunnel.
      </div>
      <!-- functionality description: WINDOWS -->
      <div v-else>
        <div class="fwDescription" style="margin-top: 0px; margin-bottom: 0px">
          <span
            class="settingsGrayLongDescriptionFont"
            style="font-weight: bold"
            >Warning:</span
          >
          When adding a running application, any connections already established
          by the application will continue to be routed through the VPN tunnel
          until the TCP connection/s are reset or the application is restarted
        </div>
        <div class="fwDescription" style="margin-top: 0px">
          For more information refer to the
          <button class="link" v-on:click="onLearnMoreLink">
            Split Tunnel Uses and Limitations
          </button>
          webpage
        </div>
      </div>
    </div>

    <!-- APPS -->
    <div style="height: 100%">
      <!-- HEADER: Applications -->
      <div class="flexRow" style="margin-top: 12px; margin-bottom: 12px">
        <div
          class="flexRowRestSpace settingsBoldFont settingsDefaultTextColor"
          style="margin-top: 0px; margin-bottom: 0px; white-space: nowrap"
        >
          {{ textApplicationsHeader }}
        </div>

        <!-- ADD APP BUTTON -->
        <div>
          <button
            class="settingsButton opacityOnHoverLight"
            style="min-width: 156px"
            v-on:click="showAddApplicationPopup(true)"
          >
            {{ textAddAppButton }}
          </button>
        </div>
      </div>

      <div class="horizontalLine" />
      <div class="flexRow" style="position: relative">
        <!-- Configured apps view -->

        <!-- No applications in Split Tunnel configuration -->
        <div
          v-if="isNoConfiguredApps"
          style="
            text-align: center;
            width: 100%;
            margin-top: 50px;
            padding: 50px;
          "
        >
          <div class="settingsGrayTextColor">
            {{ textNoAppInSplittunConfig }}
          </div>
        </div>

        <!-- Configured apps list -->
        <div
          v-if="!isShowAppAddPopup && !isNoConfiguredApps"
          :style="appsListStyle"
        >
          <spinner
            :loading="isLoadingAllApps"
            style="
              position: absolute;
              background: transparent;
              width: 100%;
              height: 100%;
            "
          />

          <div
            v-for="app of filteredApps"
            v-bind:key="app.RunningApp ? app.RunningApp.Pid : app.AppBinaryPath"
          >
            <div class="flexRow grayedOnHover" style="padding-top: 4px">
              <!-- APP INFO  -->
              <binaryInfoControl :app="app" style="width: 100%" />
              <!-- APP REMOVE BUTTON -->
              <div>
                <button
                  class="noBordersBtn opacityOnHover"
                  v-on:click="removeApp(app)"
                  style="pointer-events: auto"
                  title="Remove"
                >
                  <img width="24" height="24" src="@/assets/minus.svg" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- SELECT apps 'popup' view -->
        <transition name="fade-super-quick" mode="out-in">
          <div v-if="isShowAppAddPopup" class="appsSelectionPopup">
            <div>
              <div class="flexRow" style="margin-bottom: 10px">
                <div class="flexRowRestSpace settingsGrayTextColor">
                  {{ textAddAppFromInstalledAppsHeader }}
                </div>

                <button
                  class="noBordersBtn opacityOnHoverLight settingsGrayTextColor"
                  style="pointer-events: auto"
                  v-on:click="showAddApplicationPopup(false)"
                >
                  CANCEL
                </button>
              </div>

              <!-- filter -->
              <input
                ref="installedAppsFilterInput"
                id="filter"
                class="styled"
                placeholder="Search for app"
                v-model="filterAppsToAdd"
                v-bind:style="{
                  backgroundImage: 'url(' + searchImageInstalledApps + ')',
                }"
                style="margin: 0px; margin-bottom: 10px"
              />
              <div class="horizontalLine" />

              <!--all apps-->
              <div
                style="
                  overflow: auto;
                  position: relative;
                  height: 320px;
                  max-height: 320px;
                "
              >
                <!-- No applications that are fit the filter -->
                <div
                  v-if="!filteredAppsToAdd || filteredAppsToAdd.length == 0"
                  style="text-align: center; width: 100%; margin-top: 100px"
                >
                  <div class="settingsGrayTextColor">
                    No applications that are fit the filter:
                  </div>
                  <div>
                    '<span
                      class="settingsGrayTextColor"
                      style="
                        display: inline-block;
                        font-weight: bold;
                        overflow: hidden;
                        white-space: nowrap;
                        text-overflow: ellipsis;
                        max-width: 300px;
                      "
                      >{{ filterAppsToAdd }}</span
                    >'
                  </div>
                </div>

                <div
                  v-else
                  v-for="app of filteredAppsToAdd"
                  v-bind:key="app.AppBinaryPath"
                >
                  <div
                    v-on:click="addApp(app.AppBinaryPath)"
                    class="flexRow grayedOnHover"
                    style="padding-top: 4px"
                  >
                    <binaryInfoControl :app="app" style="width: 100%" />
                  </div>
                </div>
              </div>
              <div style="height: 100%" />
              <div class="horizontalLine" />

              <div>
                <button
                  class="settingsButton flexRow grayedOnHover"
                  style="
                    margin-top: 10px;
                    margin-bottom: 10px;
                    height: 40px;
                    width: 100%;
                  "
                  v-on:click="onManualAddNewApplication"
                >
                  <div class="flexRowRestSpace"></div>
                  <div class="flexRow">
                    <img
                      width="24"
                      height="24"
                      style="margin: 8px"
                      src="@/assets/plus.svg"
                    />
                  </div>
                  <div class="flexRow settingsGrayTextColor">
                    {{ textAddAppManuallyButton }}
                  </div>
                  <div class="flexRowRestSpace"></div>
                </button>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </div>

    <!-- FOOTER -->

    <div style="position: sticky; bottom: 20px">
      <div class="horizontalLine" />

      <div class="flexRow" style="margin-top: 15px">
        <div class="flexRowRestSpace" />
        <button
          class="settingsButton opacityOnHoverLight"
          v-on:click="onResetToDefaultSettings"
          style="white-space: nowrap"
        >
          Reset to default settings
        </button>
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

import { isStrNullOrEmpty } from "@/helpers/helpers";
import { Platform, PlatformEnum } from "@/platform/platform";

import Image_search_windows from "@/assets/search-windows.svg";
import Image_search_macos from "@/assets/search-macos.svg";
import Image_search_linux from "@/assets/search-linux.svg";

import binaryInfoControl from "@/components/controls/control-app-binary-info.vue";

import spinner from "@/components/controls/control-spinner.vue";

function processError(e) {
  console.error(e);
  sender.showMessageBox({
    type: "error",
    buttons: ["OK"],
    message: e.toString(),
  });
}

let timerBackgroundCheckOfStatus = 0;

export default {
  components: {
    spinner,
    binaryInfoControl,
  },

  data: function () {
    return {
      isLoadingAllApps: false,
      isShowAppAddPopup: false,

      filterAppsToAdd: "",

      // allInstalledApps [] - an array of configured application path's
      // Type (AppInfo):
      //    AppName       string
      //    AppGroup      string // optional
      //    AppIcon       string - base64 icon of the executable binary
      //    AppBinaryPath string - The unique parameter describing an application
      //                    Windows: absolute path to application binary
      //                    Linux: program to execute, possibly with arguments.
      allInstalledApps: null,
      allInstalledAppsHashed: {},

      // []AppInfoEx -  configured (running) apps
      // Type:
      //  AppInfo fields
      //  + RunningApp: (Linux: info about running apps in ST environment):
      //      RunningApp.Pid     int
      //      RunningApp.Ppid    int        // The PID of the parent of this process.
      //      RunningApp.Cmdline string
      //      RunningApp.Exe     string     // The actual pathname of the executed command
      //      RunningApp.ExtIvpnRootPid int // PID of the known parent process registered by AddPid() function
      //      RunningApp.ExtModifiedCmdLine string
      appsToShow: null,
    };
  },

  async mounted() {
    // show base information about splitted apps immediately
    //this.updateAppsToShow();

    let allApps = null;
    try {
      this.isLoadingAllApps = true;
      allApps = await sender.GetInstalledApps();
      await sender.SplitTunnelGetStatus();
    } finally {
      this.isLoadingAllApps = false;
    }

    if (allApps) {
      // create a list of hashed appinfo (by app path)
      allApps.forEach((appInfo) => {
        this.allInstalledAppsHashed[appInfo.AppBinaryPath.toLowerCase()] =
          appInfo;
      });

      this.allInstalledApps = allApps;
    }

    // now we are able to update information about splitted apps
    this.updateAppsToShow();
  },

  watch: {
    STConfig() {
      if (this.$refs.checkboxIsSTEnabled) {
        // we have to update checkbox manually
        this.$refs.checkboxIsSTEnabled.checked =
          this.$store.state.vpnState.splitTunnelling.IsEnabled;
      }

      this.updateAppsToShow();

      // if there are running apps - start requesting ST status
      this.startBackgroundCheckOfStatus();
    },
  },

  methods: {
    isRunningAppsAvailable() {
      let stStatus = this.$store.state.vpnState.splitTunnelling;
      return (
        Array.isArray(stStatus.RunningApps) && stStatus.RunningApps.length > 0
      );
    },
    stopBackgroundCheckOfStatus() {
      if (timerBackgroundCheckOfStatus != 0) {
        clearInterval(timerBackgroundCheckOfStatus);
        timerBackgroundCheckOfStatus = 0;
      }
    },
    startBackgroundCheckOfStatus() {
      if (Platform() !== PlatformEnum.Linux) return;
      // timer already started
      if (timerBackgroundCheckOfStatus) return;

      if (this.isRunningAppsAvailable()) {
        timerBackgroundCheckOfStatus = setInterval(() => {
          if (
            !this.isRunningAppsAvailable() ||
            this.$store.state.uiState.currentSettingsViewName != "splittunnel"
          ) {
            this.stopBackgroundCheckOfStatus();
            return;
          }
          try {
            sender.SplitTunnelGetStatus();
          } catch (e) {
            console.error(e);
          }
        }, 5000);
      }
    },
    onLearnMoreLink: () => {
      sender.shellOpenExternal(
        `https://www.ivpn.net/knowledgebase/general/split-tunnel-uses-and-limitations`
      );
    },

    updateAppsToShow() {
      // preparing list of apps to show (AppInfo fields + RunningApp)
      let appsToShowTmp = [];

      try {
        let splitTunnelling = this.$store.state.vpnState.splitTunnelling;
        if (Platform() === PlatformEnum.Linux) {
          // Linux:
          let runningApps = splitTunnelling.RunningApps;
          runningApps.forEach((runningApp) => {
            // check if we can get info from the installed apps list
            let cmdLine = "";
            if (
              runningApp.ExtModifiedCmdLine &&
              runningApp.ExtModifiedCmdLine.length > 0
            ) {
              cmdLine = runningApp.ExtModifiedCmdLine.toLowerCase();
            } else {
              cmdLine = runningApp.Cmdline.toLowerCase();
            }

            let knownApp = this.allInstalledAppsHashed[cmdLine];
            // Do not show child processes (child processes of known root PID)
            if (
              runningApp.ExtIvpnRootPid > 0 &&
              runningApp.ExtIvpnRootPid !== runningApp.Pid
            )
              return;
            if (!knownApp)
              // app is not found in 'installed apps list'
              appsToShowTmp.push({
                AppBinaryPath: cmdLine,
                AppName: cmdLine,
                AppGroup: null,
                RunningApp: runningApp,
              });
            else {
              // app is found in 'installed apps list'
              // use 'Object.assign' to not update data in 'this.allInstalledAppsHashed'
              knownApp = Object.assign({}, knownApp);
              knownApp.RunningApp = runningApp;
              appsToShowTmp.push(Object.assign({}, knownApp));
            }
          });
        } else {
          // Windows:
          let configApps = splitTunnelling.SplitTunnelApps;
          configApps.forEach((appPath) => {
            if (!appPath) return;
            // check if we can get info from the installed apps list
            let knownApp = this.allInstalledAppsHashed[appPath.toLowerCase()];
            if (!knownApp) {
              let file = appPath.split("\\").pop().split("/").pop();
              let folder = appPath.substring(0, appPath.length - file.length);
              // app is not found in 'installed apps list'
              appsToShowTmp.push({
                AppBinaryPath: appPath,
                AppName: file,
                AppGroup: folder,
              });
            } else {
              // app is found in 'installed apps list'
              // use 'Object.assign' to not update data in 'this.allInstalledAppsHashed'
              appsToShowTmp.push(Object.assign({}, knownApp));
            }
          });
        }
      } catch (e) {
        console.error(e);
      }

      // sorting the list
      appsToShowTmp.sort(function (a, b) {
        if (a.RunningApp && b.RunningApp) {
          if (
            a.RunningApp.ExtIvpnRootPid > 0 &&
            b.RunningApp.ExtIvpnRootPid === 0
          )
            return -1;
          if (
            a.RunningApp.ExtIvpnRootPid === 0 &&
            b.RunningApp.ExtIvpnRootPid > 0
          )
            return 1;

          if (a.RunningApp.Pid < b.RunningApp.Pid) return -1;
          if (a.RunningApp.Pid > b.RunningApp.Pid) return 1;
        }

        if (a.AppName && b.AppName) {
          let app1 = a.AppName.toUpperCase();
          let app2 = b.AppName.toUpperCase();
          if (app1 > app2) return 1;
          if (app1 < app2) return -1;
        } else {
          if (a.AppName > b.AppName) return 1;
          if (a.AppName < b.AppName) return -1;
        }
        return 0;
      });

      this.appsToShow = appsToShowTmp;
    },

    showAddApplicationPopup(isShow) {
      this.resetFilters();

      if (isShow === true) {
        this.filterAppsToAdd = "";
        let appsToAdd = this.filteredAppsToAdd;
        if (!appsToAdd || appsToAdd.length == 0) {
          // if no info about all installed applications - show dialog to manually select binary
          this.onManualAddNewApplication();
          return;
        }
        this.isShowAppAddPopup = true;
        setTimeout(() => {
          try {
            this.$refs.installedAppsFilterInput.focus();
          } catch (e) {
            console.error(e);
          }
        }, 0);
      } else this.isShowAppAddPopup = false;
    },

    async onManualAddNewApplication() {
      try {
        let dlgFilters = [];
        if (Platform() === PlatformEnum.Windows) {
          dlgFilters = [
            { name: "Executables", extensions: ["exe"] },
            { name: "All files", extensions: ["*"] },
          ];
        } else {
          dlgFilters = [{ name: "All files", extensions: ["*"] }];
        }

        var diagConfig = {
          properties: ["openFile"],
          filters: dlgFilters,
        };
        var ret = await sender.showOpenDialog(diagConfig);
        if (!ret || ret.canceled || ret.filePaths.length == 0) return;

        await sender.SplitTunnelAddApp(ret.filePaths[0]);
      } catch (e) {
        processError(e);
      } finally {
        this.showAddApplicationPopup(false);
      }
    },

    async removeApp(app) {
      try {
        if (!app) return;
        if (app.RunningApp)
          await sender.SplitTunnelRemoveApp(
            app.RunningApp.Pid,
            app.AppBinaryPath
          );
        else await sender.SplitTunnelRemoveApp(0, app.AppBinaryPath);
      } catch (e) {
        processError(e);
      } finally {
        this.showAddApplicationPopup(false);
      }
    },

    async addApp(appPath) {
      try {
        await sender.SplitTunnelAddApp(appPath);
      } catch (e) {
        processError(e);
      } finally {
        this.showAddApplicationPopup(false);
      }
    },

    async onResetToDefaultSettings() {
      let actionNo = sender.showMessageBoxSync({
        type: "question",
        buttons: ["Yes", "Cancel"],
        message: "Reset all settings to default values",
        detail: `Are you sure you want to reset the Split Tunnel configuration for all applications?`,
      });
      if (actionNo == 1) return;

      this.resetFilters();
      await sender.SplitTunnelSetConfig(false, true);
    },

    resetFilters: function () {
      this.filterAppsToAdd = "";
    },
  },

  computed: {
    textApplicationsHeader: function () {
      if (Platform() === PlatformEnum.Linux) return "Launched applications";
      return "Applications";
    },

    textNoAppInSplittunConfig: function () {
      return "No applications in Split Tunnel configuration";
    },

    textAddAppButton: function () {
      if (Platform() === PlatformEnum.Linux) return "Launch application...";
      return "Add application...";
    },
    textAddAppFromInstalledAppsHeader: function () {
      if (Platform() === PlatformEnum.Linux)
        return "Launch application in Split Tunnel configuration";
      return "Add application to Split Tunnel configuration";
    },
    textAddAppManuallyButton: function () {
      if (Platform() === PlatformEnum.Linux)
        return "Launch application manually...";
      return "Add application manually ...";
    },

    appsListStyle: function () {
      // TODO: avoid hardcoding element height.
      let height = 244;
      if (Platform() === PlatformEnum.Linux) height = 268;

      return `\
            overflow: auto;\
            width: 100%;\
            position: relative;\
            height: ${height}px;\
            min-height: ${height}px;\
            max-height: ${height}px;\
          `;
    },
    isLinux: function () {
      return Platform() === PlatformEnum.Linux;
    },

    isSTEnabled: {
      get() {
        return this.$store.state.vpnState.splitTunnelling.IsEnabled;
      },
      set(value) {
        (async function () {
          await sender.SplitTunnelSetConfig(value);
        })();
      },
    },

    // needed for 'watch'
    STConfig: function () {
      return this.$store.state.vpnState.splitTunnelling;
    },

    isNoConfiguredApps: function () {
      if (
        this.isLoadingAllApps == false &&
        (!this.appsToShow || this.appsToShow.length == 0)
      )
        return true;
      return false;
    },

    filteredApps: function () {
      return this.appsToShow;
    },

    filteredAppsToAdd: function () {
      let retInstalledApps = [];
      if (this.allInstalledApps)
        retInstalledApps = Object.assign(
          retInstalledApps,
          this.allInstalledApps
        );

      // filter: exclude already configured apps (not a running apps)
      // from the list installed apps
      let confAppsHashed = {};
      this.appsToShow.forEach((appInfo) => {
        confAppsHashed[appInfo.AppBinaryPath.toLowerCase()] = appInfo;
      });

      let funcFilter = function (appInfo) {
        let confApp = confAppsHashed[appInfo.AppBinaryPath.toLowerCase()];
        if (confApp && (!confApp.RunningApp || !confApp.RunningApp.Pid))
          return false;
        return true;
      };
      retInstalledApps = retInstalledApps.filter((appInfo) =>
        funcFilter(appInfo)
      );

      // filter: default (filtering apps according to user input)
      let filter = this.filterAppsToAdd.toLowerCase();
      if (filter && filter.length > 0) {
        let funcFilter = function (appInfo) {
          return (
            appInfo.AppName.toLowerCase().includes(filter) ||
            appInfo.AppGroup.toLowerCase().includes(filter)
          );
        };

        retInstalledApps = retInstalledApps.filter((appInfo) =>
          funcFilter(appInfo)
        );
      }

      return retInstalledApps;
    },

    searchImageInstalledApps: function () {
      if (!isStrNullOrEmpty(this.filterAppsToAdd)) return null;

      switch (Platform()) {
        case PlatformEnum.Windows:
          return Image_search_windows;
        case PlatformEnum.macOS:
          return Image_search_macos;
        default:
          return Image_search_linux;
      }
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/platform/base.scss";

.grayedOnHover:hover {
  background: rgba(100, 100, 100, 0.2);
  border-radius: 2px;
}

.opacityOnHover:hover {
  opacity: 0.6;
}

.opacityOnHoverLight:hover {
  opacity: 0.8;
}

.defColor {
  @extend .settingsDefaultTextColor;
}

div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
  max-width: 425px;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

button.link {
  @extend .noBordersTextBtn;
  @extend .settingsLinkText;
  font-size: inherit;
}
label {
  margin-left: 1px;
  font-weight: 500;
}

input#filter {
  margin-left: 20px;
  margin-right: 20px;
  margin-top: 0px;
  margin-bottom: 0px;
  height: auto;

  background-position: 97% 50%; //right
  background-repeat: no-repeat;
}

$popup-background: var(--background-color);
$shadow: 0px 3px 12px rgba(var(--shadow-color-rgb), var(--shadow-opacity));
.appsSelectionPopup {
  position: absolute;
  z-index: 1;

  height: 100%;
  width: 100%;

  padding: 15px;
  height: 435px; //calc(100% + 140px);
  width: calc(100% + 10px);
  left: -20px;
  top: -180px;

  border-width: 1px;
  border-style: solid;
  border-color: $popup-background;

  //border-radius: 8px;
  background-color: $popup-background;
  box-shadow: $shadow;
}
</style>
