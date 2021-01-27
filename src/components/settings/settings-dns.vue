<template>
  <div>
    <div class="settingsTitle">DNS</div>

    <div class="param">
      <input type="checkbox" id="dnsIsCustom" v-model="dnsIsCustom" />
      <label class="defColor" for="dnsIsCustom"
        >Use custom DNS server when connected to IVPN</label
      >
    </div>
    <div class="paramProps">
      <div class="defColor">IP address:</div>

      <input
        class="settingsTextInput"
        placeholder="0.0.0.0"
        v-model="dnsCustom"
        :disabled="dnsIsCustom === false"
      />

      <div class="fwDescription">
        AntiTracker will override the custom DNS when enabled.
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  async beforeDestroy() {
    // when component closing ->  update changed DNS (if necessary)
    if (this.isDnsValueChanged) await sender.SetDNS();
    this.isDnsValueChanged = false;
  },

  data: function() {
    return {
      isDnsValueChanged: false
    };
  },
  methods: {},
  computed: {
    dnsIsCustom: {
      get() {
        return this.$store.state.settings.dnsIsCustom;
      },
      async set(value) {
        this.$store.dispatch("settings/dnsIsCustom", value);
        await sender.SetDNS();
      }
    },
    dnsCustom: {
      get() {
        return this.$store.state.settings.dnsCustom;
      },
      set(value) {
        this.isDnsValueChanged = true;
        this.$store.dispatch("settings/dnsCustom", value);
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

label {
  margin-left: 1px;
  font-weight: 500;
}

input:disabled {
  opacity: 0.5;
}
</style>
