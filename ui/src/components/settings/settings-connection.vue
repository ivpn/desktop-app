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
        v-bind:class="{ disabled: connectionUseObfsproxy || !isDisconnected }"
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
              <select v-model="obfsproxyType">
                <option
                  v-for="item in obfsproxyTypes"
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

        <ComponentDialog ref="helpDialogObfsproxy" header="Info">
          <div>
            <p>
              <b>Obfsproxy</b> attempts to circumvent censorship, by
              transforming the traffic between the client and the server.
            </p>
            <p>
              The <i>obfs4</i> protocol is less likely to be blocked than
              <i>obfs3</i>.
            </p>
            <p>
              Inter-Arrival Time (<b>IAT</b>) parameter is applicable for
              <i>obfs4</i>:
            </p>

            <ul>
              <li>
                When IAT-mode is disabled large packets will be split by the
                network drivers which may result in network fingerprints that
                could be detected by censors.
              </li>
              <li>
                IAT1 - Large packets will be split into MTU-size packets by
                Obfsproxy (instead the network drivers), resulting in smaller
                packets that are more resistant to being reassembled for
                analysis and censoring.
              </li>
              <li>
                IAT2 - (paranoid mode) - Large packets will be split into
                variable size packets by Obfsproxy.
              </li>
            </ul>

            <p>
              <b> Note! </b> Enabling IAT mode will affect overall VPN speed and
              CPU load.
            </p>
          </div>
        </ComponentDialog>

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
            <select v-model="obfsproxyType">
              <option
                v-for="item in obfsproxyTypes"
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
              title="Valid range [1280 - 65535]. Please note that changing this value make affect the proper functioning of the VPN tunnel."
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

    //  -------- obfsproxy BEGIN ------------------
    connectionUseObfsproxy: {
      get() {
        return this.$store.getters["settings/isConnectionUseObfsproxy"];
      },
    },

    obfsproxyType: {
      get() {
        let obfsCfg = null;
        if (this.isOpenVPN === true)
          obfsCfg = this.$store.state.settings.daemonSettings.ObfsproxyConfig;

        let v2RayCfg = this.$store.state.settings.daemonSettings.V2RayConfig;
        if (!obfsCfg && !v2RayCfg) return makeObfsInfoUiObj();
        return makeObfsInfoUiObj(v2RayCfg, obfsCfg?.Version, obfsCfg?.Obfs4Iat);
      },
      set(value) {
        // erase obfuscation parameters
        if (value.obfsVer == undefined && this.isOpenVPN === true)
          sender.SetObfsproxy(ObfsproxyVerEnum.None, Obfs4IatEnum.IAT0); // do not chane obfsproxy parames from WireGuard settings
        if (value.v2RayType == undefined)
          sender.SetV2RayProxy(V2RayObfuscationEnum.None);

        // set new obfuscation parameters
        if (value.obfsVer != undefined && this.isOpenVPN === true) {
          sender.SetObfsproxy(value.obfsVer, value.obfs4Iat); // do not chane obfsproxy parames from WireGuard settings
        } else if (value.v2RayType != undefined)
          sender.SetV2RayProxy(value.v2RayType);
      },
    },

    obfsproxyTypes: {
      get() {
        let ret = [makeObfsInfoUiObj()];
        if (this.isOpenVPN === true) {
          var obfsproxyTypes = [
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT0),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT1),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs4, Obfs4IatEnum.IAT2),
            makeObfsInfoUiObj(null, ObfsproxyVerEnum.obfs3, Obfs4IatEnum.IAT0),
          ];
          ret = [...ret, ...obfsproxyTypes];
        }
        let v2RayTypes = [
          makeObfsInfoUiObj(V2RayObfuscationEnum.QUIC),
          makeObfsInfoUiObj(V2RayObfuscationEnum.TCP),
        ];
        ret = [...ret, ...v2RayTypes];
        return ret;
      },
    },

    //  -------- obfsproxy END ------------------

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
          this.$store.state.account.session.WgKeysRegenIntervalSec
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
    prefferedPorts: function () {
      let ret = [];
      let ports = this.$store.getters["vpnState/connectionPorts"];

      const isMH = this.$store.state.settings.isMultiHop;
      const isObfsproxy =
        this.$store.getters["settings/isConnectionUseObfsproxy"];

      if (isObfsproxy) {
        // For Obfsproxy: port number is ignored. Only TCP protocol is applicable.
        // try to use currently selected port
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
          (p) => p.port === this.port.port && p.type === this.port.type
        );
        if (curPort) {
          if (curPort.type === PortTypeEnum.TCP) portsByProtoHash.tcp = curPort;
          else portsByProtoHash.udp = curPort;
        }
        // get first port definition for each protocol
        if (!portsByProtoHash.tcp)
          portsByProtoHash.tcp = ports.find((p) => p.type === PortTypeEnum.TCP);
        if (!portsByProtoHash.udp)
          portsByProtoHash.udp = ports.find((p) => p.type === PortTypeEnum.UDP);

        if (portsByProtoHash.tcp || portsByProtoHash.udp) {
          ports = [];
          if (portsByProtoHash.udp) ports.push(portsByProtoHash.udp);
          if (portsByProtoHash.tcp) ports.push(portsByProtoHash.tcp);
        }
      }

      ports.forEach((p) =>
        ret.push({
          text:
            isMH === true || isObfsproxy === true // port number ignored for multi-hop and obfsproxy
              ? `${enumValueName(PortTypeEnum, p.type)}`
              : `${enumValueName(PortTypeEnum, p.type)} ${p.port}`,
          key: `${enumValueName(PortTypeEnum, p.type)} ${p.port}`,
          port: p,
        })
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
      text: `V2Ray (${enumValueName(V2RayObfuscationEnum, v2rayType)})`,
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
