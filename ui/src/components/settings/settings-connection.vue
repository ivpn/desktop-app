<template>
  <div>
    <ComponentDialog ref="addCustomPortDlg" noCloseButtons>
      <ComponentAddCustomPort
        style="margin: 0px"
        :onClose="onCloseAddCustomPortDlg"
      />
    </ComponentDialog>

    <div class="settingsTitle">CONNECTION SETTINGS</div>

    <div class="settingsBoldFont">VPN protocol:</div>

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

      <div
        class="flexRow paramBlock"
        v-bind:class="{ disabled: prefferedPorts.length <= 1 }"
      >
        <div class="defColor paramName">Preferred port:</div>
        <select v-model="port">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.key"
          >
            {{ item.text }}
          </option>
          <option
            v-if="isShowAddPortOption"
            key="keyAddCustomPort"
            value="valueAddCustomPort"
          >
            Add custom port ...
          </option>
        </select>
      </div>

      <div
        v-bind:class="{
          disabled:
            connectionUseObfsproxy || V2RayType !== 0 || !isDisconnected,
        }"
      >
        <div class="flexRow paramBlock">
          <div class="defColor paramName">Network proxy:</div>
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
      <div v-bind:class="{ disabled: !isDisconnected }">
        <div class="flexRowAlignTop">
          <div class="flexRowAlignTop paramName" style="padding-top: 2px">
            <label class="defColor">Obfuscation:</label>
          </div>

          <div>
            <div class="flexRow">
              <select v-model="obfuscationType">
                <option
                  v-for="item in obfuscationTypes"
                  :value="item"
                  :key="item.text"
                >
                  {{ item.text }}
                </option>
              </select>

              <button
                class="noBordersBtn flexRow"
                title="Help"
                v-on:click="onShowHelpObfsproxy"
              >
                <img src="@/assets/question.svg" />
              </button>
            </div>
            <div class="description" style="margin-left: 0px">
              Only enable if you have trouble connecting.
            </div>
          </div>
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
              Please be aware that this is a feature for advanced users as
              adding parameters may affect the proper functioning and security
              of the VPN tunnel
            </div>
            <button
              style="margin-top: 4px"
              class="settingsButton"
              v-on:click="onVPNConfigFileLocation"
            >
              Open configuration file location ...
            </button>
            <!--
          <div style="max-width: 500px; margin: 0px; padding: 0px">
            <div
              class="settingsGrayLongDescriptionFont selectable"
              style="
                margin-top: 5px;
                font-size: 10px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
              "
            >
              {{ userDefinedOvpnFile }}
            </div>
          </div>
          --></div>
        </div>
      </div>
    </div>

    <!-- Wireguard -->
    <div v-show="!isOpenVPN">
      <div class="settingsBoldFont">Wireguard configuration:</div>

      <div
        v-bind:class="{ disabled: prefferedPorts.length <= 1 }"
        class="flexRow paramBlock"
      >
        <div class="defColor paramName">Preferred port:</div>
        <select v-model="port">
          <option
            v-for="item in prefferedPorts"
            :value="item.port"
            :key="item.key"
          >
            {{ item.text }}
          </option>
          <option
            v-if="isShowAddPortOption"
            key="keyAddCustomPort"
            value="valueAddCustomPort"
          >
            Add custom port ...
          </option>
        </select>
      </div>

      <div class="flexRow paramBlock">
        <div class="defColor paramName">Rotate key every:</div>
        <select class="defInputWidth" v-model="wgKeyRegenerationInterval">
          <option
            v-for="item in wgRegenerationIntervals"
            :value="item.seconds"
            :key="item.seconds"
          >
            {{ item.text }}
          </option>
        </select>
      </div>

      <div
        class="flexRow paramBlock"
        v-bind:class="{ disabled: !isDisconnected }"
      >
        <div class="flexRow paramName">
          <label class="defColor">Obfuscation:</label>
        </div>

        <div>
          <div class="flexRow">
            <select v-model="obfuscationType">
              <option
                v-for="item in obfuscationTypes"
                :value="item"
                :key="item.text"
              >
                {{ item.text }}
              </option>
            </select>

            <button
              class="noBordersBtn flexRow"
              title="Help"
              v-on:click="onShowHelpObfsproxy"
            >
              <img src="@/assets/question.svg" />
            </button>
          </div>
        </div>
      </div>

      <div v-bind:class="{ disabled: !isDisconnected }">
        <div class="flexRow paramBlock" style="margin: 0px; margin-top: 2px">
          <div class="defColor paramName">Custom MTU:</div>
          <div>
            <input
              ref="mtuInput"
              v-model="mtu"
              type="number"
              step="1"
              style="width: 165px"
              class="settingsTextInput"
              title="Valid range [1280 - 65535]. Please note that changing this value may affect the proper functioning of the VPN tunnel."
            />
          </div>
          <div
            v-if="isMtuBadValue"
            class="description"
            style="margin-top: 4px; margin-left: 4px; width: 180px; color: red"
          >
            Expected value: [1280 - 65535]
          </div>
        </div>
        <div class="flexRow">
          <div class="paramName" />
          <div class="description" style="margin-left: 0px">
            Leave blank to use default value
          </div>
        </div>
      </div>

      <div v-if="IsAccountActive">
        <div class="settingsBoldFont">Wireguard key information:</div>

        <spinner :loading="isProcessing" />
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">Local IP Address:</div>
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
          <div class="defColor paramName">Scheduled rotation date:</div>
          <div class="detailedParamValue">
            {{ wgKeysWillBeRegeneratedStr }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">Expiration date:</div>
          <div class="detailedParamValue">
            {{ wgKeysExpirationDateStr }}
          </div>
        </div>
        <div class="flexRow paramBlockDetailedConfig">
          <div class="defColor paramName">Quantum Resistance:</div>
          <div class="detailedParamValue">
            {{ wgQuantumResistanceStr }}
          </div>
          <button
            class="noBordersBtn flexRow"
            title="Info"
            v-on:click="this.$refs.infoWgQuantumResistance.showModal()"
          >
            <img src="@/assets/question.svg" />
          </button>
        </div>
        <ComponentDialog ref="infoWgQuantumResistance" header="Info">
          <div>
            <p>
              Quantum Resistance: Indicates whether your current WireGuard VPN
              connection is using additional protection measures against
              potential future quantum computer attacks.
            </p>
            <p>
              When Enabled, a Pre-shared key has been securely exchanged between
              your device and the server using post-quantum Key Encapsulation
              Mechanism (KEM) algorithms. If Disabled, the current VPN
              connection, while secure under today's standards, does not include
              this extra layer of quantum resistance.
            </p>
          </div>
        </ComponentDialog>

        <button
          class="settingsButton paramBlock"
          style="margin-top: 10px; height: 24px"
          v-on:click="onWgKeyRegenerate"
        >
          Regenerate
        </button>
      </div>
    </div>

    <!-- Help dialogs -->
    <ComponentDialog ref="helpDialogObfsproxy" header="Info">
      <div>
        <p>
          VPN obfuscation is a technique that masks VPN traffic to make it
          appear like standard internet traffic, helping to evade detection and
          bypass internet restrictions or censorship.
        </p>
        <p v-if="isOpenVPN">
          When using OpenVPN we offer two solutions, V2Ray and Obfsproxy. Both
          solutions generally work well but you may find one solution is more
          performant and/or reliable depending on multiple variables relating to
          your location and the path your traffic takes to the VPN server. We
          recommend experimenting with both Obfsproxy and V2Ray options.
        </p>
        <p v-else>
          When using WireGuard we offer the powerful V2Ray proxy protocol. It is
          available in two variants, you may find one is more performant and/or
          reliable depending on multiple variables relating to your location and
          the path your traffic takes to the VPN server. We recommend
          experimenting with both variants.
        </p>
        <!--<div class="horizontalLine" />-->
        <div v-show="isOpenVPN">
          <p>
            <b>Obfsproxy options</b><br /><b>Note:</b> The Inter-Arrival Time
            (<b>IAT</b>) parameter in <b>obfs4</b> will negatively affect
            overall VPN speed and CPU load. The options below are listed in
            order from highest performance and least stealthy to lowest
            performance and most stealthy.
          </p>
          <ul>
            <li>
              <b>obfs4</b> - The IAT-mode is disabled. Large packets will be
              split by the network drivers which may result in network
              fingerprints that could be detected by censors.
            </li>
            <li>
              <b>obfs4 (IAT1)</b> - Large packets will be split into MTU-size
              packets by Obfsproxy (instead of the network drivers), resulting
              in smaller packets that are more resistant to being reassembled
              for analysis and censoring.
            </li>
            <li>
              <b>obfs4 (IAT2)</b> - (paranoid mode) - Large packets will be
              split into variable size packets by Obfsproxy.
            </li>
          </ul>
        </div>
        <div>
          <p>
            <b>V2Ray</b>
          </p>
          <ul>
            <li>
              <b>V2Ray (VMESS/QUIC)</b> is a modern protocol designed to provide
              robust security and high performance, while reducing latency
              compared to traditional protocols. It makes your data appear as
              regular HTTPS traffic.
            </li>
            <li>
              <b>V2Ray (VMESS/TCP)</b> is a traditional, widely-used protocol
              that guarantees reliable, ordered data delivery. It makes your
              data appear as regular HTTP traffic.
            </li>
          </ul>
        </div>
      </div>
    </ComponentDialog>
  </div>
</template>

<script>
import spinner from "@/components/controls/control-spinner.vue";
import {
  VpnTypeEnum,
  VpnStateEnum,
  PortTypeEnum,
  ObfsproxyVerEnum,
  Obfs4IatEnum,
  V2RayObfuscationEnum,
} from "@/store/types";

import { enumValueName, dateDefaultFormat } from "@/helpers/helpers";
import { SetInputFilterNumbers } from "@/helpers/renderer";
import ComponentDialog from "@/components/component-dialog.vue";
import ComponentAddCustomPort from "@/views/dialogs/addCustomPort.vue";

const sender = window.ipcSender;

export default {
  components: {
    spinner,
    ComponentDialog,
    ComponentAddCustomPort,
  },
  data: function () {
    return {
      isPortModified: false,
      isProcessing: false,
      openvpnManualConfig: false,
    };
  },
  mounted() {
    SetInputFilterNumbers(this.$refs.mtuInput);
  },
  watch: {
    // If port was changed in conneted state - reconnect
    async port(newValue, oldValue) {
      if (this.isPortModified === false) return;
      if (newValue == null || oldValue == null) return;
      if (newValue.port === oldValue.port && newValue.type === oldValue.type)
        return;
      await this.reconnect();
    },
  },

  methods: {
    async reconnect() {
      if (
        !this.$store.getters["vpnState/isConnected"] &&
        !this.$store.getters["vpnState/isConnecting"]
      )
        return; // not connected. Reconnection not required

      // Re-connect
      try {
        await sender.Connect();
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to connect: ` + e,
        });
      }
    },

    onVpnChange: async function (e) {
      let type = VpnTypeEnum.OpenVPN;
      if (e.target.value === "wireguard") type = VpnTypeEnum.WireGuard;
      else type = VpnTypeEnum.OpenVPN;

      if (type === this.$store.state.settings.vpnType) return;

      this.$store.dispatch("settings/vpnType", type);

      await this.reconnect();
    },
    onVPNConfigFileLocation: function () {
      const file = this.userDefinedOvpnFile;
      if (file) sender.shellShowItemInFolder(file);
    },
    onWgKeyRegenerate: async function () {
      try {
        this.isProcessing = true;
        await sender.WgRegenerateKeys();
      } catch (e) {
        console.log(`ERROR: ${e}`);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: "Error generating WireGuard keys",
          detail: e,
        });
      } finally {
        this.isProcessing = false;
      }
    },
    getWgKeysGenerated: function () {
      if (
        this.$store.state.account == null ||
        this.$store.state.account.session == null ||
        this.$store.state.account.session.WgKeyGenerated == null
      )
        return null;
      return new Date(this.$store.state.account.session.WgKeyGenerated);
    },
    formatDate: function (d) {
      if (d == null) return null;
      return dateDefaultFormat(d);
    },
    onShowHelpObfsproxy: function () {
      try {
        this.$refs.helpDialogObfsproxy.showModal();
      } catch (e) {
        console.error(e);
      }
    },

    onCloseAddCustomPortDlg: function () {
      try {
        this.$refs.addCustomPortDlg.close();
      } catch (e) {
        console.error(e);
      }
    },
  },
  computed: {
    isDisconnected: function () {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      );
    },
    isCanUseIPv6InTunnel: function () {
      return this.$store.getters["isCanUseIPv6InTunnel"];
    },
    enableIPv6InTunnel: {
      get() {
        return this.$store.state.settings.enableIPv6InTunnel;
      },
      async set(value) {
        if (value === this.$store.state.settings.enableIPv6InTunnel) return;

        this.$store.dispatch("settings/enableIPv6InTunnel", value);
        await this.reconnect();
      },
    },
    showGatewaysWithoutIPv6: {
      get() {
        return this.$store.state.settings.showGatewaysWithoutIPv6;
      },
      set(value) {
        this.$store.dispatch("settings/showGatewaysWithoutIPv6", value);
      },
    },
    IsAccountActive: function () {
      // if no info about account status - let's believe that account is active
      if (
        !this.$store.state.account ||
        !this.$store.state.account.accountStatus
      )
        return true;
      return this.$store.state.account?.accountStatus?.Active === true;
    },
    mtu: {
      get() {
        return this.$store.state.settings.mtu;
      },
      set(value) {
        this.$store.dispatch("settings/mtu", value);
      },
    },
    isMtuBadValue: function () {
      if (
        this.mtu != null &&
        this.mtu != "" &&
        this.mtu != 0 &&
        (this.mtu < 1280 || this.mtu > 65535)
      ) {
        return true;
      }
      return false;
    },
    userDefinedOvpnFile: function () {
      if (!this.$store.state.settings.daemonSettings) return null;
      return this.$store.state.settings.daemonSettings.UserDefinedOvpnFile;
    },
    wgKeyRegenerationInterval: {
      get() {
        return this.$store.state.account.session.WgKeysRegenIntervalSec;
      },
      set(value) {
        // daemon will send back a Hello response with updated 'session.WgKeysRegenIntervalSec'
        sender.WgSetKeysRotationInterval(value);
      },
    },

    //  -------- obfuscation BEGIN ------------------
    V2RayType: {
      get() {
        return this.$store.getters["settings/getV2RayConfig"];
      },
    },
    connectionUseObfsproxy: {
      get() {
        return this.$store.getters["settings/isConnectionUseObfsproxy"];
      },
    },

    obfuscationType: {
      get() {
        let obfsCfg = null;
        if (this.isOpenVPN === true)
          obfsCfg = this.$store.state.settings.openvpnObfsproxyConfig;

        let v2RayCfg = this.$store.getters["settings/getV2RayConfig"];
        if (!obfsCfg && !v2RayCfg) return makeObfsInfoUiObj();
        return makeObfsInfoUiObj(v2RayCfg, obfsCfg?.Version, obfsCfg?.Obfs4Iat);
      },
      set(value) {
        let v2RayCfg = V2RayObfuscationEnum.None;
        let obfsCfg = {
          Version: ObfsproxyVerEnum.None,
          Obfs4Iat: Obfs4IatEnum.IAT0,
        };

        // Set new obfuscation parameters
        // (do not chane obfsproxy parames from WireGuard settings)
        if (value.obfsVer != undefined && this.isOpenVPN === true) {
          obfsCfg = {
            Version: value.obfsVer,
            Obfs4Iat: value.obfs4Iat,
          };
        } else if (value.v2RayType != undefined) {
          v2RayCfg = value.v2RayType;
        }

        this.$store.dispatch("settings/openvpnObfsproxyConfig", obfsCfg);
        this.$store.dispatch("settings/setV2RayConfig", v2RayCfg);
      },
    },

    obfuscationTypes: {
      get() {
        let ret = [makeObfsInfoUiObj()];
        if (this.isOpenVPN === true) {
          var obfuscationTypes = [
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT0),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT1),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT2),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs3, Obfs4IatEnum.IAT0),
          ];
          ret = [...ret, ...obfuscationTypes];
        }
        let v2RayTypes = [
          makeObfsInfoUiObj(V2RayObfuscationEnum.QUIC),
          makeObfsInfoUiObj(V2RayObfuscationEnum.TCP),
        ];
        ret = [...ret, ...v2RayTypes];
        return ret;
      },
    },

    //  -------- obfuscation END ------------------

    ovpnProxyType: {
      get() {
        return this.$store.state.settings.ovpnProxyType;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyType", value);
      },
    },
    ovpnProxyServer: {
      get() {
        return this.$store.state.settings.ovpnProxyServer;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyServer", value);
      },
    },
    ovpnProxyPort: {
      get() {
        return this.$store.state.settings.ovpnProxyPort;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyPort", value);
      },
    },
    ovpnProxyUser: {
      get() {
        return this.$store.state.settings.ovpnProxyUser;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyUser", value);
      },
    },
    ovpnProxyPass: {
      get() {
        return this.$store.state.settings.ovpnProxyPass;
      },
      set(value) {
        this.$store.dispatch("settings/ovpnProxyPass", value);
      },
    },

    isOpenVPN: function () {
      return this.$store.state.settings.vpnType === VpnTypeEnum.OpenVPN;
    },
    wgKeysGeneratedDateStr: function () {
      return this.formatDate(this.getWgKeysGenerated());
    },
    wgKeysWillBeRegeneratedStr: function () {
      let t = this.getWgKeysGenerated();
      if (t == null) return null;

      t.setSeconds(
        t.getSeconds() +
          this.$store.state.account.session.WgKeysRegenIntervalSec,
      );

      let now = new Date();
      if (t < now) {
        // Do not show planned regeneration date in the past (it can happen after the computer wake up from a long sleep)
        // Show 'today' as planned date to regenerate keys in this case.
        // (the max interval to check if regeneration required is defined on daemon side, it is less than 24 hours)
        t = now;
      }

      return this.formatDate(t);
    },
    wgKeysExpirationDateStr: function () {
      let t = this.getWgKeysGenerated();
      if (t == null) return null;
      t.setSeconds(t.getSeconds() + 40 * 24 * 60 * 60); // 40 days
      return this.formatDate(t);
    },
    wgRegenerationIntervals: function () {
      let ret = [{ text: "1 day", seconds: 24 * 60 * 60 }];
      for (let i = 2; i <= 30; i++) {
        ret.push({ text: `${i} days`, seconds: i * 24 * 60 * 60 });
      }
      return ret;
    },
    wgQuantumResistanceStr: function () {
      if (this.$store.state.account.session.WgUsePresharedKey === true)
        return "Enabled";
      return "Disabled";
    },
    isShowAddPortOption: function () {
      if (
        this.$store.state.settings.isMultiHop === true ||
        this.$store.getters["settings/isConnectionUseObfsproxy"]
      )
        return false;

      const ranges = this.$store.getters["vpnState/portRanges"];
      if (!ranges || ranges.length <= 0) return false;

      return true;
    },
    port: {
      get() {
        return this.$store.getters["settings/getPort"];
      },
      set(value) {
        this.isPortModified = true;

        if (value == "valueAddCustomPort") {
          // we need it just to update UI to show current port (except 'Add custom port...')
          this.$store.dispatch("settings/setPort", this.port);

          try {
            this.$refs.addCustomPortDlg.showModal();
          } catch (e) {
            console.error(e);
          }
          return;
        }

        this.$store.dispatch("settings/setPort", value);
      },
    },

    // Return suitable ports for current connection type.
    // If Obfuscation is enabled - the ports can differ from the default ports:
    // - Obfsproxy uses only TCP ports
    // - V2Ray (TCP) uses only TCP ports
    // - V2Ray (QUIC) uses only UDP ports
    prefferedPorts: function () {
      let ret = [];
      let ports = this.$store.getters["vpnState/connectionPorts"];

      const isMH = this.$store.state.settings.isMultiHop;
      const isObfsproxy =
        this.$store.getters["settings/isConnectionUseObfsproxy"];

      const V2RayType = this.$store.getters["settings/getV2RayConfig"];

      const isV2Ray =
        V2RayType === V2RayObfuscationEnum.QUIC ||
        V2RayType === V2RayObfuscationEnum.TCP;

      if (!isV2Ray) {
        // Non-V2Ray
        if (isObfsproxy) {
          // For Obfsproxy: port number is ignored. Only TCP protocol is applicable.
          // So we keep only one port (of required type; port number will not be shown)
          // (try to use currently selected port number)
          let port = ports.find((p) => p.port === this.port.port);
          if (port) ports = [port];
          else if (ports.length > 0) ports = [ports[0]];
        } else if (isMH) {
          // For Multi-Hop: port number is ignored. Only protocol has sense.
          // So we just return one port definition for each protocol applicable for current VPN
          // (by default, using currently selected port if it can be applied)
          let portsByProtoHash = { udp: null, tcp: null };

          // try to use currently selected port
          let curPort = ports.find(
            (p) => p.port === this.port.port && p.type === this.port.type,
          );
          if (curPort) {
            if (curPort.type === PortTypeEnum.TCP)
              portsByProtoHash.tcp = curPort;
            else portsByProtoHash.udp = curPort;
          }
          // get first port definition for each protocol
          if (!portsByProtoHash.tcp)
            portsByProtoHash.tcp = ports.find(
              (p) => p.type === PortTypeEnum.TCP,
            );
          if (!portsByProtoHash.udp)
            portsByProtoHash.udp = ports.find(
              (p) => p.type === PortTypeEnum.UDP,
            );

          if (portsByProtoHash.tcp || portsByProtoHash.udp) {
            ports = [];
            if (portsByProtoHash.udp) ports.push(portsByProtoHash.udp);
            if (portsByProtoHash.tcp) ports.push(portsByProtoHash.tcp);
          }
        }
      }

      // create UI items for ports
      ports.forEach((p) =>
        ret.push({
          text:
            !isV2Ray && (isMH === true || isObfsproxy === true) // port number ignored for multi-hop and obfsproxy (but not for V2Ray!)
              ? `${enumValueName(PortTypeEnum, p.type)}`
              : `${enumValueName(PortTypeEnum, p.type)} ${p.port}`,
          key: `${enumValueName(PortTypeEnum, p.type)} ${p.port}`,
          port: p,
        }),
      );
      return ret;
    },
  },
};

function makeObfsInfoUiObj(v2rayType, obfsVer, obfs4Iat) {
  if (!v2rayType && !obfsVer) return { text: "Disabled" };

  // V2Ray
  if (v2rayType) {
    return {
      text: `V2Ray (VMESS/${enumValueName(V2RayObfuscationEnum, v2rayType)})`,
      v2RayType: v2rayType,
    };
  }

  // Obfsproxy
  let iatStr = "";
  if (obfs4Iat && obfs4Iat > 0)
    iatStr = ` (${enumValueName(Obfs4IatEnum, obfs4Iat)})`;
  return {
    text: `${enumValueName(ObfsproxyVerEnum, obfsVer)}${iatStr}`,
    obfsVer: obfsVer,
    obfs4Iat: obfs4Iat,
  };
}
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

div.disabled {
  pointer-events: none;
  opacity: 0.5;
}

input:disabled {
  opacity: 0.5;
}
input:disabled + label {
  opacity: 0.5;
}

div.paramBlock {
  @extend .flexRow;
  margin-top: 6px;
}

div.paramBlockText {
  margin-top: 6px;
  margin-right: 21px;
}

div.paramBlockDetailedConfig {
  @extend .flexRow;
  margin-top: 2px;
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
  user-select: text;
  letter-spacing: 0.1px;
}

div.defInputWidth {
  width: 100px;
  background: red;
}

div.paramName {
  min-width: 161px;
  max-width: 161px;
}

div.settingsRadioBtnProxy {
  @extend .settingsRadioBtn;
  padding-right: 20px;
}

select {
  border: 0.5px solid rgba(0, 0, 0, 0.2);
  border-radius: 3.5px;
  width: 170px;
}

.description {
  @extend .settingsGrayLongDescriptionFont;
  margin-left: 20px;
}

input.proxyParam {
  width: 100px;
}
</style>
