<template>
  <div>
    <div class="settingsTitle">CONNECTION SETTINGS</div>
    <div class="settingsBoldFont">
      VPN protocol:
    </div>

    <div>
      <div class="settingsRadioBtn">
        <input
          type="radio"
          id="openvpn"
          name="vpntype"
          value="openvpn"
          :checked="isOpenVPN"
          @click="onVpnChange"
        />
        <label class="defColor" for="openvpn">OpenVPN</label>
      </div>

      <div class="settingsRadioBtn">
        <input
          type="radio"
          id="wireguard"
          name="vpntype"
          value="wireguard"
          :checked="!isOpenVPN"
          @click="onVpnChange"
        />
        <label class="defColor" for="wireguard">Wireguard</label>
      </div>
    </div>

    <!-- IPv6 -->
    <div>
      <div class="param">
        <input
          type="checkbox"
          id="enableIPv6InTunnel"
          v-model="enableIPv6InTunnel"
          :disabled="!isCanUseIPv6InTunnel"
        />
        <label class="defColor" for="enableIPv6InTunnel"
          >Enable IPv6 in VPN tunnel</label
        >
      </div>

      <div class="param">
        <input
          type="checkbox"
          id="showGatewaysWithoutIPv6"
          v-model="showGatewaysWithoutIPv6"
          :disabled="!isCanUseIPv6InTunnel || enableIPv6InTunnel === false"
        />
        <label class="defColor" for="showGatewaysWithoutIPv6"
          >Show servers without IPv6 support</label
        >
      </div>
    </div>

    <!-- OpenVPN -->
    <div v-if="isOpenVPN">
      <div class="settingsBoldFont">OpenVPN configuration:</div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Preferred port:</div>
        <select v-model="port" style="background: var(--background-color);">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.text"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div v-bind:class="{ disabled: connectionUseObfsproxy }">
        <div class="flexRow paramBlock">
          <div class="defColor paramName">
            Network proxy:
          </div>
          <div class="settingsRadioBtnProxy">
            <input
              type="radio"
              id="proxyNone"
              name="proxy"
              v-model="ovpnProxyType"
              value=""
            />
            <label class="defColor" for="proxyNone">None</label>
          </div>
          <div class="settingsRadioBtnProxy">
            <input
              type="radio"
              id="proxyHTTP"
              name="proxy"
              v-model="ovpnProxyType"
              value="http"
            />
            <label class="defColor" for="proxyHTTP">HTTP</label>
          </div>
          <div class="settingsRadioBtnProxy">
            <input
              type="radio"
              id="proxySocks"
              name="proxy"
              v-model="ovpnProxyType"
              value="socks"
            />
            <label class="defColor" for="proxySocks">Socks</label>
          </div>
        </div>

        <div v-bind:class="{ disabled: ovpnProxyType.length == 0 }">
          <div class="flexRow">
            <div class="paramBlockText">
              <div>Server:</div>
              <input
                class="settingsTextInput proxyParam"
                placeholder="0.0.0.0"
                v-model="ovpnProxyServer"
              />
            </div>
            <div class="paramBlockText">
              <div>Port:</div>
              <input
                class="settingsTextInput proxyParam"
                v-model="ovpnProxyPort"
              />
            </div>
            <div class="paramBlockText">
              <div>Login:</div>
              <input
                class="settingsTextInput proxyParam"
                v-model="ovpnProxyUser"
              />
            </div>
            <div class="paramBlockText">
              <div>Password:</div>
              <input
                type="password"
                class="settingsTextInput proxyParam"
                v-model="ovpnProxyPass"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="settingsBoldFont">Additional settings:</div>
      <div class="param">
        <input
          type="checkbox"
          id="connectionUseObfsproxy"
          v-model="connectionUseObfsproxy"
        />
        <label class="defColor" for="connectionUseObfsproxy"
          >Use obfsproxy</label
        >
      </div>
      <div class="description">
        Only enable if you have trouble connecting
      </div>

      <div class="param" v-if="userDefinedOvpnFile">
        <input
          type="checkbox"
          id="openvpnManualConfig"
          v-model="openvpnManualConfig"
        />
        <label class="defColor" for="openvpnManualConfig"
          >Add additional OpenVPN configuration parameters</label
        >
      </div>
      <div v-if="openvpnManualConfig && userDefinedOvpnFile">
        <div class="description">
          <div class="settingsGrayLongDescriptionFont">
            Please be aware that this is a feature for advanced users as adding
            parameters may affect the proper functioning and security of the VPN
            tunnel
          </div>
          <button
            style="margin-top: 4px"
            class="settingsButton"
            v-on:click="onVPNConfigFileLocation"
          >
            Open configuration file location ...
          </button>
          <div style="max-width: 500px; margin: 0px; padding: 0px;">
            <div
              class="settingsGrayLongDescriptionFont selectable"
              style=" margin-top:5px; font-size: 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;"
            >
              {{ userDefinedOvpnFile }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Wireguard -->
    <div v-if="!isOpenVPN">
      <div class="settingsBoldFont">Wireguard configuration:</div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Preferred port:</div>
        <select v-model="port" style="background: var(--background-color);">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.text"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Rotate key every:</div>
        <select
          v-model="wgKeyRegenerationInterval"
          style="background: var(--background-color);"
        >
          <option
            v-for="item in wgRegenerationIntervals"
            :value="item.seconds"
            :key="item.seconds"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div v-if="IsAccountActive">
        <div class="settingsBoldFont">Wireguard key information:</div>

        <spinner :loading="isProcessing" />
        <div class="flexRow paramBlock">
          <div class="defColor paramName">Local IP:</div>
          <div class="detailedParamValue">
            {{ this.$store.state.account.session.WgLocalIP }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">Public key:</div>
          <div class="detailedParamValue">
            {{ this.$store.state.account.session.WgPublicKey }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">Generated:</div>
          <div class="detailedParamValue">
            {{ wgKeysGeneratedDateStr }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">
            Expiration date:
          </div>
          <div class="detailedParamValue">
            {{ wgKeysExpirationDateStr }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">
            Will be automatically rotated:
          </div>
          <div class="detailedParamValue">
            {{ wgKeysWillBeRegeneratedStr }}
          </div>
        </div>

        <button
          class="settingsButton paramBlock"
          style="margin-top: 10px; height: 24px;"
          v-on:click="onWgKeyRegenerate"
        >
          Regenerate
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import spinner from "@/components/controls/control-spinner.vue";
import { VpnTypeEnum, VpnStateEnum, PortTypeEnum, Ports } from "@/store/types";
import { enumValueName } from "@/helpers/helpers";
const sender = window.ipcSender;
import { dateDefaultFormat } from "@/helpers/helpers";

export default {
  components: {
    spinner
  },
  data: function() {
    return {
      isProcessing: false,
      openvpnManualConfig: false
    };
  },
  methods: {
    isAbleToChangeVpnSettings: function() {
      if (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      )
        return true;

      sender.showMessageBoxSync({
        type: "info",
        buttons: ["OK"],
        message: "You are now connected to IVPN",
        detail:
          "You can change VPN protocol settings only when IVPN is disconnected."
      });

      return false;
    },
    onVpnChange: function(e) {
      if (this.isAbleToChangeVpnSettings() != true) {
        e.preventDefault();
        return;
      }

      let type = VpnTypeEnum.OpenVPN;
      if (e.target.value === "wireguard") type = VpnTypeEnum.WireGuard;
      else type = VpnTypeEnum.OpenVPN;
      this.$store.dispatch("settings/vpnType", type);
    },
    onVPNConfigFileLocation: function() {
      const file = this.userDefinedOvpnFile;
      if (file) sender.shellShowItemInFolder(file);
    },
    onWgKeyRegenerate: async function() {
      try {
        this.isProcessing = true;
        await sender.WgRegenerateKeys();
      } catch (e) {
        console.log(`ERROR: ${e}`);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: "Error generating WireGuard keys",
          detail: "Please check your internet connection"
        });
      } finally {
        this.isProcessing = false;
      }
    },
    getWgKeysGenerated: function() {
      if (
        this.$store.state.account == null ||
        this.$store.state.account.session == null ||
        this.$store.state.account.session.WgKeyGenerated == null
      )
        return null;
      return new Date(this.$store.state.account.session.WgKeyGenerated);
    },
    formatDate: function(d) {
      if (d == null) return null;
      return dateDefaultFormat(d);
    }
  },
  computed: {
    isCanUseIPv6InTunnel: function() {
      return this.$store.getters["isCanUseIPv6InTunnel"];
    },
    enableIPv6InTunnel: {
      get() {
        return this.$store.state.settings.enableIPv6InTunnel;
      },
      set(value) {
        let isCanChange = this.isAbleToChangeVpnSettings();
        this.$store.dispatch("settings/enableIPv6InTunnel", value);

        if (isCanChange != true) {
          this.$store.dispatch("settings/enableIPv6InTunnel", !value);
        }
      }
    },
    showGatewaysWithoutIPv6: {
      get() {
        return this.$store.state.settings.showGatewaysWithoutIPv6;
      },
      set(value) {
        this.$store.dispatch("settings/showGatewaysWithoutIPv6", value);
      }
    },
    IsAccountActive: function() {
      // if no info about account status - let's believe that account is active
      if (
        !this.$store.state.account ||
        !this.$store.state.account.accountStatus
      )
        return true;
      return this.$store.state.account?.accountStatus?.Active === true;
    },
    port: {
      get() {
        return this.$store.getters["settings/getPort"];
      },
      set(value) {
        this.$store.dispatch("settings/setPort", value);
      }
    },
    userDefinedOvpnFile: function() {
      if (!this.$store.state.configParams) return null;
      return this.$store.state.configParams.UserDefinedOvpnFile;
    },
    wgKeyRegenerationInterval: {
      get() {
        return this.$store.state.account.session.WgKeysRegenIntervalSec;
      },
      set(value) {
        // daemon will send back a Hello response with updated 'session.WgKeysRegenIntervalSec'
        sender.WgSetKeysRotationInterval(value);
      }
    },
    connectionUseObfsproxy: {
      get() {
        return this.$store.state.settings.connectionUseObfsproxy;
      },
      set(value) {
        this.$store.dispatch("settings/connectionUseObfsproxy", value);
        sender.SetObfsproxy();
      }
    },

    ovpnProxyType: {
      get() {
        return this.$store.state.settings.ovpnProxyType;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyType", value);
      }
    },
    ovpnProxyServer: {
      get() {
        return this.$store.state.settings.ovpnProxyServer;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyServer", value);
      }
    },
    ovpnProxyPort: {
      get() {
        return this.$store.state.settings.ovpnProxyPort;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyPort", value);
      }
    },
    ovpnProxyUser: {
      get() {
        return this.$store.state.settings.ovpnProxyUser;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyUser", value);
      }
    },
    ovpnProxyPass: {
      get() {
        return this.$store.state.settings.ovpnProxyPass;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyPass", value);
      }
    },

    isOpenVPN: function() {
      return this.$store.state.settings.vpnType === VpnTypeEnum.OpenVPN;
    },
    wgKeysGeneratedDateStr: function() {
      return this.formatDate(this.getWgKeysGenerated());
    },
    wgKeysWillBeRegeneratedStr: function() {
      let t = this.getWgKeysGenerated();
      if (t == null) return null;
      t.setSeconds(
        t.getSeconds() +
          this.$store.state.account.session.WgKeysRegenIntervalSec
      );
      return this.formatDate(t);
    },
    wgKeysExpirationDateStr: function() {
      let t = this.getWgKeysGenerated();
      if (t == null) return null;
      t.setSeconds(t.getSeconds() + 40 * 24 * 60 * 60); // 40 days
      return this.formatDate(t);
    },
    wgRegenerationIntervals: function() {
      let ret = [{ text: "1 day", seconds: 24 * 60 * 60 }];
      for (let i = 2; i <= 30; i++) {
        ret.push({ text: `${i} days`, seconds: i * 24 * 60 * 60 });
      }
      return ret;
    },
    prefferedPorts: function() {
      let data = this.isOpenVPN ? Ports.OpenVPN : Ports.WireGuard;
      let ret = [];
      data.forEach(p =>
        ret.push({
          text: `${enumValueName(PortTypeEnum, p.type)} ${p.port}`,
          port: p
        })
      );

      return ret;
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/platform/base";

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

div.paramBlock {
  @extend .flexRow;
  margin-top: 10px;
}

div.paramBlockDetailedConfig {
  @extend .flexRow;
  margin-top: 5px;
}
div.detailedConfigBlock {
  margin-left: 22px;
  max-width: 325px;
}
div.detailedConfigBlock input {
  width: 100%;
}
div.detailedConfigBlock select {
  width: 100%;
}
div.detailedConfigParamBlock {
  @extend .flexRow;
  margin-top: 10px;
  width: 100%;
}
div.detailedParamValue {
  opacity: 0.7;

  overflow-wrap: break-word;
  -webkit-user-select: text;
  letter-spacing: 0.1px;
}

div.paramName {
  min-width: 161px;
  max-width: 161px;
}

div.settingsRadioBtnProxy {
  @extend .settingsRadioBtn;
  padding-right: 20px;
}

div.paramBlockText {
  margin-top: 16px;
  margin-right: 21px;
}

select {
  background: linear-gradient(180deg, #ffffff 0%, #ffffff 100%);
  border: 0.5px solid rgba(0, 0, 0, 0.2);
  border-radius: 3.5px;
  width: 186px;
}

div.description {
  @extend .settingsGrayLongDescriptionFont;
  margin-left: 22px;
}

input.proxyParam {
  width: 100px;
}

div.disabled {
  pointer-events: none;
  opacity: 0.5;
}
</style>
