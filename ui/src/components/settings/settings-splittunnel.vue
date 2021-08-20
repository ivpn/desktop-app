<template>
  <div>
    <div class="settingsTitle">SPLIT TUNNEL SETTINGS</div>

    <div class="param">
      <input type="checkbox" id="isSTEnabled" v-model="isSTEnabled" />
      <label class="defColor" for="isSTEnabled">Split Tunnel</label>
    </div>
    <div class="fwDescription">
      By enabling this feature you can exclude traffic of specific applications
      from the VPN tunnel
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
            Another application ...
          </button>
        </div>
      </div>

      <div class="horizontalLine" />

      <div
        class="scrollableColumnContainer"
        style="padding:1px;margin-top: 1px; margin-bottom:1px; max-height: 295px;  height: 320px;"
      >
        <div v-for="app of filteredApps" v-bind:key="app.AppBinaryPath">
          <div
            class="flexRow grayedOnHover"
            style="padding: 4px; padding-bottom: 8px; height: 32px; min-height: 32px;"
          >
            <div class="flexRowRestSpace" style="height: 100%">
              <div v-if="!app.AppName">
                <div>
                  {{ app.AppBinaryPath }}
                </div>
              </div>
              <div v-else>
                <div>
                  {{ app.AppName }}
                </div>
                <div
                  class="settingsGrayLongDescriptionFont"
                  v-if="app.AppName != app.AppGroup"
                >
                  {{ app.AppGroup }}
                </div>
              </div>
            </div>

            <div>
              <button
                class="noBordersBtn"
                v-if="app.isSplitted"
                v-on:click="removeApp(app.AppBinaryPath)"
                style="pointer-events: auto;"
              >
                <img width="24" height="24" src="@/assets/minus.svg" />
              </button>

              <button
                class="noBordersBtn"
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

export default {
  data: function() {
    return {
      filter: "",
      showAllApps: false,

      // Heshed info about all available applications.
      //  allAppsHashed[binaryPath] = AppInfo
      // Where the AppInfo object:
      //	  AppBinaryPath string
      //    AppName  string
      //    AppGroup string
      //    isSplitted (true or (false/null))
      allAppsHashed: {},
      appsToShow: null
    };
  },
  async mounted() {
    // show base information about splitted apps immediately
    //this.updateAppsToShow();

    let allApps = await sender.GetInstalledApps();
    // create a list of hashed appinfo (by app path)
    allApps.forEach(appInfo => {
      this.allAppsHashed[appInfo.AppBinaryPath] = appInfo;
    });
    // now we are able to update information about splitted apps
    this.updateAppsToShow();
    // If no applications selected: show all applications for selection
    if (this.showAllApps == false) {
      var st = this.$store.state.settings.splitTunnelling;
      if (!st.apps || st.apps.length == 0) {
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
      let configApps = this.$store.state.settings.splitTunnelling.apps;

      // hashed list of splitted apps (needed to avoid duplicates in final list)
      let configAppsHashed = {};

      // prepare information for selected apps: update app info (if exists)
      if (configApps) {
        configApps.forEach(appPath => {
          // use 'Object.assign' to not update data in 'this.allAppsHashed'
          let appInfoConst = this.allAppsHashed[appPath];
          let appInfo = {};

          if (!appInfoConst)
            appInfo = { AppBinaryPath: appPath, AppName: null, AppGroup: null };
          else appInfo = Object.assign({}, appInfoConst);

          appInfo.isSplitted = true;
          configAppsHashed[appPath] = appInfo;
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
        for (const [appPath, appInfo] of Object.entries(this.allAppsHashed)) {
          let splittedApp = configAppsHashed[appPath];
          if (splittedApp) {
            appsInfo.push(splittedApp); // splitted app
          } else {
            appsInfo.push(appInfo); // not-splitted app
          }
        }
      }

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

      var st = this.$store.state.settings.splitTunnelling;
      var stApps = [];
      if (st.apps) stApps = Object.assign(stApps, st.apps);

      ret.filePaths.forEach(appPath => {
        if (stApps.includes(appPath) == false) stApps.push(appPath);
      });

      await sender.SplitTunnelSetConfig(st.enabled, stApps);
    },

    async removeApp(appPath) {
      var st = this.$store.state.settings.splitTunnelling;
      var stApps = [];
      if (st.apps) stApps = Object.assign(stApps, st.apps);

      var index = stApps.indexOf(appPath);
      if (index === -1) return;

      stApps.splice(index, 1);

      await sender.SplitTunnelSetConfig(st.enabled, stApps);
    },

    async addApp(appPath) {
      var st = this.$store.state.settings.splitTunnelling;
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
      this.updateAppsToShow();
    }
  },

  computed: {
    isSTEnabled: {
      get() {
        return this.$store.state.settings.splitTunnelling.enabled;
      },
      async set(value) {
        var st = this.$store.state.settings.splitTunnelling;
        await sender.SplitTunnelSetConfig(value, st.apps);
      }
    },

    // needed for 'watch'
    STConfig: function() {
      return this.$store.state.settings.splitTunnelling;
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
</style>
