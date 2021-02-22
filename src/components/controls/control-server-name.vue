<template>
  <div class="main">
    <img class="flag" :src="serverImage" />
    <div class="text" v-bind:class="{ text_large: isLargeText }">
      {{ serverName }}
    </div>

    <img
      :src="selectedImg"
      style="margin-left:10px"
      v-if="isShowSelected === true"
    />
  </div>
</template>

<script>
import { PingQuality } from "@/store/types";

import Image_speedometer from "@/assets/speedometer.svg";
import Image_shuffle from "@/assets/shuffle.svg";
import Image_check from "@/assets/check.svg";
import Image_iconStatusGood from "@/assets/iconStatusGood.svg";
import Image_iconStatusModerate from "@/assets/iconStatusModerate.svg";
import Image_iconStatusBad from "@/assets/iconStatusBad.svg";

export default {
  props: {
    server: Object,
    isLargeText: Boolean,
    isFullName: String,
    isShowSelected: Boolean,
    isFastestServer: Boolean,
    isRandomServer: Boolean
  },

  computed: {
    serverName: function() {
      if (this.isFastestServer === true) return "Fastest server";
      if (this.isRandomServer === true) return "Random server";
      if (!this.server) return "";
      if (!this.server.city && !this.server.country) return "";
      if (this.server.city == "") return this.server.country;
      if (this.isFullName === "true")
        return `${this.server.city}, ${this.server.country}`;
      return `${this.server.city}, ${this.server.country_code}`;
    },
    serverImage: function() {
      if (this.isFastestServer === true) return Image_speedometer;
      if (this.isRandomServer === true) return Image_shuffle;
      if (!this.server) return `/flags/unk.svg`;
      try {
        const ccode = this.server.country_code.toLowerCase();
        return `/flags/${ccode}.svg`;
      } catch (e) {
        console.log(e);
        return null;
      }
    },
    selectedImg: function() {
      return Image_check;
    },
    pingStatusImg: function() {
      if (!this.server) return null;
      switch (this.server.pingQuality) {
        case PingQuality.Good:
          return Image_iconStatusGood;
        case PingQuality.Moderate:
          return Image_iconStatusModerate;
        case PingQuality.Bad:
          return Image_iconStatusBad;
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
