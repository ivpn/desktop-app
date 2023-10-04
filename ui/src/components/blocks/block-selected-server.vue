<template>
  <div id="main">
    <button class="serverSelectBtn" v-on:click="showServersList()">
      <div class="flexRow" style="height: 100%">
        <div class="flexColumn" align="left">
          <div class="small_text" style="margin-top: 8px">
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
          <div style="min-height: 4px" />
          <div class="flexRow">
            <serverNameControl
              class="serverName"
              style="max-width: 245px"
              SecondLineMaxWidth="245px"
              :isLargeText="true"
              :server="this.server"
              :serverHostName="this.serverHostName"
              :isFastestServer="isFastestServer"
              :isRandomServer="isRandomServer"
              :isShowPingPicture="!(isFastestServer || isRandomServer)"
            />
          </div>
        </div>

        <div class="flexRow flexRowRestSpace" />

        <serverPingInfoControl
          v-show="!(isFastestServer || isRandomServer)"
          :server="this.server"
          style="margin-left: 9px; margin-right: 8px"
        />

        <div class="arrowRightSimple"></div>
      </div>
    </button>
  </div>
</template>

<script>
import serverNameControl from "@/components/controls/control-server-name.vue";
import serverPingInfoControl from "@/components/controls/control-server-ping.vue";
import { VpnStateEnum } from "@/store/types";

export default {
  props: ["onShowServersPressed", "isExitServer"],
  components: {
    serverNameControl,
    serverPingInfoControl,
  },
  computed: {
    server: function () {
      return this.isExitServer
        ? this.$store.state.settings.serverExit
        : this.$store.state.settings.serverEntry;
    },
    serverHostName: function () {
      return this.isExitServer
        ? this.$store.state.settings.serverExitHostId
        : this.$store.state.settings.serverEntryHostId;
    },
    isConnected: function () {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      );
    },
    isConnecting: function () {
      return this.$store.getters["vpnState/isConnecting"];
    },
    isDisconnected: function () {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      );
    },
    isFastestServer: function () {
      if (
        (this.isDisconnected || this.$store.state.vpnState.isPingingServers) &&
        this.$store.getters["settings/isFastestServer"]
      )
        return true;
      return false;
    },
    isRandomServer: function () {
      if (!this.isDisconnected) return false;
      return this.isExitServer
        ? this.$store.getters["settings/isRandomExitServer"]
        : this.$store.getters["settings/isRandomServer"];
    },
  },
  methods: {
    showServersList() {
      if (this.onShowServersPressed != null)
        this.onShowServersPressed(this.isExitServer);
    },
  },
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
  color: var(--text-color-details);
}

.serverSelectBtn {
  padding: 0px;
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  height: 82px;
  width: 100%;

  padding-bottom: 4px;
}

.serverName {
  max-width: 270px;
}
</style>
