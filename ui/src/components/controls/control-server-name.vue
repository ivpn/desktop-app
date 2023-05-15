<template>
  <div class="main">
    <div class="flexColumn" style="min-width: 24px">
      <img
        class="pic"
        v-bind:class="{
          flag: isCountryFlagInUse,
        }"
        :src="serverImage"
        v-show="isImgLoadError !== true"
        @error="onImgLoadError"
      />
    </div>

    <div
      class="textBlock text"
      v-if="isShowSingleLine"
      v-bind:class="{ text_large: isLargeText, firstLine: !isSingleLine }"
    >
      {{ singleLine }}
    </div>
    <div class="textBlock" v-else>
      <div class="firstLine flexRow" style="max-width: 154px">
        <div class="text">{{ multilineFirstLine }}</div>
        <div>
          <button
            class="noBordersBtn expandButton"
            v-if="onExpandClick != null && isExpanded !== undefined"
            v-on:click.stop
            v-on:click="onExpandClick(server)"
          >
            <img
              v-if="isExpanded"
              style="transform: rotate(180deg)"
              src="@/assets/arrow-bottom.svg"
            />
            <img v-else src="@/assets/arrow-bottom.svg" />
          </button>
        </div>
      </div>

      <div
        class="text secondLine flexRow"
        :style="{ maxWidth: SecondLineMaxWidth }"
      >
        <div v-if="isShowIPVersionBage && isIPv6" class="bage">IPv6</div>
        {{ multilineSecondLine }} {{ selectedHostInfo }}
        <div
          v-show="showISPInfo && isFavoriteServersView === false"
          style="margin-left: 4px"
          class="text secondLine"
        >
          {{ isp }}
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { PingQuality } from "@/store/types";
import { IsServerSupportIPv6 } from "@/helpers/helpers_servers";

import Image_speedometer from "@/assets/speedometer.svg";
import Image_shuffle from "@/assets/shuffle.svg";
import Image_iconStatusGood from "@/assets/iconStatusGood.svg";
import Image_iconStatusModerate from "@/assets/iconStatusModerate.svg";
import Image_iconStatusBad from "@/assets/iconStatusBad.svg";

export default {
  props: {
    server: Object,
    serverHostName: String, // in use on Main view to show selected host for selected server
    isFavoriteServersView: Boolean,
    isLargeText: Boolean,

    isSingleLine: Boolean,
    isCountryFirst: Boolean,

    isFullName: String,

    isFastestServer: Boolean,
    isRandomServer: Boolean,

    onExpandClick: Function,
    isExpanded: Boolean,
    SecondLineMaxWidth: String,
  },
  data: () => ({
    isImgLoadError: false,
  }),
  computed: {
    isShowSingleLine: function () {
      return (
        this.isSingleLine ||
        this.isFullName ||
        this.isFastestServer ||
        this.isRandomServer
      );
    },

    singleLine: function () {
      if (this.isFastestServer === true) return "Fastest server";
      if (this.isRandomServer === true) return "Random server";
      if (!this.server) return "";
      if (!this.server.city && !this.server.country) return "";
      if (!this.server.city) return this.server.country;

      if (this.isCountryFirst) {
        if (this.isFullName === "true")
          return `${this.server.country}, ${this.server.city}`;
        return `${this.server.country_code}, ${this.server.city}`;
      } else {
        if (this.isFullName === "true")
          return `${this.server.city}, ${this.server.country}`;
        return `${this.server.city}, ${this.server.country_code}`;
      }
    },
    selectedHostInfo: function () {
      if (!this.serverHostName) return "";
      return "(" + this.serverHostName.split(".")[0] + ")";
    },
    isp: function () {
      if (!this.server || !this.server.isp) return "";
      return "(ISP: " + this.server.isp + ")";
    },
    showISPInfo: function () {
      return this.$store.state.settings.showISPInfo;
    },

    multilineFirstLine: function () {
      if (!this.server) return "";
      if (this.isCountryFirst) return this.server.country;
      return this.server.city;
    },
    multilineSecondLine: function () {
      if (!this.server) return "";

      if (this.isFavoriteServersView && this.$store.state.settings.showHosts) {
        // only for favorite hosts (host object extended by all properties from parent server object +favHostParentServerObj +favHost)
        let favHost = this.server.favHost;
        // if favorite server has only one host: show it as a host
        if (this.server.hosts.length === 1) {
          favHost = this.server.hosts[0];
        }
        if (favHost)
          return `${favHost.hostname} (${Math.round(favHost.load)}%)`;
      }

      if (this.isCountryFirst) return this.server.city;
      return this.server.country;
    },
    isCountryFlagInUse: function () {
      return this.isFastestServer !== true && this.isRandomServer !== true;
    },
    isShowIPVersionBage: function () {
      return (
        this.$store.state.settings.enableIPv6InTunnel && // IPv6 enabled
        this.$store.state.settings.showGatewaysWithoutIPv6 && // and we show both types of servers (IPv4 and IPv6)
        this.isIPv6 === true
      );
    },
    isIPv6: function () {
      return IsServerSupportIPv6(this.server);
    },
    serverImage: function () {
      if (this.isFastestServer === true) return Image_speedometer;
      if (this.isRandomServer === true) return Image_shuffle;
      if (!this.server) return `/flags/unk.svg`;
      try {
        const ccode = this.server.country_code.toUpperCase();
        return `/flags/svg/${ccode}.svg`;
      } catch (e) {
        console.log(e);
        return null;
      }
    },
    pingStatusImg: function () {
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
    },
  },

  methods: {
    onImgLoadError() {
      this.isImgLoadError = true;
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
.main {
  display: flex;
}

img.pic {
  width: 22px;
  margin: 1px;
  margin-top: 2.4px;
}

img.flag {
  //border: 1px solid rgba(var(--flag-border-color-rgb), 0.5);

  //border-radius: 2px;
  box-shadow: 0 0 0.4pt 0.4pt rgba(var(--flag-border-color-rgb), 1);
}

.text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

div.textBlock {
  font-size: 14px;
  line-height: 20.8px;
  margin-left: 16px;
}

.text_large {
  font-size: 18px;
  line-height: 21px;
  margin-left: 10px;
}

div.firstLine {
  text-align: left;
  font-size: 16px;
}

div.secondLine {
  text-align: left;
  font-size: 12px;
  color: grey;
}

div.bage {
  font-size: 10px;
  line-height: 14px;
  color: grey;
  display: inline-block;
  padding-left: 4px;
  padding-right: 4px;
  border: 1px solid #777777;
  border-radius: 4px;
  margin-right: 4px;
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

button.expandButton {
  opacity: 0.6;
}
button.expandButton:hover {
  opacity: 1;
}
</style>
