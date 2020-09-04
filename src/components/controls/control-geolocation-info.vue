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
        Loading connection info failed
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
      return this.$store.state.location != null;
    },
    ip: function() {
      if (this.$store.state.location == null) return null;
      return this.$store.state.location.ip_address;
    },
    locationName: function() {
      if (this.$store.state.location == null) return null;
      return `${this.$store.state.location.city}, ${this.$store.state.location.country_code}`;
    },
    isp: function() {
      if (this.$store.state.location == null) return null;
      return this.$store.state.location.isp;
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
  text-shadow: 0px 1px 0px #ffffff;
}
div.value {
  text-shadow: 0px 1px 0px #ffffff;
}
</style>
