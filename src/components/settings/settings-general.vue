<template>
  <div>
    <div class="settingsTitle">GENERAL SETTINGS</div>

    <div class="param">
      <input
        type="checkbox"
        id="launchAtLogin"
        v-model="isLaunchAtLogin"
        :disabled="isLaunchAtLogin == null"
      />
      <label class="defColor" for="launchAtLogin">Launch at login</label>
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="showAppInSystemDock"
        v-model="showAppInSystemDock"
        :disabled="minimizeToTray !== true"
      />
      <label class="defColor" for="showAppInSystemDock"
        >Show application icon in system dock</label
      >
    </div>

    <div class="param">
      <input type="checkbox" id="minimizeToTray" v-model="minimizeToTray" />
      <label class="defColor" for="minimizeToTray">Minimize to tray</label>
    </div>
    <div v-if="canShowMinimizeToTrayDescription">
      <div class="description">
        By enabling this parameter, the application will stay in memory after
        closing the window and it will be accessible only via the tray icon.
      </div>
      <div class="description">
        Caution: Not all Linux desktop environments support displaying the
        application icon in the system tray.
      </div>
    </div>

    <div class="settingsBoldFont">
      View:
    </div>
    <div class="flexRow paramBlock">
      <div class="defColor paramName">Color theme:</div>
      <select
        v-model="colorTheme"
        style="margin-left:30px; background: var(--background-color);"
      >
        <option :value="colorThemeEnum.system" :key="colorThemeEnum.system"
          >System default</option
        >
        <option :value="colorThemeEnum.light" :key="colorThemeEnum.light"
          >Light</option
        >
        <option :value="colorThemeEnum.dark" :key="colorThemeEnum.dark"
          >Dark</option
        >
      </select>
    </div>

    <div class="settingsBoldFont">
      Autoconnect:
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="connectOnLaunch"
        v-model="autoConnectOnLaunch"
      />
      <label class="defColor" for="connectOnLaunch">On launch</label>
    </div>
    <div class="param" v-if="isCanAutoconnectOnInsecureWIFI">
      <input
        type="checkbox"
        id="connectVPNOnInsecureNetwork"
        v-model="connectVPNOnInsecureNetwork"
      />
      <label class="defColor" for="connectVPNOnInsecureNetwork"
        >When joining insecure WiFi networks</label
      >
    </div>

    <div class="settingsBoldFont">
      On exit:
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="quitWithoutConfirmation"
        v-model="quitWithoutConfirmation"
      />
      <label class="defColor" for="quitWithoutConfirmation"
        >Quit without confirmation when closing application</label
      >
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="disconnect"
        v-model="disconnectOnQuit"
        :disabled="quitWithoutConfirmation === false"
      />
      <label class="defColor" for="disconnect"
        >Disconnect when closing application</label
      >
    </div>

    <!-- DIAGNOSTIC LOGS-->
    <div class="settingsBoldFont">
      Diagnostics:
    </div>
    <div class="flexRow">
      <div class="param">
        <input type="checkbox" id="logging" v-model="logging" />
        <label class="defColor" for="logging">Allow logging</label>
      </div>
      <div class="flexRowRestSpace"></div>

      <button class="btn" v-on:click="onLogs" v-if="isCanSendDiagLogs">
        Diagnostic logs ...
      </button>
    </div>
    <div id="diagnosticLogs" v-if="diagnosticLogsShown">
      <ComponentDiagnosticLogs
        :onClose="
          evtId => {
            diagnosticLogsShown = false;
          }
        "
      />
    </div>
  </div>
</template>

<script>
import { ColorTheme } from "@/store/types";
import ComponentDiagnosticLogs from "@/components/DiagnosticLogs.vue";
import { Platform, PlatformEnum } from "@/platform/platform";
import sender from "@/ipc/renderer-sender";

// VUE component
export default {
  components: {
    ComponentDiagnosticLogs
  },
  data: function() {
    return {
      diagnosticLogsShown: false,
      isLaunchAtLoginValue: null
    };
  },
  mounted() {
    this.doUpdateIsLaunchAtLogin();
  },
  methods: {
    async onLogs() {
      this.diagnosticLogsShown = true;
    },
    async doUpdateIsLaunchAtLogin() {
      try {
        this.isLaunchAtLoginValue = await sender.AutoLaunchIsEnabled();
      } catch (err) {
        console.error("Error obtaining 'LaunchAtLogin' value: ", err);
        this.isLaunchAtLoginValue = null;
      }
    }
  },
  computed: {
    isCanAutoconnectOnInsecureWIFI() {
      return Platform() != PlatformEnum.Linux;
    },
    canShowMinimizeToTrayDescription() {
      return Platform() === PlatformEnum.Linux;
    },

    isLaunchAtLogin: {
      get() {
        return this.isLaunchAtLoginValue;
      },
      set(value) {
        this.isLaunchAtLoginValue = value;
        let theThis = this;
        (async function() {
          try {
            await sender.AutoLaunchSet(theThis.isLaunchAtLoginValue);
          } catch (err) {
            console.error("Error changing 'LaunchAtLogin' value: ", err);
            theThis.isLaunchAtLoginValue = null;
          }
        })();
      }
    },
    autoConnectOnLaunch: {
      get() {
        return this.$store.state.settings.autoConnectOnLaunch;
      },
      set(value) {
        this.$store.dispatch("settings/autoConnectOnLaunch", value);
      }
    },
    connectVPNOnInsecureNetwork: {
      get() {
        return this.$store.state.settings.wifi?.connectVPNOnInsecureNetwork;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        wifi.connectVPNOnInsecureNetwork = value;
        this.$store.dispatch("settings/wifi", wifi);
      }
    },

    minimizeToTray: {
      get() {
        return this.$store.state.settings.minimizeToTray;
      },
      set(value) {
        this.$store.dispatch("settings/minimizeToTray", value);
        if (value !== true)
          this.$store.dispatch("settings/showAppInSystemDock", true);
      }
    },
    showAppInSystemDock: {
      get() {
        return this.$store.state.settings.showAppInSystemDock;
      },
      set(value) {
        this.$store.dispatch("settings/showAppInSystemDock", value);
      }
    },
    disconnectOnQuit: {
      get() {
        return this.$store.state.settings.disconnectOnQuit;
      },
      set(value) {
        this.$store.dispatch("settings/disconnectOnQuit", value);
      }
    },
    quitWithoutConfirmation: {
      get() {
        return this.$store.state.settings.quitWithoutConfirmation;
      },
      set(value) {
        this.$store.dispatch("settings/quitWithoutConfirmation", value);
      }
    },
    logging: {
      get() {
        return this.$store.state.settings.logging;
      },
      set(value) {
        this.$store.dispatch("settings/logging", value);
        sender.SetLogging();
      }
    },

    isCanSendDiagLogs() {
      return sender.IsAbleToSendDiagnosticReport();
    },

    colorThemeEnum() {
      return ColorTheme;
    },
    colorTheme: {
      get() {
        return sender.ColorScheme();
      },
      set(value) {
        sender.ColorSchemeSet(value);
      }
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

input:disabled {
  opacity: 0.5;
}
input:disabled + label {
  opacity: 0.5;
}

label {
  margin-left: 1px;
}

div.description {
  @extend .settingsGrayLongDescriptionFont;
  margin-left: 22px;
  max-width: 425px;
}

button.btn {
  background: transparent;
  border: 0.5px solid #c8c8c8;
  box-sizing: border-box;
  border-radius: 4px;
  cursor: pointer;
}
#diagnosticLogs {
  background: white;
  z-index: 99;
  position: absolute;
  left: 0%;
  top: 0%;
  width: 100%;
  height: 100%;
}

select {
  background: linear-gradient(180deg, #ffffff 0%, #ffffff 100%);
  border: 0.5px solid rgba(0, 0, 0, 0.2);
  border-radius: 3.5px;
  width: 186px;
}
</style>
