<template>
  <div class="main flexRow">
    <img :src="pingStatusImg" />

    <div class="pingtext marginLeft" v-if="isShowPingTime && server.ping > 0">
      {{ server.ping }}ms
    </div>
  </div>
</template>

<script>
import { PingQuality } from "@/store/types";

import Image_iconStatusGood from "@/assets/iconStatusGood.svg";
import Image_iconStatusModerate from "@/assets/iconStatusModerate.svg";
import Image_iconStatusBad from "@/assets/iconStatusBad.svg";

export default {
  props: {
    server: Object,
    isShowPingTime: Boolean,
  },
  computed: {
    pingStatusImg: function () {
      if (this.server == null) return null;
      switch (this.server.pingQuality) {
        case PingQuality.Good:
          return Image_iconStatusGood;
        case PingQuality.Moderate:
          return Image_iconStatusModerate;
        case PingQuality.Bad:
          return Image_iconStatusBad;
      }
      return null;
    },
  },

  methods: {},
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
.main {
  display: flex;
  align-items: center;
}

.pingtext {
  margin-left: 9px;
  color: var(--text-color-details);
}
</style>
