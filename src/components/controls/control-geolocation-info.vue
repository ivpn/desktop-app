<template>
  <div id="main">
    <div
      v-if="isRequestingLocation"
      style="text-align: center"
      class="descriptipn"
    >
      Loading connection info ...
    </div>
    <div v-if="!isRequestingLocation">
      <div
        v-if="!isInfoAvailable"
        style="text-align: center"
        class="descriptipn"
      >
        Failed to load connection info
      </div>
      <div v-if="isInfoAvailable">
        <div class="flexRow row">
          <div class="descriptipn">Your IP</div>
          <div class="value">{{ ip }}</div>
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
export default {
  computed: {
    isRequestingLocation: function() {
      return this.$store.state.isRequestingLocation;
    },
    isInfoAvailable: function() {
      let l = this.$store.state.location;
      if (!l) return false;
      if (!l.city && !l.country && !l.isp && !l.ip_address) return false;
      return true;
    },
    ip: function() {
      let l = this.$store.state.location;
      if (!l || !l.ip_address) return null;
      return this.$store.state.location.ip_address;
    },
    locationName: function() {
      let l = this.$store.state.location;
      if (!l) return null;

      if (l.city) {
        if (l.country_code) return `${l.city}, ${l.country_code}`;
        else if (l.country) return `${l.city}, ${l.country}`;
      } else if (l.country) return `${l.country}`;
      return null;
    },
    isp: function() {
      let l = this.$store.state.location;
      if (!l) return null;
      if (l.isIvpnServer == true) return "IVPN";
      if (!l.isp) return null;
      return l.isp;
    }
  }
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
div.descriptipn {
  min-width: 100px;
  opacity: 0.5;
}
</style>
