<template>
  <div id="main">
    <!-- Loading connection info ... -->
    <div
      v-if="
        (!isIPv6View && isRequestingLocationIPv4) ||
        (isIPv6View && isRequestingLocationIPv6)
      "
      style="text-align: center"
      class="descriptipn"
    >
      Loading connection info ...
    </div>

    <!-- Main view -->
    <div
      v-if="
        (!isIPv6View && !isRequestingLocationIPv4) ||
        (isIPv6View && !isRequestingLocationIPv6)
      "
      style="width: 100%"
    >
      <!-- Failed to load connection info -->
      <div
        v-if="!isInfoAvailableIPv4 && !isInfoAvailableIPv6"
        style="text-align: center"
        class="descriptipn"
      >
        Failed to load connection info
        <div>
          <button
            class="noBordersBtn"
            style="pointer-events: auto"
            @click="onRefreshCurrLocation"
          >
            <img width="10" height="10" src="@/assets/refresh.svg" />
          </button>
        </div>
      </div>
      <!-- connection info -->
      <div v-if="isInfoAvailableIPv4 || isInfoAvailableIPv6">
        <!-- IPV4 / IPV6 buttons-->
        <div v-if="isInfoAvailableIPv4 && isInfoAvailableIPv6" class="flexRow">
          <div class="flexRow leftColumn">
            <div class="flexRow badgeContainer">
              <button
                class="badge"
                :class="{ badgeSelected: !isIPv6View }"
                style="pointer-events: auto"
                @click="onIPv4View"
              >
                IPv4
              </button>

              <button
                class="badge"
                :class="{ badgeSelected: isIPv6View }"
                style="pointer-events: auto"
                @click="onIPv6View"
              >
                IPv6
              </button>
            </div>
          </div>
          <span v-if="!isIPv4andIPv6LocationsEqual" style="opacity: 0.5">
            Location does not match
          </span>
        </div>

        <div class="flexRow row">
          <div class="descriptipn">IP Address</div>

          <div class="flexRow" style="overflow: hidden">
            <div
              style="
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
              "
            >
              {{ ip }}
            </div>

            <div style="vertical-align: top">
              <button
                class="noBordersBtn"
                style="
                  padding: 0px;
                  margin: 0px;
                  margin-left: 4px;
                  pointer-events: auto;
                "
                @click="onRefreshCurrLocation"
              >
                <img width="10" height="10" src="@/assets/refresh.svg" />
              </button>
            </div>
          </div>
        </div>

        <div class="flexRow row">
          <div class="descriptipn">Location</div>
          <div class="value">{{ locationName }}</div>
        </div>
        <div class="flexRow row">
          <div class="descriptipn">ISP</div>
          <div class="value">{{ isp }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  computed: {
    isIPv6View: function () {
      return this.$store.getters["getIsIPv6View"];
    },
    isIPv4andIPv6LocationsEqual: function () {
      return this.$store.getters["isIPv4andIPv6LocationsEqual"];
    },
    isInfoAvailableIPv4: function () {
      return this.$store.getters["getIsInfoAvailableIPv4"];
    },
    isInfoAvailableIPv6: function () {
      return this.$store.getters["getIsInfoAvailableIPv6"];
    },

    // isRequestingLocation
    isRequestingLocationIPv4: function () {
      return this.$store.state.isRequestingLocation;
    },
    isRequestingLocationIPv6: function () {
      return this.$store.state.isRequestingLocationIPv6;
    },

    ip: function () {
      let l = this.$store.state.location;
      if (this.isIPv6View) l = this.$store.state.locationIPv6;
      if (!l || !l.ip_address) return null;
      return l.ip_address;
    },

    locationName: function () {
      let l = this.$store.state.location;
      if (this.isIPv6View) l = this.$store.state.locationIPv6;
      if (!l) return null;

      if (l.city) {
        if (l.country_code) return `${l.city}, ${l.country_code}`;
        else if (l.country) return `${l.city}, ${l.country}`;
      } else if (l.country) return `${l.country}`;
      return null;
    },
    isp: function () {
      let l = this.$store.state.location;
      if (this.isIPv6View) l = this.$store.state.locationIPv6;
      if (!l) return null;
      if (l.isIvpnServer == true) return "IVPN";
      if (!l.isp) return null;
      return l.isp;
    },
  },
  methods: {
    onRefreshCurrLocation() {
      try {
        sender.GeoLookup();
      } catch (e) {
        console.error(e);
      }
    },
    onIPv4View() {
      this.$store.dispatch("uiState/isIPv6View", false);
    },
    onIPv6View() {
      this.$store.dispatch("uiState/isIPv6View", true);
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

#main {
  font-size: 12px;
  line-height: 20px;
  min-height: 60px;
}
div.row {
  margin-top: 4px;
  margin-bottom: 4px;
}
div.leftColumn {
  min-width: 100px;
}
div.descriptipn {
  @extend .leftColumn;
  opacity: 0.5;
}
button.badge {
  @extend .noBordersBtn;
  color: white;
  background: transparent;
  border-radius: 4px;
  min-width: 45px;
  min-height: 22px;
}

button.badge:hover {
  opacity: 0.7;
}

button.badgeSelected {
  background: rgba(130, 130, 130, 0.6);
}

.badgeContainer {
  padding: 2px;

  border-radius: 5px;
  background: rgba(99, 99, 99, 0.3);
}
</style>
