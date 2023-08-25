<template>
  <div class="flexColumn">
    <div class="settingsTitle">WIFI CONTROL SETTINGS</div>

    <div
      class="param"
      :title="
        isParanoidMode
          ? 'The option is not applicable when `Enhanced App Authentication` enabled'
          : ''
      "
    >
      <input
        :disabled="
          !canApplyInBackground &&
          (isParanoidMode === true ||
            (!connectVPNOnInsecureNetwork && !trustedNetworksControl))
        "
        type="checkbox"
        id="canApplyInBackground"
        @click="canApplyInBackgroundClick"
        v-model="canApplyInBackground"
      />
      <label class="defColor" for="canApplyInBackground"
        >Allow background daemon to Apply WiFi Control settings</label
      >

      <button
        class="noBordersBtn flexRow"
        title="Help"
        v-on:click="$refs.helpCanApplyInBackground.showModal()"
      >
        <img src="@/assets/question.svg" />
      </button>

      <ComponentDialog ref="helpCanApplyInBackground" header="Info">
        <div>
          <p>
            By enabling this feature the IVPN daemon will apply the WiFi control
            settings before the IVPN app has been launched. This enables the
            WiFi control settings to be applied as quickly as possible as the
            daemon is started early in the operating system boot process and
            before the IVPN app (The GUI).
          </p>
        </div>
      </ComponentDialog>
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="connectVPNOnInsecureNetwork"
        @click="connectVPNOnInsecureNetworkOnClick"
        v-model="connectVPNOnInsecureNetwork"
      />
      <label class="defColor" for="connectVPNOnInsecureNetwork"
        >Autoconnect on joining WiFi networks without encryption</label
      >
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="trustedNetworksControl"
        @click="trustedNetworksControlOnClick"
        v-model="trustedNetworksControl"
      />
      <label class="defColor" for="trustedNetworksControl"
        >Trusted/Untrusted WiFi network control</label
      >
    </div>
    <div class="fwDescription">
      By enabling this feature you can define a WiFi network as trusted or
      untrusted and what actions to take when joining the WiFi network
    </div>

    <div class="flexRow">
      <button
        v-on:click="onNetworks"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: !isActionsView }"
      >
        WiFi networks
      </button>
      <button
        v-on:click="onActions"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: isActionsView }"
      >
        Actions
      </button>
      <button
        style="cursor: auto; flex-grow: 1"
        class="selectableButtonSeparator"
      ></button>
    </div>
    <div class="flexColumn" style="min-height: 0px">
      <!-- ACTIONS -->
      <div v-if="isActionsView" style="flex-grow: 1">
        <div class="settingsBoldFont">Actions for Untrusted WiFi</div>
        <div class="param">
          <input
            type="checkbox"
            id="unTrustedConnectVpn"
            v-model="unTrustedConnectVpn"
          />
          <label class="defColor" for="unTrustedConnectVpn"
            >Connect to VPN</label
          >
        </div>
        <div class="param">
          <input
            type="checkbox"
            id="unTrustedEnableFirewall"
            v-model="unTrustedEnableFirewall"
          />
          <label class="defColor" for="unTrustedEnableFirewall"
            >Enable firewall</label
          >
        </div>

        <div class="settingsBoldFont">Actions for Trusted WiFi</div>
        <div class="param">
          <input
            type="checkbox"
            id="trustedDisconnectVpn"
            v-model="trustedDisconnectVpn"
          />
          <label class="defColor" for="trustedDisconnectVpn"
            >Disconnect from VPN</label
          >
        </div>
        <div class="param">
          <input
            type="checkbox"
            id="trustedDisableFirewall"
            v-model="trustedDisableFirewall"
          />
          <label class="defColor" for="trustedDisableFirewall"
            >Disable firewall</label
          >
        </div>
      </div>

      <!-- NETWORKS -->
      <div v-if="!isActionsView" class="flexColumn">
        <div class="flexRow" style="margin-top: 12px; margin-bottom: 12px">
          <div class="flexRowRestSpace">
            Default trust status for undefined networks:
          </div>
          <div>
            <select
              v-model="defaultTrustStatusIsTrusted"
              class="trustedConfigBase"
              style="background: var(--background-color)"
              v-bind:class="{
                trustedConfigUntrusted: defaultTrustStatusIsTrusted == false,
                trustedConfigTrusted: defaultTrustStatusIsTrusted == true,
              }"
            >
              <option :value="false">Untrusted</option>
              <option :value="true">Trusted</option>
              <option :value="null">No status</option>
            </select>
          </div>
        </div>

        <div class="horizontalLine" />

        <div
          class="scrollableColumnContainer"
          style="
            padding: 1px;
            margin-top: 8px;
            margin-bottom: 8px;
            max-height: 235px;
            height: 225px;
          "
        >
          <div v-for="wifi of networks" v-bind:key="wifi.SSID">
            <trustedNetConfigControl
              :wifiInfo="wifi"
              :onChange="onNetworkTrustChanged"
            />
          </div>
        </div>
      </div>

      <!-- FOOTER -->
      <div style="position: sticky; bottom: 20px">
        <div class="horizontalLine" />

        <div class="flexRow" style="margin-top: 15px">
          <div class="param" v-if="isActionsView == false">
            <input
              type="checkbox"
              id="showAllWifi"
              v-on:click="onShowAllNetworks"
              style="margin: 0px 5px 0px 0px"
            />
            <label class="defColor" for="showAllWifi">
              Show all WiFi networks</label
            >
          </div>

          <div class="flexRowRestSpace" />

          <button class="settingsButton" v-on:click="onResetToDefaultSettings">
            Reset to default settings
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import trustedNetConfigControl from "@/components/controls/control-trusted-network-config.vue";
import ComponentDialog from "@/components/component-dialog.vue";

const sender = window.ipcSender;

export default {
  components: {
    trustedNetConfigControl,
    ComponentDialog,
  },
  mounted() {
    //if (this.trustedNetworksControl === true) sender.GetWiFiAvailableNetworks();
    this.doUpdateIsLaunchAtLogin();
  },
  data: function () {
    return {
      isActionsView: false,
      showAllNetworks: false,
      isLaunchAtLoginValue: null,
    };
  },
  methods: {
    async doUpdateIsLaunchAtLogin() {
      try {
        this.isLaunchAtLoginValue = await sender.AutoLaunchIsEnabled();
      } catch (err) {
        console.error("Error obtaining 'LaunchAtLogin' value: ", err);
        this.isLaunchAtLoginValue = null;
      }
    },

    onShowAllNetworks() {
      this.showAllNetworks = !this.showAllNetworks;
      if (
        (this.showAllNetworks == true && !this.networks) ||
        this.availableWiFiNetworks.length == 0
      )
        sender.GetWiFiAvailableNetworks();
    },
    onActions() {
      this.isActionsView = true;
    },
    onNetworks() {
      this.isActionsView = false;
    },
    onNetworkTrustChanged(ssid, isTrusted) {
      let wifi = Object.assign({}, this.wifiSettings);
      var nets = [];

      if (this.wifiSettings?.networks != null)
        nets = [...this.wifiSettings.networks];

      if (isTrusted == null) {
        nets = nets.filter((wifi) => wifi.ssid != ssid);
      } else {
        let alreadyExists = nets.filter((wifi) => wifi.ssid == ssid);
        if (alreadyExists != null && alreadyExists.length > 0) {
          // replace item with a new value
          nets = [
            ...nets.map((item) =>
              item.ssid !== ssid ? item : { ssid: ssid, isTrusted: isTrusted }
            ),
          ];
        } else nets.push({ ssid: ssid, isTrusted: isTrusted });
      }
      wifi.networks = nets;

      sender.SetWiFiSettings(wifi);
    },

    onResetToDefaultSettings() {
      let actionNo = sender.showMessageBoxSync({
        type: "question",
        buttons: ["Yes", "Cancel"],
        message: "Reset all settings to default values",
        detail: `Are you sure you want to reset the trust status for all networks and actions to default settings?`,
      });
      if (actionNo == 1) return;

      let wifi = Object.assign({}, this.wifiSettings);
      wifi.actions = {
        unTrustedConnectVpn: true,
        unTrustedEnableFirewall: true,

        trustedDisconnectVpn: true,
        trustedDisableFirewall: true,
      };
      wifi.networks = null;
      wifi.defaultTrustStatusTrusted = null;

      sender.SetWiFiSettings(wifi);
    },

    async trustedNetworksControlOnClick(evt) {
      if (
        (this.trustedNetworksControl === false) & // going to enable
        (this.$store.state.paranoidModeStatus.IsEnabled === true) // EAA enabled
      ) {
        let ret = await sender.showMessageBoxSync(
          {
            type: "warning",
            message: `Enhanced App Authentication`,
            detail:
              "Warning: On application start Trusted WiFi will be disabled until the EAA password is entered",
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

    async connectVPNOnInsecureNetworkOnClick(evt) {
      if (
        (this.connectVPNOnInsecureNetwork === false) & // going to enable
        (this.$store.state.paranoidModeStatus.IsEnabled === true) // EAA enabled
      ) {
        let ret = await sender.showMessageBoxSync(
          {
            type: "warning",
            message: `Enhanced App Authentication`,
            detail:
              "Warning: On application start `Autoconnect on joining networks without encryption` will be disabled until the EAA password is entered",
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

    async canApplyInBackgroundClick(evt) {
      if (this.canApplyInBackground === true) return; // we are going to disable this option. No messages required

      if (this.isLaunchAtLoginValue !== true) {
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
          setTimeout(async () => {
            try {
              await sender.AutoLaunchSet(true);
              this.doUpdateIsLaunchAtLogin();
            } catch (err) {
              console.error("Error enabling 'LaunchAtLogin': ", err);
            }
          }, 0);
        }
      }
    },

    resetBackgroundOptionIfReqiuired() {
      if (!this.canApplyInBackground) return;
      if (!this.connectVPNOnInsecureNetwork && !this.trustedNetworksControl) {
        this.canApplyInBackground = false;
      }
    },
  },
  watch: {
    connectVPNOnInsecureNetwork() {
      this.resetBackgroundOptionIfReqiuired();
    },
    trustedNetworksControl() {
      this.resetBackgroundOptionIfReqiuired();
    },
  },
  computed: {
    isParanoidMode() {
      return this.$store.state.paranoidModeStatus.IsEnabled === true;
    },

    canApplyInBackground: {
      get() {
        return this.wifiSettings?.canApplyInBackground;
      },
      set(value) {
        let wifi = Object.assign({}, this.wifiSettings);
        wifi.canApplyInBackground = value;

        sender.SetWiFiSettings(wifi);
      },
    },

    connectVPNOnInsecureNetwork: {
      get() {
        return this.wifiSettings?.connectVPNOnInsecureNetwork;
      },
      set(value) {
        let wifi = Object.assign({}, this.wifiSettings);
        wifi.connectVPNOnInsecureNetwork = value;

        sender.SetWiFiSettings(wifi);
      },
    },

    wifiSettings: function () {
      if (!this.$store.state.settings.daemonSettings?.WiFi) return null;
      return JSON.parse(
        JSON.stringify(this.$store.state.settings.daemonSettings?.WiFi)
      );
    },

    availableWiFiNetworks: function () {
      var nets = [];
      try {
        let allNets = this.$store.state.vpnState.availableWiFiNetworks;
        if (allNets != null) nets = allNets.filter((w) => w.SSID);
      } catch (e) {
        console.error(e);
      }
      return nets;
    },
    networks: function () {
      var nets = [];
      try {
        if (this.wifiSettings?.networks != null)
          nets = [...this.wifiSettings.networks];

        let currWiFi = this.$store.state.vpnState.currentWiFiInfo;
        if (currWiFi != null && currWiFi.SSID != "") {
          let alreadyExists = nets.filter((wifi) => wifi.ssid == currWiFi.SSID);

          // check is current wifi already exists
          if (alreadyExists == null || alreadyExists.length == 0)
            nets.unshift({ ssid: currWiFi.SSID, isTrusted: null });

          if (this.showAllNetworks) {
            // add rest of available networks
            let restNetworks = this.availableWiFiNetworks;
            if (restNetworks != null) {
              for (let w of restNetworks) {
                if (
                  w.SSID != "" &&
                  nets.findIndex((t) => t.ssid === w.SSID) == -1
                )
                  nets.push({ ssid: w.SSID, isTrusted: null });
              }
            }
          }
        }
      } catch (e) {
        console.error(e);
      }
      return nets;
    },
    defaultTrustStatusIsTrusted: {
      get() {
        return this.wifiSettings?.defaultTrustStatusTrusted;
      },
      set(value) {
        let wifi = Object.assign({}, this.wifiSettings);
        wifi.defaultTrustStatusTrusted = value;

        sender.SetWiFiSettings(wifi);
      },
    },
    trustedNetworksControl: {
      get() {
        return this.wifiSettings?.trustedNetworksControl;
      },
      async set(value) {
        // INFO: see also method "trustedNetworksControlOnClick()"
        let wifi = Object.assign({}, this.wifiSettings);
        wifi.trustedNetworksControl = value;

        sender.SetWiFiSettings(wifi);
      },
    },
    unTrustedConnectVpn: {
      get() {
        return this.wifiSettings?.actions?.unTrustedConnectVpn;
      },
      set(value) {
        let wifi = JSON.parse(JSON.stringify(this.wifiSettings));
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.unTrustedConnectVpn = value;

        sender.SetWiFiSettings(wifi);
      },
    },
    unTrustedEnableFirewall: {
      get() {
        return this.wifiSettings?.actions?.unTrustedEnableFirewall;
      },
      set(value) {
        let wifi = JSON.parse(JSON.stringify(this.wifiSettings));
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.unTrustedEnableFirewall = value;

        sender.SetWiFiSettings(wifi);
      },
    },
    trustedDisconnectVpn: {
      get() {
        return this.wifiSettings?.actions?.trustedDisconnectVpn;
      },
      set(value) {
        let wifi = JSON.parse(JSON.stringify(this.wifiSettings));
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.trustedDisconnectVpn = value;

        sender.SetWiFiSettings(wifi);
      },
    },
    trustedDisableFirewall: {
      get() {
        return this.wifiSettings?.actions?.trustedDisableFirewall;
      },
      set(value) {
        let wifi = JSON.parse(JSON.stringify(this.wifiSettings));
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.trustedDisableFirewall = value;

        sender.SetWiFiSettings(wifi);
      },
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/platform/base";

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

select.trustedConfigBase {
  min-width: 90px;
  border-width: 0px;
  background: inherit;
}

select.trustedConfigUntrusted {
  @extend .trustedConfigBase;
  color: red;
}
select.trustedConfigTrusted {
  @extend .trustedConfigBase;
  color: #3b99fc;
}

input:disabled {
  opacity: 0.5;
}
input:disabled + label {
  opacity: 0.5;
}
</style>
