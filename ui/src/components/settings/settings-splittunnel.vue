<template>
  <div class="flexColumn">
    <div class="settingsTitle flexRow">SPLIT TUNNEL SETTINGS</div>

    <div class="param">
      <input type="checkbox" id="isSTEnabled" v-model="isSTEnabled" />
      <label class="defColor" for="isSTEnabled">Split Tunnel (Beta) </label>
    </div>
    <div class="fwDescription" style="margin-bottom: 0px">
      Exclude traffic from specific applications from being routed through the
      VPN
    </div>
    <div class="fwDescription" style="margin-top: 0px; margin-bottom: 0px">
      <span class="settingsGrayLongDescriptionFont" style="font-weight: bold;"
        >Warning:</span
      >
      When adding a running application, any connections already established by
      the application will continue to be routed through the VPN tunnel until
      the TCP connection/s are reset or the application is restarted
    </div>

    <div class="fwDescription" style="margin-top: 0px">
      For more information refer to the
      <button class="link" v-on:click="onLearnMoreLink">
        Split Tunnel Uses and Limitations
      </button>
      webpage
    </div>

    <!-- APPS -->
    <div style="height:100%;">
      <div class="flexRow" style="margin-top: 12px; margin-bottom:12px">
        <div
          class="flexRowRestSpace settingsBoldFont settingsDefaultTextColor"
          style="margin-top: 0px; margin-bottom:0px; white-space: nowrap;"
        >
          Applications
        </div>

        <!-- CONFIGURED APPS FILETR -->

        <!--
        <input
          id="filter"
          class="styled"
          placeholder="Search for app"
          v-model="filter"
          v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
        />
        -->

        <div>
          <button
            class="settingsButton opacityOnHoverLight"
            style="min-width: 156px"
            v-on:click="showAddApplicationPopup(true)"
          >
            {{ addAppButtonText }}
          </button>
        </div>
      </div>

      <div class="horizontalLine" />
      <div class="flexRow" style="position: relative;">
        <!-- Configured apps view -->

        <!-- No applications in Split Tunnel configuration -->
        <div
          v-if="isNoConfiguredApps"
          style="text-align: center; width:100%; margin-top: 50px; padding: 50px;"
        >
          <div class="settingsGrayTextColor">
            No applications in Split Tunnel configuration
          </div>
        </div>

        <!-- No applications that are fit the filter -->
        <div
          v-if="!isNoConfiguredApps && isNoConfiguredAppsMatchFilter"
          style="text-align: center; width:100%; padding: 50px;"
        >
          <div class="settingsGrayTextColor">
            No applications in Split Tunnel configuration that are fit the
            filter:
          </div>
          <div>
            '<span
              class="settingsGrayTextColor"
              style="display: inline-block; font-weight: bold; overflow: hidden; white-space: nowrap;  text-overflow: ellipsis; max-width: 300px"
              >{{ filter }}</span
            >'
          </div>
        </div>

        <!-- Configured apps list -->
        <div
          v-if="
            !isShowAppAddPopup &&
              !isNoConfiguredApps &&
              !isNoConfiguredAppsMatchFilter
          "
          style="overflow: auto;
          width: 100%;
          position: relative;
          height:244px; min-height:244px; max-height:244px;"
        >
          <spinner
            :loading="isLoadingAllApps"
            style="position: absolute; background: transparent; width: 100%; height: 100%;"
          />

          <div v-for="app of filteredApps" v-bind:key="app.AppBinaryPath">
            <div class="flexRow grayedOnHover" style="padding-top: 4px;">
              <!-- APP INFO  -->
              <binaryInfoControl :app="app" style="width: 100%" />

              <!-- APP BUTTONS -->
              <div>
                <button
                  class="noBordersBtn opacityOnHover"
                  v-if="app.isSplitted"
                  v-on:click="removeApp(app.AppBinaryPath)"
                  style="pointer-events: auto;"
                  title="Remove"
                >
                  <img width="24" height="24" src="@/assets/minus.svg" />
                </button>

                <button
                  class="noBordersBtn opacityOnHover"
                  v-else
                  v-on:click="addApp(app.AppBinaryPath)"
                  style="pointer-events: auto;"
                >
                  <img width="24" height="24" src="@/assets/plus.svg" />
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
                  Add application to Split Tunnel configuration
                </div>

                <button
                  class="noBordersBtn opacityOnHoverLight settingsGrayTextColor"
                  style="pointer-events: auto;"
                  v-on:click="showAddApplicationPopup(false)"
                >
                  CANCEL
                </button>
              </div>

              <!-- filter -->
              <input
                id="filter"
                class="styled"
                placeholder="Search for app"
                v-model="filterAppsToAdd"
                v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
                style="margin: 0px;  margin-bottom: 10px"
              />
              <div class="horizontalLine" />

              <!--all apps-->
              <div
                style="overflow: auto; position: relative; height: 320px; max-height: 320px"
              >
                <!-- No applications that are fit the filter -->
                <div
                  v-if="!filteredAppsToAdd || filteredAppsToAdd.length == 0"
                  style="text-align: center; width:100%; margin-top: 100px;"
                >
                  <div class="settingsGrayTextColor">
                    No applications that are fit the filter:
                  </div>
                  <div>
                    '<span
                      class="settingsGrayTextColor"
                      style="display: inline-block; font-weight: bold; overflow: hidden; white-space: nowrap;  text-overflow: ellipsis; max-width: 300px"
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
                    style="padding-top: 4px;"
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
                  style="margin-top:10px; margin-bottom:10px; height: 40px; width: 100%"
                  v-on:click="onManuaAddNewApplication"
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
                    Add application manually ...
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

    <div style="position: sticky; bottom: 20px;">
      <div class="horizontalLine" />

      <div class="flexRow" style="margin-top: 15px;">
        <!-- CONFIGURED APPS FILETR -->
        <!--
        <input
          id="filter"
          class="styled flexRow"
          placeholder="Search for configured app"
          v-model="filter"
          v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
          style="margin: 0px; margin-right: 20px"
        /> -->

        <div class="flexRowRestSpace" />
        <button
          class="settingsButton opacityOnHoverLight"
          v-on:click="onResetToDefaultSettings"
          style="white-space: nowrap;"
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

export default {
  components: {
    spinner,
    binaryInfoControl
  },

  data: function() {
    return {
      isLoadingAllApps: false,
      isShowAppAddPopup: false,
      filter: "",
      filterAppsToAdd: "",
      allInstalledApps: null, // []; array of configured application path's (only absolute file path)
      // Heshed info about all available applications.
      //  allAppsHashed[binaryPath] = AppInfo
      // Where the AppInfo object:
      //	  AppBinaryPath string
      //    AppName  string
      //    AppGroup string
      //    AppIcon string
      //    isSplitted (true or (false/null))
      allAppsHashed: {},
      appsToShow: null, // []; array of appInfo
      configAppsHashed: {} // hashed list of splitted apps (needed to avoid duplicates in final list)
    };
  },
  async mounted() {
    // show base information about splitted apps immediately
    //this.updateAppsToShow();

    let allApps = null;
    try {
      this.isLoadingAllApps = true;
      allApps = await sender.GetInstalledApps();
    } finally {
      this.isLoadingAllApps = false;
    }

    if (allApps) {
      // create a list of hashed appinfo (by app path)
      allApps.forEach(appInfo => {
        this.allAppsHashed[appInfo.AppBinaryPath.toLowerCase()] = appInfo;
      });

      this.allInstalledApps = allApps;
    }

    // now we are able to update information about splitted apps
    this.updateAppsToShow();
  },

  watch: {
    STConfig() {
      this.updateAppsToShow();
    }
  },

  methods: {
    onLearnMoreLink: () => {
      sender.shellOpenExternal(
        `https://www.ivpn.net/knowledgebase/general/split-tunnel-uses-and-limitations`
      );
    },

    updateAppsToShow() {
      // 'splitted' applications
      let configApps = this.$store.state.vpnState.splitTunnelling.apps;

      // erase hashed list of configured apps
      this.configAppsHashed = {};

      // prepare information for selected apps: update app info (if exists)
      if (configApps) {
        configApps.forEach(appPath => {
          // use 'Object.assign' to not update data in 'this.allAppsHashed'
          let appInfoConst = this.allAppsHashed[appPath.toLowerCase()];
          let appInfo = {};

          if (!appInfoConst)
            appInfo = { AppBinaryPath: appPath, AppName: null, AppGroup: null };
          else appInfo = Object.assign({}, appInfoConst);

          appInfo.isSplitted = true;
          this.configAppsHashed[appPath.toLowerCase()] = appInfo;
        });
      }

      // apps to show
      let appsInfo = [];

      // show only splitted apps
      for (const [, appInfo] of Object.entries(this.configAppsHashed)) {
        appsInfo.push(appInfo);
      }

      appsInfo.sort(function(a, b) {
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

      this.appsToShow = appsInfo;
    },

    showAddApplicationPopup(isShow) {
      this.resetFilters();

      if (isShow === true) {
        this.filterAppsToAdd = "";
        let appsToAdd = this.filteredAppsToAdd;
        if (!appsToAdd || appsToAdd.length == 0) {
          this.onManuaAddNewApplication();
          return;
        }
        this.isShowAppAddPopup = true;
      } else this.isShowAppAddPopup = false;
    },

    async onManuaAddNewApplication() {
      try {
        var diagConfig = {
          properties: ["openFile"],
          filters: [
            { name: "Executables", extensions: ["exe"] },
            { name: "All files", extensions: ["*"] }
          ]
        };
        var ret = await sender.showOpenDialog(diagConfig);
        if (!ret || ret.canceled || ret.filePaths.length == 0) return;

        var st = this.$store.state.vpnState.splitTunnelling;
        var stApps = [];
        if (st.apps) stApps = Object.assign(stApps, st.apps);

        ret.filePaths.forEach(appPath => {
          if (stApps.includes(appPath) == false) stApps.push(appPath);
        });

        await sender.SplitTunnelSetConfig(st.enabled, stApps);
      } finally {
        this.showAddApplicationPopup(false);
      }
    },

    async removeApp(appPath) {
      var st = this.$store.state.vpnState.splitTunnelling;
      var stApps = [];
      if (st.apps) stApps = Object.assign(stApps, st.apps);

      var indexOfIgnoreCaseFunc = (arr, q) =>
        arr.findIndex(item => q.toLowerCase() === item.toLowerCase());

      var index = indexOfIgnoreCaseFunc(stApps, appPath);
      if (index === -1) return;
      stApps.splice(index, 1);

      // If the application has no AppName info - it means it was added manually
      // In this case, we can remove it from the 'allApps' list
      let appInfo = this.allAppsHashed[appPath.toLowerCase()];
      if (appInfo && !appInfo.AppName) {
        delete this.allAppsHashed[appPath.toLowerCase()];
      }

      await sender.SplitTunnelSetConfig(st.enabled, stApps);
    },

    async addApp(appPath) {
      try {
        var st = this.$store.state.vpnState.splitTunnelling;
        var stApps = [];
        if (st.apps) stApps = Object.assign(stApps, st.apps);

        stApps.push(appPath);

        await sender.SplitTunnelSetConfig(st.enabled, stApps);
      } finally {
        this.showAddApplicationPopup(false);
      }
    },

    async onResetToDefaultSettings() {
      let actionNo = sender.showMessageBoxSync({
        type: "question",
        buttons: ["Yes", "Cancel"],
        message: "Reset all settings to default values",
        detail: `Are you sure you want to reset the Split Tunnel configuration for all applications?`
      });
      if (actionNo == 1) return;

      this.resetFilters();
      await sender.SplitTunnelSetConfig(false, null);
    },

    appsFiletrFunc(filter, appInfo) {
      // file name of binary (without extension)
      let binaryFname = "";
      try {
        binaryFname = appInfo.AppBinaryPath.split("\\")
          .pop()
          .split("/")
          .pop();
        binaryFname = binaryFname.substring(0, binaryFname.lastIndexOf("."));
      } catch (e) {
        console.error(e);
      }

      if (binaryFname && binaryFname.toLowerCase().includes(filter)) {
        return true;
      }

      return (
        appInfo.AppName.toLowerCase().includes(filter) ||
        appInfo.AppGroup.toLowerCase().includes(filter)
      );
    },

    resetFilters: function() {
      this.filter = "";
      this.filterAppsToAdd = "";
    }
  },

  computed: {
    addAppButtonText: function() {
      return "Add application...";
    },

    isSTEnabled: {
      get() {
        return this.$store.state.vpnState.splitTunnelling.enabled;
      },
      async set(value) {
        var st = this.$store.state.vpnState.splitTunnelling;
        await sender.SplitTunnelSetConfig(value, st.apps);
      }
    },

    // needed for 'watch'
    STConfig: function() {
      return this.$store.state.vpnState.splitTunnelling;
    },

    isNoConfiguredApps: function() {
      if (
        this.isLoadingAllApps == false &&
        (!this.appsToShow || this.appsToShow.length == 0)
      )
        return true;
      return false;
    },

    isNoConfiguredAppsMatchFilter: function() {
      if (!this.filter || this.filter == "") return false;

      if (
        this.isLoadingAllApps == false &&
        (!this.filteredApps || this.filteredApps.length == 0)
      )
        return true;
      return false;
    },

    filteredApps: function() {
      if (this.filter == null || this.filter.length == 0)
        return this.appsToShow;

      let filter = this.filter.toLowerCase();
      return this.appsToShow.filter(appInfo =>
        this.appsFiletrFunc(filter, appInfo)
      );
    },

    filteredAppsToAdd: function() {
      let retApps = [];
      if (this.allInstalledApps)
        retApps = Object.assign(retApps, this.allInstalledApps);

      // filtering

      // filter: exclude apps which are already in configuration
      let confAppsHashed = this.configAppsHashed;
      let filterFunc = function(appInfo) {
        if (confAppsHashed[appInfo.AppBinaryPath.toLowerCase()]) return false;
        return true;
      };
      retApps = retApps.filter(appInfo => filterFunc(appInfo));

      // filter: default
      let filter = this.filterAppsToAdd.toLowerCase();
      if (filter && filter.length > 0)
        retApps = retApps.filter(appInfo =>
          this.appsFiletrFunc(filter, appInfo)
        );

      return retApps;
    },

    searchImage: function() {
      if (!isStrNullOrEmpty(this.filter)) return null;

      switch (Platform()) {
        case PlatformEnum.Windows:
          return Image_search_windows;
        case PlatformEnum.macOS:
          return Image_search_macos;
        default:
          return Image_search_linux;
      }
    }
  }
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
