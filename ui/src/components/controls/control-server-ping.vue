<template>
  <div class="main flexRow">
    <img :src="pingStatusImg" />

    <div class="pingtext marginLeft" v-if="isShowPingTime && ping > 0">
      {{ ping }}ms
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
  data: function () {
    return {
      funcGetPing: null,
    };
  },
  mounted() {
    this.funcGetPing = this.$store.getters["vpnState/funcGetPing"];
  },
  computed: {
    pingStatusImg: function () {
      const quality = this.getPingQuality(this.ping);

      switch (quality) {
        case PingQuality.Good:
          return Image_iconStatusGood;
        case PingQuality.Moderate:
          return Image_iconStatusModerate;
        case PingQuality.Bad:
          return Image_iconStatusBad;
      }
      return null;
    },

    ping: function () {
      if (!this.funcGetPing) return null;
      const ret = this.funcGetPing(this.server);
      return ret;
    },
  },
  methods: {
    getPingQuality: function (pingMs) {
      if (pingMs == null || pingMs == undefined) return null;
      if (pingMs < 100) return PingQuality.Good;
      if (pingMs < 300) return PingQuality.Moderate;
      return PingQuality.Bad;
    },
  },
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
  width: 100%;
  text-align: right;
  padding-right: 10px;
  margin-left: 9px;
  color: var(--text-color-details);
}
</style>
