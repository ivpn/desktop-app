<template>
  <div class="main">
    <img class="flag" v-if="isHideFlag == null" :src="serverImage" />
    <div
      class="text"
      v-if="isHideName == null"
      v-bind:class="{ text_large: size === 'large' }"
    >
      {{ serverName }}
    </div>

    <div
      class="flexRow"
      v-bind:class="{ marginLeft: isHideFlag == null || isHideName == null }"
    >
      <img :src="pingStatusImg" v-if="isShowPingPicture != null" />

      <div class="pingtext marginLeft" v-if="isShowPingTime != null">
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
    "isHideName",
    "isHideFlag"
  ],
  computed: {
    serverName: function() {
      if (this.server == null) return "";
      if (this.isFullName === "true")
        return `${this.server.city}, ${this.server.country}`;
      return `${this.server.city}, ${this.server.country_code}`;
    },
    serverImage: function() {
      if (this.server == null) return null;
      try {
        return require(`@/assets/flags/24/${this.server.country_code.toLowerCase()}.png`);
      } catch (e) {
        console.log(e);
        return require(`@/assets/flags/24/_no_flag.png`);
      }
    },
    pingStatusImg: function() {
      if (this.server == null) return null;
      switch (this.server.pingQuality) {
        case PingQuality.Good:
          return require("@/assets/iconStatusGood.png");
        case PingQuality.Moderate:
          return require("@/assets/iconStatusModerate.png");
        case PingQuality.Bad:
          return require("@/assets/iconStatusBad.png");
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

.text {
  white-space: nowrap;
  overflow: hidden;

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
  color: $base-text-color-details;
}
</style>
