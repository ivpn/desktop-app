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

          <input
            style="width: 100%"
            class="settingsTextInput"
            placeholder="https://..."
            v-model="dnsDohTemplate"
          />
          <!-- Predefined DoH/DoT configs -->
          <div v-if="isHasPredefinedDohConfigs">
            <div>
              <button
                style="position: fixed"
                class="noBordersBtn"
                title="Predefined DoH configurations"
              >
                <img style="width: 18px" src="@/assets/clipboard.svg" />
              </button>
              <!-- Popup -->
              <select
                title="Predefined DoH configurations"
                @change="onPredefinedDohConfigSelected()"
                v-model="thePredefinedDohConfigSelected"
                style="cursor: pointer; width: 24px; height: 22px; opacity: 0"
              >
                <option
                  v-for="m in predefinedDohConfigs"
                  v-bind:key="m.DohTemplate"
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

export default {
  async beforeDestroy() {
    // when component closing ->  update changed DNS (if necessary)
    if (this.isDnsValueChanged) await sender.SetDNS();
    this.isDnsValueChanged = false;
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
      if (!this.dnsIsEncrypted) return;
      let cfgs = this.$store.state.dnsPredefinedConfigurations;
      if (cfgs) return null; // configurations already initialized - no sense to request them again
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
            cfg.Encryption === expectedEnc && cfg.DnsHost && cfg.DohTemplate
        );
        return filtered;
      },
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
</style>
