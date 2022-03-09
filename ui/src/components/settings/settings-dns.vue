<template>
  <div>
    <div class="settingsTitle">DNS SETTINGS</div>

    <div class="param">
      <input type="checkbox" id="dnsIsCustom" v-model="dnsIsCustom" />
      <label class="defColor" for="dnsIsCustom"
        >Use custom DNS server when connected to IVPN</label
      >
    </div>

    <div v-bind:class="{ disabled: dnsIsCustom === false }">
      <div class="flexRow paramProps">
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
          <div class="flexRow paramBlock">
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
          class="flexRow paramProps"
          v-bind:class="{ disabled: dnsIsEncrypted === false }"
        >
          <div class="defColor paramName">
            {{ dnsEncryptionNameLabel }} URI template:
          </div>

          <div style="width: 100%">
            <input
              style="width: 100%; padding-right: 24px"
              class="settingsTextInput"
              v-bind:class="{ badData: isTemplateURIError === true }"
              placeholder="https://..."
              v-model="dnsDohTemplate"
            />
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
                  margin-top: 10px;
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
      <div class="fwDescription">
        AntiTracker will override the custom DNS when enabled.
      </div>
    </div>
  </div>
</template>

<script>
import { DnsEncryption } from "@/store/types";

const sender = window.ipcSender;

function checkIsDnsIPError(dnsIpString) {
  // IPv4 or IPv6
  //var expression = /((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))/;

  // IPv4
  var expression = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)(\.(?!$)|$)){4}$/;
  return !expression.test(dnsIpString);
}

export default {
  props: { registerBeforeCloseHandler: Function },
  created() {
    // We have to call applyChanges() even when Settings window was closed by user
    // (the 'beforeDestroy()' is not called in this case)
    window.addEventListener("beforeunload", this.applyChanges);

    if (this.registerBeforeCloseHandler != null) {
      // Register handler which will be called before closing current view
      // Handler MUST be 'async' function and MUST return 'true' to allow to switch current view
      this.registerBeforeCloseHandler(this.applyChanges);
    }
  },

  async beforeDestroy() {
    window.removeEventListener("beforeunload", this.applyChanges);
    await this.applyChanges();
  },

  data: function () {
    return {
      isDnsValueChanged: false,
      thePredefinedDohConfigSelected: null,
    };
  },
  mounted() {
    this.requestPredefinedDohConfigs();
  },
  methods: {
    async applyChanges(e) {
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
      this.isDnsValueChanged = false;
      await sender.SetDNS();
      return true;
    },

    onPredefinedDohConfigSelected() {
      const newVal = this.thePredefinedDohConfigSelected;
      if (newVal && newVal.DnsHost && newVal.DohTemplate) {
        this.isDnsValueChanged = true;
        let newDnsCfg = Object.assign(
          {},
          this.$store.state.settings.dnsCustomCfg
        );
        newDnsCfg.DnsHost = newVal.DnsHost;
        newDnsCfg.DohTemplate = newVal.DohTemplate;
        this.$store.dispatch("settings/dnsCustomCfg", newDnsCfg);
      }
    },
    requestPredefinedDohConfigs() {
      if (!this.CanUseDnsOverHttps && !this.CanUseDnsOverTls) return;
      if (this.$store.state.dnsPredefinedConfigurations) return; // configurations already initialized - no sense to request them again
      setTimeout(() => {
        sender.RequestDnsPredefinedConfigs();
      }, 0);
    },
  },
  watch: {
    dnsIsEncrypted() {
      this.requestPredefinedDohConfigs();
    },
  },

  computed: {
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

    dnsIsEncrypted: {
      get() {
        return (
          this.$store.state.settings.dnsCustomCfg.Encryption !==
          DnsEncryption.None
        );
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
        return this.$store.state.settings.dnsCustomCfg.DnsHost;
      },
      set(value) {
        this.isDnsValueChanged = true;
        let newDnsCfg = Object.assign(
          {},
          this.$store.state.settings.dnsCustomCfg
        );
        newDnsCfg.DnsHost = value;

        if (
          this.$store.state.settings.dnsCustomCfg.Encryption ===
          DnsEncryption.None
        ) {
          newDnsCfg.DohTemplate = "";
        }

        this.$store.dispatch("settings/dnsCustomCfg", newDnsCfg);
      },
    },

    dnsEncryption: {
      get() {
        switch (this.$store.state.settings.dnsCustomCfg.Encryption) {
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
        }
        this.isDnsValueChanged = true;
        let newDnsCfg = Object.assign(
          {},
          this.$store.state.settings.dnsCustomCfg
        );
        newDnsCfg.Encryption = enc;
        this.$store.dispatch("settings/dnsCustomCfg", newDnsCfg);
      },
    },

    dnsDohTemplate: {
      get() {
        return this.$store.state.settings.dnsCustomCfg.DohTemplate;
      },
      set(value) {
        this.isDnsValueChanged = true;
        let newDnsCfg = Object.assign(
          {},
          this.$store.state.settings.dnsCustomCfg
        );
        newDnsCfg.DohTemplate = value;
        this.$store.dispatch("settings/dnsCustomCfg", newDnsCfg);
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
        return this.predefinedDohConfigs && this.predefinedDohConfigs.length > 0
          ? true
          : false;
      },
    },
    predefinedDohConfigs: {
      get() {
        if (!this.dnsIsEncrypted) return null;
        let cfgs = this.$store.state.dnsPredefinedConfigurations;
        if (!cfgs) return null;

        const expectedEnc = this.$store.state.settings.dnsCustomCfg.Encryption;
        let filtered = cfgs.filter(
          (cfg) =>
            cfg.Encryption === expectedEnc &&
            cfg.DnsHost &&
            cfg.DohTemplate &&
            !checkIsDnsIPError(cfg.DnsHost)
        );
        return filtered;
      },
    },

    isTemplateURIError: function () {
      if (this.dnsIsEncrypted !== true) return false;
      try {
        new URL(this.dnsDohTemplate);
      } catch (_) {
        return true;
      }
      return !this.dnsDohTemplate.toLowerCase().startsWith("https://");
    },
    isIPError: function () {
      if (!this.dnsHost) return false;
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
