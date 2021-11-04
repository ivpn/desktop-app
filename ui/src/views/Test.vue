<template>
  <div class="about">
    <pre align="left">GETTERS {{ theGetters }}</pre>
    <pre align="left">STATE {{ theState }}</pre>
  </div>
</template>

<script>
function removeServers(key, value) {
  if (key == "vpnState/activeServers")
    return { DEBUG: `DEBUG: excluded from output (array len=${value.length})` };
  if (key == "serversHashed") return { DEBUG: `DEBUG: excluded from output` };
  if (key == "servers")
    return {
      DEBUG: `DEBUG: excluded from output (ovpn ${value.wireguard.length}; wg ${value.openvpn.length} )`,
    };
  else return value;
}

export default {
  components: {},
  data: function () {
    return {
      show: true,
    };
  },

  computed: {
    theGetters: function () {
      return JSON.stringify(this.$store.getters, removeServers, 2);
    },
    theState: function () {
      return JSON.stringify(this.$store.state, removeServers, 2);
    },
  },

  methods: {},
};
</script>

<style scoped lang="scss">
.about {
  -webkit-user-select: text;
  user-select: text;
  overflow-y: scroll;
  height: 100%;
}

.component-fade-enter-active,
.component-fade-leave-active {
  transition: opacity 0.3s ease;
}
.component-fade-enter, .component-fade-leave-to
/* .component-fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter, .fade-leave-to /* .fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}
</style>
