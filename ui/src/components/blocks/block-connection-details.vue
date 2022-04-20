<template>
  <div>
    <div>
      <div class="horizontalLine" />
      <div id="connection_header">
        <div style="height: 24px"></div>

        <span class="block datails_text"> CONNECTION DETAILS </span>
      </div>
      <div class="horizontalLine" />
    </div>

    <!-- FIREWALL -->

    <OnOffButtonControl
      v-bind:class="{ lowOpacity: IsPaused }"
      text="Firewall"
      description="Ensure that all traffic is routed through VPN"
      :onChecked="firewallOnChecked"
      :isChecked="this.$store.state.vpnState.firewallState.IsEnabled"
      :checkedColor="
        this.$store.state.vpnState.firewallState.IsPersistent
          ? '#77152a77'
          : null
      "
      :isProgress="firewallIsProgress"
    />

    <!-- ANTITRACKER -->
    <div class="horizontalLine" />

    <OnOffButtonControl
      text="AntiTracker"
      description="Block trackers whilst connected to VPN"
      :onChecked="antitrackerOnChecked"
      :isChecked="this.$store.state.settings.isAntitracker"
      :switcherOpacity="!IsConnected ? 0.4 : 1"
      :checkedColor="
        this.$store.state.settings.isAntitrackerHardcore ? '#77152a' : null
      "
      :isProgress="antitrackerIsProgress"
    />

    <!-- PROTOCOL -->
    <div class="horizontalLine" />

    <SelectButtonControl
      class="leftPanelBlock"
      :click="onShowPorts"
      v-bind:text="portProtocolText"
      description="Protocol/Port"
    />

    <!-- WIFI -->
    <transition
      name="fade"
      v-if="isTrustedNetworksControlActive || isConnectVPNOnInsecureNetwork"
    >
      <div v-if="wifiSSID">
        <div class="horizontalLine" />

        <SelectButtonControl
          class="leftPanelBlock"
          :click="onShowWifiConfig"
          v-bind:text="wifiSSID"
          :description="wifiSSID == '' ? 'No WiFi connection' : 'WiFi network'"
          :markerText="WiFiMarkerText"
          :markerColor="WiFiMarkerColor"
          :markerTextColor="'var(--text-color-details)'"
        />
      </div>
    </transition>

    <!-- GEOLOCATOIN INFO -->
    <transition name="fade">
      <div v-if="$store.state.settings.minimizedUI">
        <div class="horizontalLine" />

        <GeolocationInfoControl class="blockWithMrgings" />
      </div>
    </transition>
  </div>
</template>

<script>
import OnOffButtonControl from "@/components/controls/control-config-on-off-button.vue";
import SelectButtonControl from "@/components/controls/control-config-to-select-button.vue";
import GeolocationInfoControl from "@/components/controls/control-geolocation-info.vue";
const sender = window.ipcSender;
import { enumValueName } from "@/helpers/helpers";
import {
  VpnTypeEnum,
  PortTypeEnum,
  PauseStateEnum,
  VpnStateEnum,
} from "@/store/types";

function processError(e) {
  console.error(e);
  sender.showMessageBox({
    type: "error",
    buttons: ["OK"],
    message: e.toString(),
  });
}

export default {
  components: {
    OnOffButtonControl,
    SelectButtonControl,
    GeolocationInfoControl,
  },
  props: ["onShowPorts", "onShowWifiConfig"],
  data: function () {
    return {
      antitrackerIsProgress: false,
      firewallIsProgress: false,
    };
  },

  computed: {
    portProtocolText: function () {
      let port = this.$store.getters["settings/getPort"];
      let protocol = this.$store.getters["settings/vpnType"];
      const isMH = this.$store.state.settings.isMultiHop;
      if (isMH === true) {
        // do not show port number for multi-hop connections
        return `${enumValueName(VpnTypeEnum, protocol)}/${enumValueName(
          PortTypeEnum,
          port.type
        )}`;
      }
      return `${enumValueName(VpnTypeEnum, protocol)}/${enumValueName(
        PortTypeEnum,
        port.type
      )} ${port.port}`;
    },
    isTrustedNetworksControlActive() {
      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null) return false;
      return wifiSettings.trustedNetworksControl;
    },
    isConnectVPNOnInsecureNetwork: function () {
      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null) return false;
      return wifiSettings.connectVPNOnInsecureNetwork;
    },
    defaultTrustForUndefinedNetworks() {
      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null) return null;
      return wifiSettings.defaultTrustStatusTrusted;
    },
    wifiSSID() {
      const currWifi = this.$store.state.vpnState.currentWiFiInfo;
      if (currWifi == null || currWifi.SSID == null) return "";
      return currWifi.SSID;
    },
    WiFiMarkerText: function () {
      if (this.wifiSSID == "") return null;
      const TRUSTED = "TRUSTED";
      const UNTRUSTED = "UNTRUSTED";
      const INSECURE = "INSECURE";
      const NOTRUSTSTATUS = "NO TRUST STATUS";
      const trustState = this.getTrustInfoForCurrentWifi();
      if (trustState.isTrusted == true) return TRUSTED;
      else if (trustState.isTrusted == false) return UNTRUSTED;
      else if (trustState.isInsecure == true) return INSECURE;
      if (this.isTrustedNetworksControlActive == true) return NOTRUSTSTATUS;
      return null;
    },
    WiFiMarkerColor: function () {
      if (this.wifiSSID == "") return null;
      const TRUSTED = "#64ad07";
      const UNTRUSTED = "#FF6258";
      const INSECURE = "darkorange";
      const NOTRUSTSTATUS = "var(--background-color-alternate)"; //"#BBBBBB";
      const trustState = this.getTrustInfoForCurrentWifi();

      if (trustState.isTrusted == true) return TRUSTED;
      else if (trustState.isTrusted == false) return UNTRUSTED;
      else if (trustState.isInsecure == true) return INSECURE;
      if (this.isTrustedNetworksControlActive == true) return NOTRUSTSTATUS;
      return NOTRUSTSTATUS;
    },
    IsPaused: function () {
      return this.$store.state.vpnState.pauseState == PauseStateEnum.Paused;
    },
    IsConnected: function () {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      );
    },
  },

  methods: {
    async antitrackerOnChecked(antitrackerIsEnabled) {
      try {
        this.antitrackerIsProgress = true;
        await sender.SetDNS(antitrackerIsEnabled);
      } catch (e) {
        processError(e);
      } finally {
        this.antitrackerIsProgress = false;
      }
    },
    async firewallOnChecked(isEnabled) {
      try {
        this.firewallIsProgress = true;
        await sender.EnableFirewall(isEnabled);
      } catch (e) {
        processError(e);
      } finally {
        this.firewallIsProgress = false;
      }
    },
    getCurrentWiFiConfig() {
      let curWifiInfo = this.$store.state.vpnState.currentWiFiInfo;
      if (curWifiInfo == null || curWifiInfo.SSID == "") return null;

      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null || wifiSettings.networks == null) return null;

      for (let w of wifiSettings.networks) {
        if (w.ssid == curWifiInfo.SSID) return w;
      }
    },
    getTrustInfoForCurrentWifi() {
      let ret = { isTrusted: null, isInsecure: null };
      if (this.isTrustedNetworksControlActive) {
        let currentNetworkConfig = this.getCurrentWiFiConfig();
        if (currentNetworkConfig != null)
          ret.isTrusted = currentNetworkConfig.isTrusted;
        else if (this.defaultTrustForUndefinedNetworks != null)
          ret.isTrusted = this.defaultTrustForUndefinedNetworks;
      }

      if (ret.isTrusted == null) {
        let curWifiInfo = this.$store.state.vpnState.currentWiFiInfo;
        if (curWifiInfo != null && curWifiInfo.IsInsecureNetwork)
          ret.isInsecure = true;
      }
      return ret;
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.block {
  @extend .left_panel_block;
}

.datails_text {
  color: var(--text-color-details);
  font-size: 13px;
  line-height: 18px;

  letter-spacing: -0.08px;
  text-transform: uppercase;
}

#connection_header {
  min-height: 51px;
  background: var(--background-color-alternate);
}

.leftPanelBlock {
  @extend .left_panel_block;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

div.blockWithMrgings {
  @extend .left_panel_element;
  margin-top: 18px;
  margin-bottom: 18px;
}
.lowOpacity {
  opacity: 0.5;
}
</style>
