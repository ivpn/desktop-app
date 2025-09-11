<template>
  <div>
    <div class="settingsTitle" tabindex="0">DNS SETTINGS</div>

    <div>
      <div class="param" tabindex="0">
        <input type="checkbox" id="dnsIsCustom" v-model="dnsIsCustom" />
        <label class="defColor" for="dnsIsCustom"
          >Use custom DNS server when connected to IVPN</label
        >
      </div>

      <div v-bind:class="{ disabled: dnsIsCustom === false }">
        <div class="flexRow paramProps" tabindex="0">
          <div class="defColor paramName">IP address:</div>

          <input
            class="settingsTextInput"
            v-bind:class="{ badData: isIPError === true }"
            placeholder="0.0.0.0"
            v-model="dnsHost"
          />
        </div>

        <div v-if="CanUseDnsOverHttps || CanUseDnsOverTls">
          <div class="paramProps">
            <div class="flexRow paramBlock" tabindex="0">
              <div class="defColor paramName">DNS encryption:</div>
              <div class="settingsRadioBtnEx">
                <input
                  style="margin-left: 0px"
                  type="radio"
                  id="dnsEncryptionNone"
                  name="dnsEnc"
                  v-model="dnsEncryption"
                  value="None"
                />
                <label class="defColor" for="dnsEncryptionNone">None</label>
              </div>
              <div class="settingsRadioBtnEx" v-if="CanUseDnsOverHttps">
                <input
                  type="radio"
                  id="dnsEncryptionHttps"
                  name="dnsEnc"
                  v-model="dnsEncryption"
                  value="DoH"
                />
                <label class="defColor" for="dnsEncryptionHttps"
                  >DNS over HTTPS</label
                >
              </div>
              <div class="settingsRadioBtnEx" v-if="CanUseDnsOverTls">
                <input
                  type="radio"
                  id="dnsEncryptionTls"
                  name="dnsEnc"
                  v-model="dnsEncryption"
                  value="DoT"
                />
                <label class="defColor" for="dnsEncryptionTls"
                  >DNS over TLS</label
                >
              </div>
            </div>
          </div>

          <div
            class="flexRowAlignTop paramProps" tabindex="0"
            v-bind:class="{ disabled: dnsIsEncrypted === false }"
          >
            <div class="defColor paramName">
              {{ dnsEncryptionNameLabel }} URI template:
            </div>

            <div style="width: 100%">
              <input
                style="width: 100%; padding-right: 24px; margin-top: 0px"
                class="settingsTextInput"
                v-bind:class="{ badData: isTemplateURIError === true }"
                placeholder="https://..."
                v-model="dnsDohTemplate"
                spellcheck="false"
              />
              <div v-if="isShowDnsproxyDescription" class="fwDescription">
                DNS over HTTPS (DoH) is implemented using dnscrypt-proxy from
                the DNSCrypt project. Your DNS settings will be configured to
                send requests to dnscrypt-proxy listening on localhost
                (127.0.0.1).
              </div>
            </div>

            <!-- Predefined DoH/DoT configs -->
            <div
              v-bind:class="{ HiddenDiv: isHasPredefinedDohConfigs !== true }"
              style="margin-left: 5px"
            >
              <div>
                <img
                  style="
                    position: fixed;
                    width: 12px;
                    margin-left: 5px;
                    margin-top: 8px;
                  "
                  src="@/assets/arrow-bottom.svg"
                />
                <!-- Popup -->
                <select
                  title="Predefined DoH configurations"
                  @change="onPredefinedDohConfigSelected()"
                  v-model="thePredefinedDohConfigSelected"
                  style="cursor: pointer; width: 24px; height: 22px; opacity: 0"
                >
                  <option
                    v-for="m in predefinedDohConfigs"
                    v-bind:key="m.DohTemplate + m.DnsHost"
                    style="color: black; background-color: white"
                    v-bind:value="m"
                  >
                    {{ m.DnsHost }} ({{ m.DohTemplate }})
                  </option>
                </select>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="paramProps">
        <div class="fwDescription" tabindex="0">
          AntiTracker will override the custom DNS when enabled.
        </div>
      </div>
    </div>

    <div v-if="linuxIsShowResolvConfMgmtOption">
      <div class="param">
        <input
          type="checkbox"
          id="linuxDnsIsResolvConfMgmt"
          v-model="linuxDnsIsResolvConfMgmt"
        />
        <label class="defColor" for="linuxDnsIsResolvConfMgmt"
          >Force management of DNS using resolv.conf</label
        >
      </div>
      <div class="paramProps fwDescription">
        By default IVPN manages DNS resolvers using the 'systemd-resolved'
        daemon which is the correct method for systems based on Systemd. This
        option enables you to override this behavior and allow the IVPN app to
        directly modify the '/etc/resolv.conf' file.
      </div>
    </div>
  </div>
</template>

<script>
import { DnsEncryption, VpnStateEnum } from "@/store/types";
import { Platform, PlatformEnum } from "@/platform/platform";

const sender = window.ipcSender;

function checkIPv4List(ipListString) {
  if (!ipListString || ipListString.trim() === '') return false;
  // Split by comma and validate each IP
  const ips = ipListString.split(',').map(ip => ip.trim());
  const singleIPRegex = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)(\.(?!$)|$)){4}$/;
  return ips.every(ip => singleIPRegex.test(ip));
}

function checkIsDnsIPError(dnsIpString) {
  // IPv4 or IPv6
  //var expression = /((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))/;

  // IPv4
  //var expression = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)(\.(?!$)|$)){4}$/;
  //return !expression.test(dnsIpString.trim());

  return !checkIPv4List(dnsIpString);
}

function processError(e) {
  if (!e) return;
  console.error(e);
  sender.showMessageBoxSync({
    type: "error",
    buttons: ["OK"],
    message: e,
  });
}

export default {
  props: { registerBeforeCloseHandler: Function },
  created() {
    // We have to call applyChanges() even when Settings window was closed by user
    // (the 'beforeUnmount()' is not called in this case)
    window.addEventListener("beforeunload", this.applyChanges);

    if (this.registerBeforeCloseHandler != null) {
      // Register handler which will be called before closing current view
      // Handler MUST be 'async' function and MUST return 'true' to allow to switch current view
      this.registerBeforeCloseHandler(this.applyChanges);
    }
  },

  async beforeUnmount() {
    window.removeEventListener("beforeunload", this.applyChanges);
    await this.applyChanges();
  },

  data: function () {
    return {
      isEditingFinished: false,
      isDnsValueChanged: false,
      thePredefinedDohConfigSelected: null,
      val_linuxDnsIsResolvConfMgmt: false,

      // Since UI currently has single input for DNS IPs - we need to keep string with comma-separated IPs
      // This is temporary solution until UI will be updated to have multiple inputs for DNS IPs
      _dnsAddressesString: "",
      _encryption: DnsEncryption.None,
      _template: "",
    };
  },
  mounted() {
    this.updateLocalDnsSettings();
    this.requestPredefinedDohConfigs();
    this.val_linuxDnsIsResolvConfMgmt =
      this.daemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle;
  },
  methods: {
    checkIsDisconnectedAndWarn: function () {
      if (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      )
        return true;

      sender.showMessageBoxSync({
        type: "info",
        buttons: ["OK"],
        message: "You are now connected to IVPN",
        detail: "You can change this settings only when IVPN is disconnected.",
      });

      return false;
    },

    // Validate and APPLY changes
    async applyChanges(e) {
      this.isEditingFinished = true;
      // when component closing ->  update changed DNS (if necessary)

      if (this.dnsIsCustom && (this.isTemplateURIError || this.isIPError)) {
        sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Error in DNS configuration.",
          detail: `Custom DNS will not be applied.`,
        });

        if (e && typeof e.preventDefault === "function") {
          // it is 'beforeunload' handler. Prevent closing window.
          e.preventDefault();
          e.returnValue = "";
        }
        return false;
      }

      if (this.isDnsValueChanged !== true) return true;
      try {
        // convert value to comma-separated list of IPs
        let ipList = this._dnsAddressesString.split(',').map(ip => ip.trim()).filter(ip => ip !== '');            
        // update newDnsCfg.Servers for all provided IPs
        let newDnsCfg = Object.assign({}, this.$store.state.settings.dnsCustomCfg);
        newDnsCfg.Servers = ipList.map(ip => ({
          Address: ip,
          Encryption: this.dnsEncryptionValue || DnsEncryption.None,
          Template: this.dnsDohTemplate || ""
        }));

        console.debug('Applying DNS :', this._dnsAddressesString, " => ", ipList, " => ", newDnsCfg);
        this.$store.dispatch("settings/dnsCustomCfg", newDnsCfg);

        // Apply changes
        await sender.SetDNS();
        this.isDnsValueChanged = false;
      } catch (e) {
        processError(e);
        // it is 'beforeunload' handler. Prevent closing window.
        e.preventDefault();
        e.returnValue = "";
      }
      return true;
    },

    onPredefinedDohConfigSelected() {
      const newVal = this.thePredefinedDohConfigSelected;
      if (newVal && newVal.DnsHost && newVal.DohTemplate) {
        this.isDnsValueChanged = true;
        
        this._dnsAddressesString = newVal.DnsHost;
        this._encryption = DnsEncryption.DnsOverHttps;
        this._template = newVal.DohTemplate;
      }
    },
    requestPredefinedDohConfigs() {
      if (!this.CanUseDnsOverHttps && !this.CanUseDnsOverTls) return;
      if (this.$store.state.dnsPredefinedConfigurations) return; // configurations already initialized - no sense to request them again
      setTimeout(() => {
        sender.RequestDnsPredefinedConfigs();
      }, 0);
    },

    updateLocalDnsSettings() {
      const dnsServers = this.dnsCustomCfg.Servers || [];
       // Since UI currently has single input for DNS IPs - we need to keep string with comma-separated IPs
      // This is temporary solution until UI will be updated to have multiple inputs for DNS IPs
      this._dnsAddressesString = dnsServers.map(server => server.Address).join(', ');
       // Take into account only first server (have to be changed in future when full support of multiple servers will be implemented)
      this._encryption = this.$store.state.settings.dnsCustomCfg?.Servers?.[0]?.Encryption || DnsEncryption.None;
      this._template = this.$store.state.settings.dnsCustomCfg?.Servers?.[0]?.Template || "";
    },
  },
  watch: {
    dnsIsEncrypted() {
      this.requestPredefinedDohConfigs();
    },
    daemonSettings() {
      this.val_linuxDnsIsResolvConfMgmt =
        this.daemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle;
    },
    dnsCustomCfg() {
      this.updateLocalDnsSettings();
    },
  },

  computed: {
    // needed for 'watch'
    daemonSettings() {
      return this.$store.state.settings.daemonSettings;
    },

    dnsCustomCfg() {
      return this.$store.state.settings.dnsCustomCfg;
    },

    isShowDnsproxyDescription() {
      return Platform() !== PlatformEnum.Windows;
    },
    CanUseDnsOverTls: {
      get() {
        return this.$store.state.dnsAbilities.CanUseDnsOverTls === true;
      },
    },
    CanUseDnsOverHttps: {
      get() {
        return this.$store.state.dnsAbilities.CanUseDnsOverHttps === true;
      },
    },

    dnsIsCustom: {
      get() {
        return this.$store.state.settings.dnsIsCustom;
      },
      async set(value) {
        this.isDnsValueChanged = true;
        this.$store.dispatch("settings/dnsIsCustom", value);
      },
    },

    linuxIsShowResolvConfMgmtOption() {
      try {
        if (Platform() !== PlatformEnum.Linux) return false;

        const disabledFuncs = this.$store.state.disabledFunctions;
        if (
          disabledFuncs.Platform.Linux.DnsMgmtOldResolvconfError != "" ||
          disabledFuncs.Platform.Linux.DnsMgmtNewResolvectlError != ""
        )
          return false;

        const dSettings = this.$store.state.settings.daemonSettings;
        if (dSettings?.UserPrefs?.Linux?.IsDnsMgmtOldStyle == null)
          return false;

        return true;
      } catch (e) {
        console.error(e);
        return false;
      }
    },

    linuxDnsIsResolvConfMgmt: {
      get() {
        return this.val_linuxDnsIsResolvConfMgmt;
      },
      async set(value) {
        const clone = function (obj) {
          return JSON.parse(JSON.stringify(obj));
        };

        try {
          // We need to erase value in order to the check-box be updated correctly according to confirmation response from daemon
          // The value will be updated in "watch: daemonSettings()"
          this.val_linuxDnsIsResolvConfMgmt = null;

          if (!this.checkIsDisconnectedAndWarn()) {
            return;
          }

          let prefs = clone(
            this.$store.state.settings.daemonSettings.UserPrefs,
          );
          if (prefs.Linux.IsDnsMgmtOldStyle != value) {
            prefs.Linux.IsDnsMgmtOldStyle = value;
            await sender.SetUserPrefs(prefs);
          }
        } catch (e) {
          processError(e);
        } finally {
          setTimeout(() => {
            this.val_linuxDnsIsResolvConfMgmt =
              this.daemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle;
          }, 0);
        }
      },
    },

    dnsIsEncrypted: {
      get() {
        return this._encryption  !==  DnsEncryption.None;
      },
    },

    dnsEncryptionNameLabel: {
      get() {
        if (this.dnsEncryption === DnsEncryption.DnsOverTls) return "DoT";
        return "DoH";
      },
    },

    dnsHost: {
      get() {
        return this._dnsAddressesString; 
      },
      set(value) {
        this.isDnsValueChanged = true;
        this._dnsAddressesString = value;
      },
    },
    dnsEncryptionValue: {
      get() {
        return this._encryption;
      },
    },
    dnsEncryption: {
      get() {
        switch (this.dnsEncryptionValue) {
          case DnsEncryption.DnsOverTls:
            return "DoT";
          case DnsEncryption.DnsOverHttps:
            return "DoH";
          default:
            return "None";
        }
      },
      set(value) {
        let enc = DnsEncryption.None;
        switch (value) {
          case "DoT":
            enc = DnsEncryption.DnsOverTls;
            break;
          case "DoH":
            enc = DnsEncryption.DnsOverHttps;
            break;
          default:
            enc = DnsEncryption.None;
            this._template = "";
        }
        this.isDnsValueChanged = true;
        this._encryption = enc;
      },
    },

    dnsDohTemplate: {
      get() {
        return this._template;
      },
      set(value) {
        this.isDnsValueChanged = true;
        this._encryption = DnsEncryption.DnsOverHttps;
        this._template = value;
      },
    },

    isHasPredefinedDohConfigs: {
      get() {
        if (!this.CanUseDnsOverHttps && !this.CanUseDnsOverTls) return false;

        // Next group of check is more for 'nice UI'
        // We show "paste" image even when selected not-encrypted DNS
        let cfgs = this.$store.state.dnsPredefinedConfigurations;
        if (!cfgs) return false;

        if (!this.dnsIsEncrypted && cfgs.length > 0) return true;

        // check if there are any predefined configuration available (for current encryption)
        return this.predefinedDohConfigs && this.predefinedDohConfigs.length > 0;        
      },
    },
    predefinedDohConfigs: {
      get() {
        if (!this.dnsIsEncrypted) return null;
        let cfgs = this.$store.state.dnsPredefinedConfigurations;
        if (!cfgs) return null;

        const expectedEnc = this._encryption;
        let filtered = cfgs.filter(
          (cfg) =>
            cfg.Encryption === expectedEnc &&
            cfg.DnsHost &&
            cfg.DohTemplate &&
            !checkIsDnsIPError(cfg.DnsHost),
        );
        return filtered;
      },
    },

    isTemplateURIError: function () {
      if (this.isEditingFinished !== true) return false;
      if (!this.dnsIsCustom) return false;
      if (this.dnsIsEncrypted !== true) return false;
      try {
        new URL(this.dnsDohTemplate);
      } catch (_) {
        return true;
      }
      return !this.dnsDohTemplate.toLowerCase().startsWith("https://");
    },
    isIPError: function () {
      if (this.isEditingFinished !== true) return false;
      if (!this.dnsHost) {
        if (this.dnsIsCustom) return true;
        return false;
      }
      return checkIsDnsIPError(this.dnsHost);
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}

div.paramProps {
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
}
div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 8px;
  max-width: 425px;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

div.paramName {
  min-width: 120px;
  max-width: 120px;
}

label {
  margin-left: 1px;
  font-weight: 500;
}

input.badData {
  border-color: red;
}

input:disabled {
  opacity: 0.5;
}

div.disabled {
  pointer-events: none;
  opacity: 0.5;
}

div.settingsRadioBtnEx {
  @extend .settingsRadioBtn;
  padding-right: 20px;
}

div.HiddenDiv {
  opacity: 0;
}
div.HiddenDiv > * {
  opacity: 0;
  pointer-events: none;
  cursor: default;
}
</style>
