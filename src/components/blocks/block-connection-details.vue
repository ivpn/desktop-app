<template>
  <div>
    <div>
      <div class="horizontalLine" />
      <div id="connection_header">
        <div style="height: 24px;"></div>

        <span class="block datails_text">
          CONNECTION DETAILS
        </span>
      </div>
      <div class="horizontalLine" />
    </div>

    <!-- FIREWALL -->

    <OnOffButtonControl
      text="Firewall"
      description="Ensure that all traffic is routed through VPN"
      :onChecked="firewallOnChecked"
      :isChecked="this.$store.state.vpnState.firewallState.IsEnabled"
      :isProgress="firewallIsProgress"
    />

    <!-- ANTITRACKER -->
    <div class="horizontalLine" />

    <OnOffButtonControl
      text="AntiTracker"
      description="AntiTracker blocks all known data trackers"
      :onChecked="antitrackerOnChecked"
      :isChecked="this.$store.state.settings.isAntitracker"
      :checkedColor="
        this.$store.state.settings.isAntitrackerHardcore ? '#00008B99' : null
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
    <div class="horizontalLine" />

    <SelectButtonControl
      class="leftPanelBlock"
      :click="onShowWifiConfig"
      v-bind:text="$store.state.vpnState.currentWiFiInfo.SSID"
      :description="
        $store.state.vpnState.currentWiFiInfo.SSID == ''
          ? 'No WiFi connection'
          : 'WiFi network'
      "
      :markerText="WiFiMarkerText"
      :markerColor="WiFiMarkerColor"
    />

    <!-- GEOLOCATOIN INFO -->
    <div v-if="$store.state.settings.minimizedUI">
      <div class="horizontalLine" />

      <GeolocationInfoControl class="blockWithMrgings" />
    </div>
  </div>
</template>

<script>
const { dialog, getCurrentWindow } = require("electron").remote;
import OnOffButtonControl from "@/components/controls/control-config-on-off-button.vue";
import SelectButtonControl from "@/components/controls/control-config-to-select-button.vue";
import GeolocationInfoControl from "@/components/controls/control-geolocation-info.vue";
import sender from "@/ipc/renderer-sender";
import { enumValueName } from "@/helpers/helpers";
import { VpnTypeEnum, PortTypeEnum } from "@/store/types";

function processError(e) {
  console.error(e);
  dialog.showMessageBoxSync(getCurrentWindow(), {
    type: "error",
    buttons: ["OK"],
    message: e
  });
}

export default {
  components: {
    OnOffButtonControl,
    SelectButtonControl,
    GeolocationInfoControl
  },
  props: ["onShowPorts", "onShowWifiConfig"],
  data: function() {
    return {
      antitrackerIsProgress: false,
      firewallIsProgress: false
    };
  },

  computed: {
    portProtocolText: function() {
      let port = this.$store.getters["settings/getPort"];
      let protocol = this.$store.getters["settings/vpnType"];
      return `${enumValueName(VpnTypeEnum, protocol)}/${enumValueName(
        PortTypeEnum,
        port.type
      )} ${port.port}`;
    },
    isTrustedNetworksControlActive() {
      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null || wifiSettings.networks == null) return false;
      return wifiSettings.trustedNetworksControl;
    },
    defaultTrustForUndefinedNetworks() {
      let wifiSettings = this.$store.state.settings.wifi;
      if (wifiSettings == null) return null;
      return wifiSettings.defaultTrustStatusTrusted;
    },
    WiFiMarkerText: function() {
      const TRUSTED = "TRUSTED";
      const UNTRUSTED = "UNTRUSTED";
      const INSECURE = "INSECURE";
      const trustState = this.getTrustInfoForCurrentWifi();
      if (trustState.isTrusted == true) return TRUSTED;
      else if (trustState.isTrusted == false) return UNTRUSTED;
      else if (trustState.isInsecure == true) return INSECURE;
      return null;
    },
    WiFiMarkerColor: function() {
      const TRUSTED = "#64ad07";
      const UNTRUSTED = "#FF6258";
      const INSECURE = "orange";
      const trustState = this.getTrustInfoForCurrentWifi();
      if (trustState.isTrusted == true) return TRUSTED;
      else if (trustState.isTrusted == false) return UNTRUSTED;
      else if (trustState.isInsecure == true) return INSECURE;
      return null;
    }
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
      } else {
        let curWifiInfo = this.$store.state.vpnState.currentWiFiInfo;
        if (curWifiInfo != null && curWifiInfo.IsInsecureNetwork)
          ret.isInsecure = true;
      }
      return ret;
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.block {
  @extend .left_panel_block;
}

.datails_text {
  color: $base-text-color-details;
  font-size: 13px;
  line-height: 18px;

  letter-spacing: -0.08px;
  text-transform: uppercase;
}

#connection_header {
  min-height: 51px;
  background: #f2f3f6;
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
</style>
