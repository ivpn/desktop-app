<template>
  <div>
    <div class="settingsTitle">NETWORKS</div>

    <div class="param">
      <input
        type="checkbox"
        id="trustedNetworksControl"
        v-model="trustedNetworksControl"
      />
      <label class="defColor" for="trustedNetworksControl"
        >Trusted\Untrusted networks control</label
      >
    </div>
    <div class="fwDescription">
      By enabling this feature you can define a network as trusted or untrusted
      and what actions to take when joining the network
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
        style="cursor: auto; flex-grow: 1;"
        class="selectableButtonSeparator"
      ></button>
    </div>
    <div>
      <!-- ACTIONS -->
      <div v-if="isActionsView">
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
      <div v-if="!isActionsView">
        <div style="margin-top: 12px; margin-bottom:12px">
          Default trust status for undefined networks:
        </div>
        <div class="horizontalLine" style="margin-bottom: 8px" />

        <div v-for="wifi of networks" v-bind:key="wifi.SSID">
          <trustedNetConfigControl
            :wifiInfo="wifi"
            :onChange="onNetworkTrustChanged"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import trustedNetConfigControl from "@/components/controls/control-trusted-network-config.vue";
import sender from "@/ipc/renderer-sender";

export default {
  components: {
    trustedNetConfigControl
  },
  mounted() {
    sender.GetWiFiAvailableNetworks();
  },
  data: function() {
    return {
      isActionsView: false
    };
  },
  methods: {
    onActions() {
      this.isActionsView = true;
    },
    onNetworks() {
      this.isActionsView = false;
    },
    onNetworkTrustChanged(ssid, isTrusted) {
      let wifi = Object.assign({}, this.$store.state.settings.wifi);
      var nets = [];
      if (this.$store.state.settings.wifi?.networks != null)
        nets = [...this.$store.state.settings.wifi.networks];
      if (isTrusted == null) {
        nets = nets.filter(wifi => wifi.ssid != ssid);
      } else {
        let alreadyExists = nets.filter(wifi => wifi.ssid == ssid);
        if (alreadyExists != null && alreadyExists.length > 0) {
          // replace item with a new value
          nets = [
            ...nets.map(item =>
              item.ssid !== ssid ? item : { ssid: ssid, isTrusted: isTrusted }
            )
          ];
        } else nets.push({ ssid: ssid, isTrusted: isTrusted });
      }
      wifi.networks = nets;

      console.log(nets, wifi, ssid, isTrusted);
      this.$store.dispatch("settings/wifi", wifi);
    }
  },
  computed: {
    networks: function() {
      var nets = [];
      if (this.$store.state.settings.wifi?.networks != null)
        nets = [...this.$store.state.settings.wifi.networks];

      let currWiFi = this.$store.state.vpnState.currentWiFiInfo;
      if (currWiFi != null && currWiFi.SSID != "") {
        let alreadyExists = nets.filter(wifi => wifi.ssid == currWiFi.SSID);

        // check is curent wifi already exists
        if (alreadyExists == null || alreadyExists.length == 0)
          nets.unshift({ ssid: currWiFi.SSID, isTrusted: null });

        // add rest of available networks
        let restNetworks = this.$store.state.vpnState.availableWiFiNetworks;
        if (restNetworks != null) {
          for (let w of restNetworks) {
            if (w.SSID != "" && nets.findIndex(t => t.ssid === w.SSID) == -1)
              nets.push({ ssid: w.SSID, isTrusted: null });
          }
        }
      }
      return nets;
    },
    trustedNetworksControl: {
      get() {
        return this.$store.state.settings.wifi?.trustedNetworksControl;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        wifi.trustedNetworksControl = value;
        this.$store.dispatch("settings/wifi", wifi);
      }
    },
    unTrustedConnectVpn: {
      get() {
        return this.$store.state.settings.wifi?.actions?.unTrustedConnectVpn;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.unTrustedConnectVpn = value;
        this.$store.dispatch("settings/wifi", wifi);
      }
    },
    unTrustedEnableFirewall: {
      get() {
        return this.$store.state.settings.wifi?.actions
          ?.unTrustedEnableFirewall;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.unTrustedEnableFirewall = value;
        this.$store.dispatch("settings/wifi", wifi);
      }
    },
    trustedDisconnectVpn: {
      get() {
        return this.$store.state.settings.wifi?.actions?.trustedDisconnectVpn;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.trustedDisconnectVpn = value;
        this.$store.dispatch("settings/wifi", wifi);
      }
    },
    trustedDisableFirewall: {
      get() {
        return this.$store.state.settings.wifi?.actions?.trustedDisableFirewall;
      },
      set(value) {
        let wifi = Object.assign({}, this.$store.state.settings.wifi);
        if (wifi.actions == null) wifi.actions = {};
        wifi.actions.trustedDisableFirewall = value;
        this.$store.dispatch("settings/wifi", wifi);
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

button.selectableButtonOff {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  height: 38px;

  font-style: normal;
  font-size: 11px;
  line-height: 13px;

  color: #2a394b;

  border-bottom: 2px solid #d9e0e5;
}

button.selectableButtonOn {
  @extend .selectableButtonOff;
  border-bottom: 2px solid #449cf8;
}

button.selectableButtonSeparator {
  @extend .selectableButtonOff;
  cursor: auto;
}
</style>
