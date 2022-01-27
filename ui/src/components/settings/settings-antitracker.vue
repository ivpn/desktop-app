<template>
  <div>
    <div class="settingsTitle">ANTITRACKER SETTINGS</div>

    <div class="defColor" style="margin-bottom: 24px">
      When AntiTracker is enabled, IVPN blocks ads, malicious websites, and
      third-party trackers using our private DNS servers.
      <button class="link" @click="onLearnMoreLink">Learn more</button>
      about how IVPN AntiTracker is implemented.
    </div>

    <div class="param">
      <input
        id="isAntitrackerHardcore"
        v-model="isAntitrackerHardcore"
        type="checkbox"
      />
      <label class="defColor" for="isAntitrackerHardcore">Hardcore Mode</label>
    </div>
    <div class="fwDescription">
      Hardcode mode blocks the leading companies with business models relying on
      user surveillance (currently: Google and Facebook)
    </div>
    <div class="fwDescription">
      To better understand how this may impact your experience please refer to
      our
      <button class="link" @click="onHardcodeLink">hardcore mode FAQ</button>.
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  data: function () {
    return {};
  },
  computed: {
    isAntitrackerHardcore: {
      get() {
        return this.$store.state.settings.isAntitrackerHardcore;
      },
      async set(value) {
        this.$store.dispatch("settings/isAntitrackerHardcore", value);
        await sender.SetDNS();
      },
    },
  },
  methods: {
    onLearnMoreLink: () => {
      sender.shellOpenExternal(`https://www.ivpn.net/antitracker`);
    },
    onHardcodeLink: () => {
      sender.shellOpenExternal(`https://www.ivpn.net/antitracker/hardcore`);
    },
  },
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

button.link {
  @extend .noBordersTextBtn;
  @extend .settingsLinkText;
  font-size: inherit;
}
label {
  margin-left: 1px;
  font-weight: 500;
}
</style>
