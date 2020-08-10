<template>
  <div>
    <div class="settingsTitle">ANTITRACKER</div>

    <div class="defColor" style="margin-bottom: 24px;">
      IVPN blocks ads, malicious websites, and third-party trackers using our
      private DNS servers.
      <button class="link" v-on:click="onLearnMoreLink">
        Learn more
      </button>
      about how IVPN AntiTracker is implemented.
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="isAntitrackerHardcore"
        v-model="isAntitrackerHardcore"
      />
      <label class="defColor" for="isAntitrackerHardcore">Hardcore Mode</label>
    </div>
    <div class="fwDescription">
      Hardcode mode blocks the leading companies with business models relying on
      user surveilance (currently: Google and Facebook)
    </div>
    <div class="fwDescription">
      To better understand the impact please see our
      <button class="link" v-on:click="onHardcodeLink">
        hardcore mode</button
      >.
    </div>
  </div>
</template>

<script>
const { shell } = require("electron");
import sender from "@/ipc/renderer-sender";

export default {
  data: function() {
    return {};
  },
  methods: {
    onLearnMoreLink: () => {
      shell.openExternal(`https://www.ivpn.net/antitracker`);
    },
    onHardcodeLink: () => {
      shell.openExternal(`https://www.ivpn.net/antitracker/hardcore`);
    }
  },
  computed: {
    isAntitrackerHardcore: {
      get() {
        return this.$store.state.settings.isAntitrackerHardcore;
      },
      async set(value) {
        this.$store.dispatch("settings/isAntitrackerHardcore", value);
        await sender.SetDNS();
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
