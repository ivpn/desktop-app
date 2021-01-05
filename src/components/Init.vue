<template>
  <div class="flexColumn">
    <spinner :loading="isProcessing" />

    <div class="main" v-if="isDaemonInstalling">
      Installing IVPN Daemon ...
      <div class="small_text" style="margin-top: 10px">
        Please follow the instructions in the dialog
      </div>
    </div>
    <div v-else-if="isConnecting" class="main small_text">
      Connecting ...
    </div>
    <div v-else class="flexColumn">
      <div class="main">
        <div class="large_text">Error connecting to IVPN daemon</div>
        <div v-if="daemonIsOldVersionError">
          <div class="small_text">
            Unsupported IVPN daemon version v{{ currDaemonVer }} (minimum
            required v{{ minRequiredVer }}).
          </div>
          <div class="small_text">
            Please update daemon by downloading latest version from
            <button
              class="noBordersTextBtn settingsLinkText"
              v-on:click="visitWebsite"
            >
              IVPN website</button
            >.
          </div>
        </div>
        <div v-else>
          <div class="small_text">
            Not connected to daemon. Please, ensure IVPN daemon is running and
            try to reconnect.
          </div>
          <div class="small_text">
            The latest daemon version can be downloaded from
            <button
              class="noBordersTextBtn settingsLinkText"
              v-on:click="visitWebsite"
            >
              IVPN website</button
            >.
          </div>
        </div>
        <button class="btn" v-on:click="ConnectToDaemon">Retry ...</button>
      </div>
      <button
        class="noBordersTextBtn settingsLinkText"
        v-on:click="visitWebsite"
      >
        www.ivpn.net
      </button>
    </div>
  </div>
</template>

<script>
const { shell } = require("electron");
import spinner from "@/components/controls/control-spinner.vue";
import { DaemonConnectionType } from "@/store/types";
import sender from "./../ipc/renderer-sender";
import config from "@/config";

export default {
  components: {
    spinner
  },
  data: function() {
    return {
      isProcessing: false
    };
  },
  methods: {
    async ConnectToDaemon() {
      try {
        await sender.ConnectToDaemon();
      } catch (e) {
        console.error(e);
      }
    },
    visitWebsite() {
      shell.openExternal(`https://www.ivpn.net`);
    }
  },
  computed: {
    isDaemonInstalling: function() {
      return this.$store.state.daemonIsInstalling;
    },
    isConnecting: function() {
      const connState = this.$store.state.daemonConnectionState;
      return connState == null || connState === DaemonConnectionType.Connecting;
    },
    minRequiredVer: function() {
      return config.MinRequiredDaemonVer;
    },
    currDaemonVer: function() {
      return this.$store.state.daemonVersion;
    },
    daemonIsOldVersionError: function() {
      return this.$store.state.daemonIsOldVersionError;
    }
  },
  watch: {
    isConnecting() {
      this.isProcessing = this.isConnecting;
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.main {
  padding: 15px;
  margin-top: -100px;
  height: 100%;

  display: flex;
  flex-flow: column;
  justify-content: center;
  align-items: center;
}

.large_text {
  margin: 12px;
  font-weight: 600;
  font-size: 18px;
  line-height: 120%;

  color: #2a394b;
}

.small_text {
  margin: 12px;
  margin-top: 0px;

  font-size: 13px;
  line-height: 17px;
  letter-spacing: -0.208px;

  color: #98a5b3;
}

.btn {
  margin: 30px 0 0 0;
  width: 90%;
  height: 28px;
  background: #ffffff;
  border-radius: 10px;
  border: 1px solid #7d91a5;

  font-size: 15px;
  line-height: 20px;
  text-align: center;
  letter-spacing: -0.4px;
  color: #6d849a;

  cursor: pointer;
}
</style>
