<template>
  <div>
    <div class="settingsTitle">SPLIT TUNNEL SETTINGS</div>

    <div class="param">
      <input type="checkbox" id="isSTEnabled" v-model="isSTEnabled" />
      <label class="defColor" for="isSTEnabled">Split Tunnel</label>
    </div>
    <div class="fwDescription">
      By enabling this feature you can exclude traffic for a specific
      applications from the VPN tunnel
    </div>

    <!-- APPS -->
    <div class="flexColumn">
      <div class="flexRow" style="margin-top: 12px; margin-bottom:12px">
        <div
          class="flexRowRestSpace settingsBoldFont"
          style="margin-top: 0px; margin-bottom:0px"
        >
          Applications
        </div>

        <input
          id="filter"
          class="styled"
          placeholder="Search for app"
          v-model="filter"
          v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
        />

        <div>
          <button
            class="settingsButton"
            style="min-width: 156px"
            v-on:click="addNewApplication"
          >
            Add application ...
          </button>
        </div>
      </div>

      <div class="horizontalLine" />

      <div
        style="overflow: auto; padding:1px;margin-top: 1px; margin-bottom:1px; max-height: 295px;  height: 320px; position: relative; "
      >
        <spinner
          :loading="isLoadingAllApps"
          style="position: absolute; background: transparent; width: 480px;"
        />

        <div v-for="app of filteredApps" v-bind:key="app.AppBinaryPath">
          <div
            class="flexRow grayedOnHover"
            style="padding: 4px; padding-top: 7px; padding-bottom: 7px; height: 32px; min-height: 32px;"
          >
            <binaryIconControl
              :binaryPath="app.AppBinaryPath"
              :preloadedBase64Icon="app.AppIcon"
              style="min-width:32px; min-height:32px; max-width:32px; max-height:32px; padding: 4px;"
            />
            <!-- {{ app.AppBinaryPath }} -->
            <div
              class="flexRowRestSpace"
              style="max-width: 375px; padding-left: 5px"
            >
              <!-- Manually added application -->
              <div v-if="!app.AppName">
                <div class="text">
                  {{ getFileName(app.AppBinaryPath) }}
                </div>
                <div class="settingsGrayLongDescriptionFont text">
                  {{ getFileFolder(app.AppBinaryPath) }}
                </div>
              </div>
              <div v-else>
                <!-- Application from the installed apps list (AppName and AppGroup is known)-->
                <div class="text">
                  {{ app.AppName }}
                </div>
                <div
                  class="settingsGrayLongDescriptionFont text"
                  v-if="app.AppName != app.AppGroup"
                >
                  {{ app.AppGroup }}
                </div>
              </div>
            </div>

            <div>
              <button
                class="noBordersBtn opacityOnHover"
                v-if="app.isSplitted"
                v-on:click="removeApp(app.AppBinaryPath)"
                style="pointer-events: auto;"
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
    </div>

    <!-- FOOTER -->
    <div style="position: sticky; bottom: 20px;">
      <div class="horizontalLine" />

      <div class="flexRow" style="margin-top: 15px;">
        <div class="param">
          <input
            type="checkbox"
            id="showAllApplications"
            v-model="showAllApps"
            v-on:click="onShowAllApps"
            style="margin:0px 5px 0px 0px"
          />
          <label class="defColor" for="showAllApplications">
            Show all applications</label
          >
        </div>

        <div class="flexRowRestSpace" />

        <button class="settingsButton" v-on:click="onResetToDefaultSettings">
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

import binaryIconControl from "@/components/controls/control-app-binary-icon.vue";
import spinner from "@/components/controls/control-spinner.vue";

export default {
  components: {
    spinner,
    binaryIconControl
  },

  data: function() {
    return {
      isLoadingAllApps: false,
      filter: "",
      showAllApps: false,
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
      appsToShow: null // []; array of appInfo
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

    // If no applications selected: show all applications for selection
    if (this.showAllApps == false) {
      var st = this.$store.state.vpnState.splitTunnelling;
      if (allApps && (!st.apps || st.apps.length == 0)) {
        this.onShowAllApps();
      }
    }
  },

  watch: {
    STConfig() {
      this.updateAppsToShow();
    }
  },

  methods: {
    updateAppsToShow() {
      // 'splitted' applications
      let configApps = this.$store.state.vpnState.splitTunnelling.apps;

      // hashed list of splitted apps (needed to avoid duplicates in final list)
      let configAppsHashed = {};

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
          configAppsHashed[appPath.toLowerCase()] = appInfo;
        });
      }

      // apps to show
      let appsInfo = [];

      if (this.showAllApps == false) {
        // show only splitted apps
        for (const [, appInfo] of Object.entries(configAppsHashed)) {
          appsInfo.push(appInfo);
        }
      } else {
        // show all appInfo (avoid duplicates)
        let allApps = Object.assign(this.allAppsHashed, configAppsHashed);

        for (const [binPath, appInfo] of Object.entries(allApps)) {
          // ensure the apps not from config are 'unchecked'
          if (!configAppsHashed[binPath] && this.allAppsHashed[binPath])
            appInfo.isSplitted = false;

          appsInfo.push(appInfo);
        }
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

    async addNewApplication() {
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
      var st = this.$store.state.vpnState.splitTunnelling;
      var stApps = [];
      if (st.apps) stApps = Object.assign(stApps, st.apps);

      stApps.push(appPath);

      await sender.SplitTunnelSetConfig(st.enabled, stApps);
    },

    async onResetToDefaultSettings() {
      let actionNo = sender.showMessageBoxSync({
        type: "question",
        buttons: ["Yes", "Cancel"],
        message: "Reset all settings to default values",
        detail: `Are you sure you want to reset the Split Tunnel configuration for all applications?`
      });
      if (actionNo == 1) return;

      this.filter = "";
      await sender.SplitTunnelSetConfig(false, null);
    },

    async onShowAllApps() {
      this.showAllApps = !this.showAllApps;
      this.filter = "";
      setTimeout(() => {
        this.updateAppsToShow();
      }, 0);
    },

    getFileFolder(filePath) {
      let fname = this.getFileName(filePath);
      if (!fname) return filePath;
      return filePath.substring(0, filePath.length - fname.length);
    },

    getFileName(filePath) {
      if (!filePath) return null;
      return filePath
        .split("\\")
        .pop()
        .split("/")
        .pop();
    }
  },

  computed: {
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

    filteredApps: function() {
      if (this.filter == null || this.filter.length == 0)
        return this.appsToShow;

      let filter = this.filter.toLowerCase();
      let filterFunc = function(appInfo) {
        if (appInfo.AppName == null || appInfo.AppName == "") {
          return appInfo.AppBinaryPath.toLowerCase().includes(filter);
        }

        return (
          appInfo.AppName.toLowerCase().includes(filter) ||
          appInfo.AppGroup.toLowerCase().includes(filter)
        );
      };

      return this.appsToShow.filter(appInfo => filterFunc(appInfo));
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

.text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
