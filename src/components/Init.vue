<template>
  <div class="flexColumn">
    <div class="flexColumn" v-if="!(isConnecting && !isProcessing)">
      <div class="main">
        <spinner :loading="isProcessing" />
        <div class="large_text">Error connecting to IVPN daemon</div>
        <div class="small_text">
          {{ message }}
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
    <div v-else class="main small_text">
      Connecting ...
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
      this.isProcessing = true;
      setTimeout(async () => {
        try {
          await sender.ConnectToDaemon();
        } catch (e) {
          console.error(e);
        } finally {
          this.isProcessing = false;
        }
      }, 1500);
    },
    visitWebsite() {
      shell.openExternal(`https://www.ivpn.net`);
    }
  },
  computed: {
    isConnecting: function() {
      const connState = this.$store.state.daemonConnectionState;
      return connState == null || connState === DaemonConnectionType.Connecting;
    },
    message: function() {
      if (this.$store.state.daemonIsOldVersionError)
        return `Unsupported IVPN daemon version v${this.$store.state.daemonVersion} (minimum required v${config.MinRequiredDaemonVer}). Please, update IVPN daemon.`;
      return "Not connected to daemon. Please, ensure IVPN daemon is running and try to reconnect.";
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
