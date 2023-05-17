<template>
  <div>
    <div class="settingsTitle">GENERAL SETTINGS</div>

    <div class="param" :title="isLaunchAtLoginDisableBlockerInfo">
      <input
        type="checkbox"
        id="launchAtLogin"
        v-model="isLaunchAtLogin"
        :disabled="
          isLaunchAtLogin == null || isLaunchAtLoginDisableBlockerInfo != ''
        "
        @click="isLaunchAtLoginonClick"
      />
      <label class="defColor" for="launchAtLogin">Launch at login</label>
    </div>

    <div class="param" v-show="!isLinux">
      <input
        type="checkbox"
        id="showAppInSystemDock"
        v-model="showAppInSystemDock"
      />
      <label class="defColor" for="showAppInSystemDock"
        >Show application icon in system dock</label
      >
    </div>

    <div class="param">
      <input type="checkbox" id="minimizeToTray" v-model="minimizeToTray" />
      <label class="defColor" for="minimizeToTray">Minimize to tray</label>
      <div v-if="isLinux">
        <button
          class="noBordersBtn flexRow"
          title="Help"
          v-on:click="this.$refs.helpMinimizeToTrayLinux.showModal()"
        >
          <img src="@/assets/question.svg" />
        </button>
        <ComponentDialog ref="helpMinimizeToTrayLinux" header="Info">
          <div>
            <p>
              By enabling this parameter, the application will stay in memory
              after closing the window and it will be accessible only via the
              tray icon.
            </p>
            <p>
              <b>Caution:</b> Not all Linux desktop environments support
              displaying the application icon in the system tray.
            </p>
          </div>
        </ComponentDialog>
      </div>
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="connectSelectedMapLocation"
        v-model="connectSelectedMapLocation"
      />
      <label class="defColor" for="connectSelectedMapLocation"
        >Connect to location when selecting it on map screen</label
      >
    </div>

    <div class="param">
      <input type="checkbox" id="showHosts" v-model="showHosts" />
      <label class="defColor" for="showHosts"
        >Enable selection of individual servers in server selection list</label
      >
    </div>

    <div class="settingsBoldFont">View:</div>
    <div class="flexRow param">
      <div class="defColor paramName" style="min-width: 102px">
        Color theme:
      </div>
      <select v-model="colorTheme">
        <option :value="colorThemeEnum.system" :key="colorThemeEnum.system">
          System default
        </option>
        <option :value="colorThemeEnum.light" :key="colorThemeEnum.light">
          Light
        </option>
        <option :value="colorThemeEnum.dark" :key="colorThemeEnum.dark">
          Dark
        </option>
      </select>
    </div>

    <div class="flexRow param" v-if="isCanShowTrayIconTheme">
      <div class="defColor paramName" style="min-width: 102px">Tray icon:</div>
      <select v-model="colorThemeTrayIcon">
        <option
          v-if="isCanShowTrayIconThemeAuto"
          :value="colorThemeTrayIconEnum.auto"
          :key="colorThemeTrayIconEnum.auto"
        >
          Auto
        </option>
        <option
          :value="colorThemeTrayIconEnum.light"
          :key="colorThemeTrayIconEnum.light"
        >
          Light
        </option>
        <option
          :value="colorThemeTrayIconEnum.dark"
          :key="colorThemeTrayIconEnum.dark"
        >
          Dark
        </option>
      </select>
    </div>

    <!--
    <div v-if="isLinux && colorScheme === colorThemeEnum.system">
      <div class="description" style="margin-left: 0px;">
        When changing the system color theme, the new application color theme
        will be updated after reopening the application window.
      </div>
    </div> -->

    <div class="settingsBoldFont">Autoconnect:</div>
    <div class="param">
      <input
        type="checkbox"
        id="connectOnLaunch"
        @click="isAutoconnectOnLaunchOnClick"
        v-model="isAutoconnectOnLaunch"
      />
      <label class="defColor" for="connectOnLaunch">On launch</label>
    </div>

    <div
      class="param"
      :title="
        isParanoidMode
          ? 'The option is not applicable when `Enhanced App Authentication` enabled'
          : ''
      "
    >
      <input
        :disabled="isAutoconnectOnLaunch === false || isParanoidMode === true"
        type="checkbox"
        id="connectOnLaunchDaemon"
        @click="isAutoconnectOnLaunchDaemonOnClick"
        v-model="isAutoconnectOnLaunchDaemon"
      />
      <label class="defColor" for="connectOnLaunchDaemon"
        >Allow background daemon to manage autoconnect</label
      >

      <button
        class="noBordersBtn flexRow"
        title="Help"
        v-on:click="this.$refs.helpAutoconnectOnLaunchDaemon.showModal()"
      >
        <img src="@/assets/question.svg" />
      </button>
      <ComponentDialog ref="helpAutoconnectOnLaunchDaemon" header="Info">
        <div>
          <p>
            By enabling this feature the IVPN daemon will manage the
            auto-connection function. This enables the VPN tunnel to startup as
            quickly as possible as the daemon is started early in the operating
            system boot process and before the IVPN app (The GUI).
          </p>
        </div>
      </ComponentDialog>
    </div>

    <div class="settingsBoldFont">On exit:</div>
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

    <!-- TAB-view header (diagnostic) -->
    <div class="flexRow" style="margin-top: 15px">
      <button
        v-on:click="onDiagnosticViewGeneral"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: diagnosticViewIsGeneral }"
      >
        Diagnostics
      </button>
      <button
        v-if="!isLinux"
        v-on:click="onDiagnosticViewBeta"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: diagnosticViewIsBeta }"
      >
        Beta
      </button>

      <button
        style="cursor: auto; flex-grow: 1"
        class="selectableButtonSeparator"
      ></button>
    </div>

    <div v-if="!isLinux" style="height: 6px" />
    <div style="margin-top: 2px">
      <!-- TAB-view (diagnostic): general -->
      <div v-if="diagnosticViewIsGeneral" class="flexRow">
        <div class="param">
          <input type="checkbox" id="logging" v-model="logging" />
          <label class="defColor" for="logging">Allow logging</label>
        </div>
        <div class="flexRowRestSpace"></div>

        <button
          class="settingsButton"
          v-on:click="onLogs"
          v-if="isCanSendDiagLogs"
        >
          Diagnostic logs ...
        </button>
      </div>
      <!-- TAB-view (diagnostic): beta -->
      <div v-if="diagnosticViewIsBeta" class="flexRow">
        <div>
          <div class="param">
            <input type="checkbox" id="beta" v-model="beta" />
            <label class="defColor" for="beta"
              >Notify beta version updates</label
            >
          </div>
          <div class="description">
            Beta versions can break and change often
          </div>
        </div>

        <div class="flexRowRestSpace"></div>
      </div>
    </div>

    <!-- TOPMOST: Diagnostic logs 'dialog' -->
    <div id="diagnosticLogs" v-if="diagnosticLogsShown">
      <ComponentDiagnosticLogs
        :onClose="
          (evtId) => {
            diagnosticLogsShown = false;
          }
        "
      />
    </div>
  </div>
</template>

<script>
import { ColorTheme, ColorThemeTrayIcon } from "@/store/types";
import ComponentDiagnosticLogs from "@/components/DiagnosticLogs.vue";
import { Platform, PlatformEnum } from "@/platform/platform";
import ComponentDialog from "@/components/component-dialog.vue";
const sender = window.ipcSender;

// VUE component
export default {
  components: {
    ComponentDiagnosticLogs,
    ComponentDialog,
  },
  data: function () {
    return {
      diagnosticLogsShown: false,
      isLaunchAtLoginValue: null,
      colorScheme: null,
      diagnosticView: "general",
    };
  },
  mounted() {
    this.colorScheme = sender.ColorScheme();
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
    },

    onDiagnosticViewGeneral() {
      this.diagnosticView = "general";
    },
    onDiagnosticViewBeta() {
      this.diagnosticView = "beta";
    },

    async isAutoconnectOnLaunchOnClick(evt) {
      if (
        (this.isAutoconnectOnLaunch === false) & // going to enable
        (this.$store.state.paranoidModeStatus.IsEnabled === true) // EAA enabled
      ) {
        let ret = await sender.showMessageBoxSync(
          {
            type: "warning",
            message: `Enhanced App Authentication enabled`,
            detail:
              "Warning: On application start 'Autoconnect on application launch' will not be applied until the EAA password is entered.",
            buttons: ["Enable", "Cancel"],
          },
          true
        );
        if (ret == 1) {
          // cancel
          evt.returnValue = false;
        }
      }
    },
    async isAutoconnectOnLaunchDaemonOnClick(evt) {
      if (this.isAutoconnectOnLaunchDaemon === true) return; // we are going to disable this option. No messages required
      if (this.isLaunchAtLogin !== true) {
        let ret = await sender.showMessageBoxSync(
          {
            type: "warning",
            message: `"Launch at login" disabled`,
            detail:
              'This option requires "Launch at login" to be enabled.\nDo you want to enable both options?',
            buttons: ["Enable", "Cancel"],
          },
          true
        );
        if (ret == 1) {
          // Cancel
          evt.returnValue = false;
        } else {
          this.isLaunchAtLogin = true;
        }
      }
    },
  },
  watch: {
    isAutoconnectOnLaunchDaemon() {
      this.doUpdateIsLaunchAtLogin();
    },
    isWifiActionsInBackground() {
      this.doUpdateIsLaunchAtLogin();
    },
  },
  computed: {
    isParanoidMode() {
      return this.$store.state.paranoidModeStatus.IsEnabled === true;
    },
    isLinux() {
      return Platform() === PlatformEnum.Linux;
    },

    isLaunchAtLogin: {
      get() {
        return this.isLaunchAtLoginValue;
      },
      set(value) {
        this.isLaunchAtLoginValue = value;
        let theThis = this;
        (async function () {
          try {
            await sender.AutoLaunchSet(theThis.isLaunchAtLoginValue);
          } catch (err) {
            console.error("Error changing 'LaunchAtLogin' value: ", err);
            theThis.isLaunchAtLoginValue = null;
          }
        })();
      },
    },

    isLaunchAtLoginDisableBlockerInfo() {
      if (!this.isLaunchAtLogin) return "";
      if (this.isWifiActionsInBackground === true)
        return `This option can not be disabled\nbecause of 'Allow background daemon to Apply WiFi Control settings' is active`;
      if (this.isAutoconnectOnLaunchDaemon === true)
        return `This option can not be disabled\nbecause of 'Allow background daemon to manage autoconnect' is active`;
      return "";
    },

    isAutoconnectOnLaunch: {
      get() {
        return this.$store.state.settings.daemonSettings.IsAutoconnectOnLaunch;
      },
      set(value) {
        sender.SetAutoconnectOnLaunch(value, null);
        if (value === false) this.isAutoconnectOnLaunchDaemon = false;
      },
    },
    isAutoconnectOnLaunchDaemon: {
      get() {
        return this.$store.state.settings.daemonSettings
          .IsAutoconnectOnLaunchDaemon;
      },
      set(value) {
        sender.SetAutoconnectOnLaunch(null, value);
      },
    },

    isWifiActionsInBackground: {
      get() {
        return this.$store.state.settings.daemonSettings?.WiFi
          ?.canApplyInBackground;
      },
    },

    minimizeToTray: {
      get() {
        return this.$store.state.settings.minimizeToTray;
      },
      set(value) {
        this.$store.dispatch("settings/minimizeToTray", value);
        if (value !== true)
          this.$store.dispatch("settings/showAppInSystemDock", true);
      },
    },
    connectSelectedMapLocation: {
      get() {
        return this.$store.state.settings.connectSelectedMapLocation;
      },
      set(value) {
        this.$store.dispatch("settings/connectSelectedMapLocation", value);
      },
    },
    showHosts: {
      get() {
        return this.$store.state.settings.showHosts;
      },
      set(value) {
        this.$store.dispatch("settings/showHosts", value);
      },
    },
    showAppInSystemDock: {
      get() {
        return this.$store.state.settings.showAppInSystemDock;
      },
      set(value) {
        this.$store.dispatch("settings/showAppInSystemDock", value);
      },
    },
    disconnectOnQuit: {
      get() {
        return this.$store.state.settings.disconnectOnQuit;
      },
      set(value) {
        this.$store.dispatch("settings/disconnectOnQuit", value);
      },
    },
    quitWithoutConfirmation: {
      get() {
        return this.$store.state.settings.quitWithoutConfirmation;
      },
      set(value) {
        this.$store.dispatch("settings/quitWithoutConfirmation", value);
      },
    },
    logging: {
      get() {
        return this.$store.state.settings.daemonSettings?.IsLogging;
      },
      async set(value) {
        await sender.SetLogging(value);
      },
    },

    beta: {
      get() {
        return this.$store.state.settings.updates.isBetaProgram;
      },
      set(value) {
        let settingsUpdates = Object.assign(
          {},
          this.$store.state.settings.updates
        );
        settingsUpdates.isBetaProgram = value;

        this.$store.dispatch("settings/updates", settingsUpdates);
      },
    },

    isCanSendDiagLogs() {
      return sender.IsAbleToSendDiagnosticReport();
    },

    colorThemeEnum() {
      return ColorTheme;
    },
    colorTheme: {
      get() {
        return this.colorScheme;
      },
      set(value) {
        sender.ColorSchemeSet(value);
        this.colorScheme = value;
      },
    },
    isCanShowTrayIconTheme() {
      return (
        Platform() === PlatformEnum.Windows || Platform() === PlatformEnum.Linux
      );
    },
    isCanShowTrayIconThemeAuto() {
      return Platform() === PlatformEnum.Windows;
    },
    colorThemeTrayIconEnum() {
      return ColorThemeTrayIcon;
    },
    colorThemeTrayIcon: {
      get() {
        let ret = ColorThemeTrayIcon.auto;
        if (this.$store.state.settings.colorThemeTrayIcon != undefined)
          ret = this.$store.state.settings.colorThemeTrayIcon;
        if (!this.isCanShowTrayIconThemeAuto && ret === ColorThemeTrayIcon.auto)
          ret = ColorThemeTrayIcon.light;
        return ret;
      },
      set(value) {
        this.$store.dispatch("settings/colorThemeTrayIcon", value);
      },
    },
    diagnosticViewIsGeneral() {
      return !this.diagnosticViewIsBeta;
    },
    diagnosticViewIsBeta() {
      return this.diagnosticView === "beta";
    },
  },
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
  margin-left: 21px;
  max-width: 490px;
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
  border: 0.5px solid rgba(0, 0, 0, 0.2);
  border-radius: 3.5px;
  width: 186px;
}
</style>
