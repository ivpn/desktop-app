<template>
  <div class="main">
    <img
      class="flag"
      v-if="isHideFlag == null"
      v-bind:class="{ flagBigger: isFlagBigger != null }"
      :src="serverImage"
    />
    <div
      class="text"
      v-if="isHideName == null"
      v-bind:class="{ text_large: size === 'large' }"
    >
      {{ serverName }}
    </div>

    <img
      :src="selectedImg"
      style="margin-left:10px"
      v-if="isShowSelected === true"
    />

    <div
      class="flexRow"
      v-bind:class="{ marginLeft: isHideFlag == null || isHideName == null }"
    >
      <img
        :src="pingStatusImg"
        v-if="isShowPingPicture == 'true' || isShowPingPicture == true"
      />

      <div
        class="pingtext marginLeft"
        v-if="isShowPingTime != null && pingStatusImg != null"
      >
        {{ server.ping }}ms
      </div>
    </div>
  </div>
</template>

<script>
import { PingQuality } from "@/store/types";
export default {
  // possible values of "size" : 'normal' (default), 'large'
  // possible values of "isFullName" : 'false\null' (default), 'true'
  props: [
    "server",
    "size",
    "isFullName",
    "isShowPingPicture",
    "isShowPingTime",
    "isShowSelected",
    "isHideName",
    "isHideFlag",
    "isFastestServer",
    "isRandomServer",
    "isFlagBigger",
    "manualName",
    "manualImage"
  ],
  computed: {
    serverName: function() {
      if (this.isFastestServer === true) return "Fastest server";
      if (this.isRandomServer === true) return "Random server";
      if (this.manualName != null) return this.manualName;
      if (this.server == null) return "";
      if (this.server.city == "" && this.server.country == "") return "";
      if (this.server.city == "") return this.server.country;
      if (this.isFullName === "true")
        return `${this.server.city}, ${this.server.country}`;
      return `${this.server.city}, ${this.server.country_code}`;
    },
    serverImage: function() {
      if (this.isFastestServer === true)
        return require("@/assets/speedometer.svg");
      if (this.isRandomServer === true) return require("@/assets/shuffle.svg");
      if (this.manualImage != null) return this.manualImage;
      if (this.server == null) return require(`@/assets/flags/unk.svg`);
      try {
        const ccode = this.server.country_code.toLowerCase();
        return require(`@/assets/flags/${ccode}.svg`);
      } catch (e) {
        console.log(e);
        return null; //return require(`@/assets/flags/unk.svg`);
      }
    },
    selectedImg: function() {
      return require("@/assets/check.svg");
    },
    pingStatusImg: function() {
      if (this.server == null) return null;
      switch (this.server.pingQuality) {
        case PingQuality.Good:
          return require("@/assets/iconStatusGood.svg");
        case PingQuality.Moderate:
          return require("@/assets/iconStatusModerate.svg");
        case PingQuality.Bad:
          return require("@/assets/iconStatusBad.svg");
      }
      return null;
    }
  },

  methods: {}
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
.main {
  display: flex;
  align-items: center;
}

.flagBigger {
  width: 26px;
}

.text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;

  font-size: 14px;
  line-height: 20px;
  margin-left: 16px;
}

.text_large {
  font-size: 18px;
  line-height: 21px;
  margin-left: 10px;
}

.flexRow {
  display: flex;
  align-items: center;
}

.marginLeft {
  margin-left: 9px;
}

.pingtext {
  color: var(--text-color-details);
}
</style>
