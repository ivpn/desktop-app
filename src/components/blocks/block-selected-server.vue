<template>
  <div id="main">
    <button class="serverSelectBtn" v-on:click="showServersList()">
      <div align="left">
        <div class="small_text">
          {{
            this.isExitServer
              ? "Exit server"
              : isConnected
              ? "Connected to"
              : isConnecting
              ? "Connecting to ..."
              : "Connect to"
          }}
        </div>
        <div style="height: 4px" />
        <serverNameControl
          class="serverName"
          size="large"
          :server="this.server"
          :isFastestServer="
            isDisconnected && $store.getters['settings/isFastestServer']
          "
          :isRandomServer="
            isDisconnected && $store.getters['settings/isRandomServer']
          "
          :isShowPingPicture="
            !isDisconnected ||
              (isDisconnected &&
                !(
                  $store.getters['settings/isFastestServer'] ||
                  $store.getters['settings/isRandomServer']
                ))
          "
        />
      </div>

      <div class="serverSelectArrow"></div>
    </button>
  </div>
</template>

<script>
import serverNameControl from "@/components/controls/control-server-name.vue";
import { VpnStateEnum } from "@/store/types";

export default {
  props: ["onShowServersPressed", "isExitServer"],
  components: {
    serverNameControl
  },
  computed: {
    server: function() {
      return this.isExitServer
        ? this.$store.state.settings.serverExit
        : this.$store.state.settings.serverEntry;
    },
    isConnected: function() {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      );
    },
    isConnecting: function() {
      return this.$store.getters["vpnState/isConnecting"];
    },
    isDisconnected: function() {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      );
    }
  },
  methods: {
    showServersList() {
      if (this.onShowServersPressed != null)
        this.onShowServersPressed(this.isExitServer);
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

#main {
  @extend .left_panel_block;
}

.small_text {
  font-size: 14px;
  line-height: 17px;
  letter-spacing: -0.3px;
  color: $base-text-color-details;
}

.serverSelectArrow {
  border: solid #8b9aab;
  border-width: 0 1px 1px 0;
  display: inline-block;

  padding: 4px;
  transform: rotate(-45deg);
  -webkit-transform: rotate(-45deg);
}

.serverSelectBtn {
  padding: 0px;
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  display: flex;
  justify-content: space-between;
  align-items: center;

  height: 82px;
  width: 100%;
}

.serverName {
  max-width: 270px;
}
</style>
