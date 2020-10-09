<template>
  <div>
    <div class="settingsTitle">CONNECTION</div>
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

    <!-- OpenVPN -->
    <div v-if="isOpenVPN">
      <div class="configTitle">OpenVPN configuration</div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Preffered port:</div>
        <select v-model="port">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.text"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div class="flexRow paramBlock" v-if="!connectionUseObfsproxy">
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

      <div v-if="ovpnProxyType.length > 0 && !connectionUseObfsproxy">
        <div class="flexRow">
          <div class="paramBlockText">
            <div>Server:</div>
            <input
              class="settingsTextInput"
              placeholder="0.0.0.0"
              v-model="ovpnProxyServer"
            />
          </div>
          <div class="paramBlockText">
            <div>Port:</div>
            <input class="settingsTextInput" v-model="ovpnProxyPort" />
          </div>
        </div>
        <div class="flexRow">
          <div class="paramBlockText">
            <div>Login:</div>
            <input class="settingsTextInput" v-model="ovpnProxyUser" />
          </div>
          <div class="paramBlockText">
            <div>Password:</div>
            <input class="settingsTextInput" v-model="ovpnProxyPass" />
          </div>
        </div>
      </div>

      <div class="configTitle">Additional</div>
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

      <div class="param" v-if="userDefinedOvpnFile">
        <input
          type="checkbox"
          id="openvpnManualConfig"
          v-model="openvpnManualConfig"
        />
        <label class="defColor" for="openvpnManualConfig"
          >Show manual configuration parameters info</label
        >
      </div>
      <div v-if="openvpnManualConfig && userDefinedOvpnFile">
        <div class="description">
          Please be aware that adding parameters may affect the proper
          functioning of the VPN tunnel. Only add parameters if you understand
          what you are doing.
          <button
            class="btn settingsGrayLongDescriptionFont"
            v-on:click="onVPNConfigFileLocation"
          >
            Open configuration file location ...
          </button>
          <div align="right">
            <label
              class="settingsGrayLongDescriptionFont selectable"
              align="right"
              style="margin-top:5px; font-size: 10px;"
            >
              {{ userDefinedOvpnFile }}
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- Wireguard -->
    <div v-if="!isOpenVPN">
      <div class="configTitle">Wireguard configuration</div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Preffered port:</div>
        <select v-model="port">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.text"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Regenerate key every:</div>
        <select v-model="wgKeyRegenerationInterval">
          <option
            v-for="item in wgRegenerationIntervals"
            :value="item.seconds"
            :key="item.seconds"
            >{{ item.text }}</option
          >
        </select>
      </div>

      <div class="paramBlockDetailedConfig" v-if="IsAccountActive">
        <input
          type="checkbox"
          id="wgConfigDetailed"
          v-model="wgConfigDetailed"
        />
        <label class="defColor" for="wgConfigDetailed"
          >View detailed configuration</label
        >
      </div>

      <div
        class="detailedConfigBlock"
        v-if="wgConfigDetailed && IsAccountActive"
      >
        <spinner :loading="isProcessing" />
        <div class="flexRow detailedConfigParamBlock">
          <div class="defColor paramName">Local IP:</div>
          <div class="detailedParamValue">
            {{ this.$store.state.account.session.WgLocalIP }}
          </div>
        </div>
        <div class="flexRow detailedConfigParamBlock">
          <div class="defColor paramName">Public key:</div>
          <div class="detailedParamValue">
            {{ this.$store.state.account.session.WgPublicKey }}
          </div>
        </div>
        <div class="flexRow detailedConfigParamBlock">
          <div class="defColor paramName">Generated:</div>
          <div class="detailedParamValue">
            {{ wgKeysGeneratedDateStr }}
          </div>
        </div>
        <div class="flexRow detailedConfigParamBlock">
          <div class="defColor paramName">
            Expiration date:
          </div>
          <div class="detailedParamValue">
            {{ wgKeysExpirationDateStr }}
          </div>
        </div>
        <div class="flexRow detailedConfigParamBlock">
          <div class="defColor paramName">
            Will be automatically regenerated:
          </div>
          <div class="detailedParamValue">
            {{ wgKeysWillBeRegeneratedStr }}
          </div>
        </div>

        <button class="btn" v-on:click="onWgKeyRegenerate">
          Regenerate
        </button>
      </div>
    </div>
  </div>
</template>

<script>
const { dialog, getCurrentWindow, shell } = require("electron").remote;

import spinner from "@/components/controls/control-spinner.vue";
import { VpnTypeEnum, VpnStateEnum, PortTypeEnum, Ports } from "@/store/types";
import { enumValueName } from "@/helpers/helpers";
import sender from "@/ipc/renderer-sender";
import { dateDefaultFormat } from "@/helpers/helpers";

export default {
  components: {
    spinner
  },
  data: function() {
    return {
      isProcessing: false,
      wgConfigDetailed: false,
      openvpnManualConfig: false
    };
  },
  methods: {
    onVpnChange: function(e) {
      if (
        this.$store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED
      ) {
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "info",
          buttons: ["OK"],
          message: "You are now connected to IVPN",
          detail:
            "You can change VPN protocol settings only when IVPN is disconnected."
        });
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
      if (file) shell.showItemInFolder(file);
    },
    onWgKeyRegenerate: async function() {
      if (
        this.$store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED
      ) {
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "info",
          buttons: ["OK"],
          message: "You are now connected to IVPN",
          detail:
            "You can regenerate WireGuard keys only when IVPN is disconnected."
        });
        return;
      }

      try {
        this.isProcessing = true;
        await sender.WgRegenerateKeys();
      } catch (e) {
        console.log(`ERROR: ${e}`);
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "error",
          buttons: ["OK"],
          message: "Error generating WireGuard keys",
          detail: `${e.message}`
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
    IsAccountActive: function() {
      return this.$store.state.account.accountStatus.Active;
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
  margin-top: 18px;
}

div.paramBlockDetailedConfig {
  @extend .flexRow;
  margin-top: 32px;
  font-size: 12px;
  opacity: 0.8;
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
  font-size: 11px;
}
div.detailedParamValue {
  opacity: 0.7;
  max-width: 165px;
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

.configTitle {
  font-weight: 500;
  font-size: 13px;
  line-height: 18px;
  letter-spacing: -0.08px;
  color: #2a394b;
  opacity: 0.5;
  margin-top: 42px;
  margin-bottom: 12px;
}

.btn {
  margin-top: 10px;
  width: 100%;
  height: 24px;

  background: transparent;
  border: 0.5px solid #c8c8c8;
  box-sizing: border-box;
  border-radius: 4px;
  cursor: pointer;
}

div.description {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
}
</style>
