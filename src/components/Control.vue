<template>
  <div class="flexColumn">
    <transition mode="out-in">
      <div
        class="flexColumn"
        v-if="uiView === 'serversEntry'"
        key="entryServers"
      >
        <Servers
          :onBack="backToMainView"
          :onServerChanged="onServerChanged"
          :onFastestServer="onFastestServer"
          :onRandomServer="onRandomServer"
        />
      </div>

      <div
        class="flexColumn"
        v-else-if="uiView === 'serversExit'"
        key="exitServers"
      >
        <Servers
          :onBack="backToMainView"
          isExitServer="true"
          :onServerChanged="onServerChanged"
          :onRandomServer="() => onRandomServer(true)"
        />
      </div>

      <div v-else class="flexColumn">
        <div>
          <ConnectBlock
            :onChecked="switchChecked"
            :isChecked="isConnected"
            :isProgress="isInProgress"
            :onPauseResume="onPauseResume"
            :pauseState="this.$store.state.vpnState.pauseState"
          />
          <div class="horizontalLine hopButtonsSeparator" />
        </div>

        <div ref="scrollArea" class="scrollableColumnContainer">
          <div v-if="isMultihopAllowed">
            <HopButtonsBlock />
            <div class="horizontalLine hopButtonsSeparator" />
          </div>

          <SelectedServerBlock :onShowServersPressed="onShowServersPressed" />

          <div v-if="this.$store.state.settings.isMultiHop">
            <div class="horizontalLine" />
            <SelectedServerBlock
              :onShowServersPressed="onShowServersPressed"
              isExitServer="true"
            />
          </div>

          <ConnectionDetailsBlock
            :onShowPorts="onShowPorts"
            :onShowWifiConfig="onShowWifiConfig"
          />

          <transition name="fade">
            <button
              class="btnScrollDown"
              v-if="isShowScrollButton"
              v-on:click="onScrollDown()"
            >
              <img src="@/assets/arrow-bottom.svg" />
            </button>
          </transition>
        </div>
      </div>
    </transition>
  </div>
</template>

<script>
const { dialog, getCurrentWindow } = require("electron").remote;

import Servers from "./Servers.vue";
import ConnectBlock from "./blocks/block-connect.vue";
import ConnectionDetailsBlock from "./blocks/block-connection-details.vue";
import SelectedServerBlock from "@/components/blocks/block-selected-server.vue";
import HopButtonsBlock from "./blocks/block-hop-buttons.vue";

import sender from "@/ipc/renderer-sender";
import { VpnStateEnum, VpnTypeEnum, PauseStateEnum } from "@/store/types";
import { isStrNullOrEmpty } from "@/helpers/helpers";

const viewTypeEnum = Object.freeze({
  default: "default",
  serversEntry: "serversEntry",
  serversExit: "serversExit"
});

async function connect(me, isConnect) {
  try {
    me.isConnectProgress = true;
    if (isConnect === true) await sender.Connect();
    else await sender.Disconnect();
  } catch (e) {
    console.error(e);
    dialog.showMessageBoxSync(getCurrentWindow(), {
      type: "error",
      buttons: ["OK"],
      message: `Failed to ${isConnect ? "connect" : "disconnect"}: ` + e
    });
  } finally {
    me.isConnectProgress = false;
  }
}

function connected(me) {
  return me.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED;
}

export default {
  props: {
    onConnectionSettings: Function,
    onWifiSettings: Function
  },

  components: {
    HopButtonsBlock,
    Servers,
    ConnectBlock,
    SelectedServerBlock,
    ConnectionDetailsBlock
  },
  mounted() {
    this.scrollArea = this.$refs.scrollArea;
    this.scrollArea.addEventListener(
      "scroll",
      this.recalcScrollButtonVisiblity
    );
    this.recalcScrollButtonVisiblity();
  },
  data: function() {
    return {
      scrollArea: null,
      isShowScrollButton: false,
      isConnectProgress: false,
      uiView: viewTypeEnum.default,
      lastServersPingRequestTime: null
    };
  },

  computed: {
    isConnected: function() {
      return connected(this);
    },
    isOpenVPN: function() {
      return this.$store.state.settings.vpnType === VpnTypeEnum.OpenVPN;
    },
    isMultihopAllowed: function() {
      return this.isOpenVPN && this.$store.getters["account/isMultihopAllowed"];
    },
    port: function() {
      // needed for watcher
      return this.$store.getters["settings/getPort"];
    },
    isInProgress: function() {
      if (this.isConnectProgress) return this.isConnectProgress;
      return (
        this.$store.state.vpnState.connectionState !== VpnStateEnum.CONNECTED &&
        this.$store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED
      );
    },
    // needed for watcher
    connectionFailureInfo: function() {
      return this.$store.state.vpnState.disconnectedInfo;
    },
    isMinimizedUI: function() {
      return this.$store.state.settings.minimizedUI;
    },
    isMultiHop: function() {
      return this.$store.state.settings.isMultiHop;
    }
  },

  watch: {
    connectionFailureInfo() {
      if (
        this.connectionFailureInfo != null &&
        !isStrNullOrEmpty(this.connectionFailureInfo.ReasonDescription) &&
        this.connectionFailureInfo.ReasonDescription.length > 0
      ) {
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "error",
          buttons: ["OK"],
          message: `Failed to connect`,
          detail: this.connectionFailureInfo.ReasonDescription
        });
      }
    },
    port(newValue, oldValue) {
      if (!connected(this)) return;
      if (newValue == null || oldValue == null) return;
      if (newValue.port === oldValue.port && newValue.type === oldValue.type)
        return;
      connect(this, true);
    },
    isMinimizedUI() {
      setTimeout(() => this.recalcScrollButtonVisiblity(), 0);
    },
    isMultiHop() {
      setTimeout(() => this.recalcScrollButtonVisiblity(), 0);
    }
  },

  methods: {
    async switchChecked(isConnect) {
      connect(this, isConnect);
    },
    async onPauseResume(seconds) {
      if (this.$store.state.vpnState.pauseState !== PauseStateEnum.Resumed) {
        // RESUME
        await sender.ResumeConnection();
      } else if (seconds != null) {
        // PAUSE
        await sender.PauseConnection(seconds);
      }
    },
    onShowServersPressed(isExitServers) {
      this.uiView = isExitServers
        ? viewTypeEnum.serversExit
        : viewTypeEnum.serversEntry;
      this.$store.dispatch("uiState/isDefaultControlView", false);

      // request servers ping not more often than once per 30 seconds
      if (
        this.lastServersPingRequestTime == null ||
        (new Date().getTime() - this.lastServersPingRequestTime.getTime()) /
          1000 >
          30
      ) {
        sender.PingServers();
        this.lastServersPingRequestTime = new Date();
      }
    },
    onShowPorts() {
      if (this.onConnectionSettings != null) this.onConnectionSettings();
    },
    onShowWifiConfig() {
      if (this.onWifiSettings != null) this.onWifiSettings();
    },
    backToMainView() {
      this.uiView = viewTypeEnum.default;
      this.$store.dispatch("uiState/isDefaultControlView", true);
    },
    onServerChanged(server, isExitServer) {
      if (server == null || isExitServer == null) return;
      if (!isExitServer) {
        if (server === this.$store.state.settings.serverEntry) return;
        this.$store.dispatch("settings/isRandomServer", false);
        this.$store.dispatch("settings/serverEntry", server);
      } else {
        if (server === this.$store.state.settings.serverExit) return;
        this.$store.dispatch("settings/isRandomExitServer", false);
        this.$store.dispatch("settings/serverExit", server);
      }
      this.$store.dispatch("settings/isFastestServer", false);

      if (connected(this)) connect(this, true);
    },
    onFastestServer() {
      this.$store.dispatch("settings/isFastestServer", true);
      if (connected(this)) connect(this, true);
    },
    onRandomServer(isExitServer) {
      if (isExitServer === true)
        this.$store.dispatch("settings/isRandomExitServer", true);
      else this.$store.dispatch("settings/isRandomServer", true);
      if (connected(this)) connect(this, true);
    },
    recalcScrollButtonVisiblity() {
      if (this.scrollArea == null) {
        this.isShowScrollButton = false;
        return;
      }
      this.isShowScrollButton =
        this.scrollArea.scrollHeight >
        this.scrollArea.clientHeight + this.scrollArea.scrollTop;
    },
    onScrollDown() {
      if (this.scrollArea == null) return;
      this.scrollArea.scrollTo({
        top: this.scrollArea.scrollHeight,
        behavior: "smooth"
      });
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
$shadow: 0px 3px 1px rgba(0, 0, 0, 0.06), 0px 3px 8px rgba(0, 0, 0, 0.15);

button.btnScrollDown {
  position: fixed;

  z-index: 7;

  bottom: 0;
  margin-bottom: 8px;

  left: calc(320px / 2 - 12px);
  //margin-left: calc(50% - 12px);

  width: 24px;
  height: 24px;

  padding: 0px;
  border: none;
  border-radius: 50%;
  background-color: #ffffff;
  outline-width: 0;
  cursor: pointer;

  box-shadow: $shadow;

  // centering content
  display: flex;
  justify-content: center;
  align-items: center;
}
button.btnScrollDown:hover {
  background-color: #f0f0f0;
}
</style>
