<template>
  <div>
    <button
      title="Split Tunnel settings"
      id="selectBtn"
      class="flexRow"
      v-on:click="onShowSplitTunnelSettings"
    >
      <div align="left" class="flexRowRestSpace">
        <div class="large_text">
          Split Tunnel
          <span v-if="IsSplitTunnelInversed" style="color: var(--warning-color)"
            >Inverse</span
          >
          mode active
        </div>

        <div style="height: 5px" />
        <div class="small_text">
          {{ Description }}
        </div>
      </div>
    </button>

    <div
      style="margin-left: 18px"
      title="Launch application in Split Tunnel environment ..."
    >
      <div style="position: relative">
        <select
          v-model="appToLaunch"
          id="appsList"
          style="
            cursor: pointer;
            position: absolute;
            opacity: 0;
            width: 24px;
            height: 26px;
          "
        >
          <option>[ Custom application... ]</option>
          <option v-for="item in sortedApps" :key="item" :value="item">
            {{ item.AppName ? item.AppName : item.AppBinaryPath }}
          </option>
        </select>
        <img
          style="position: relative; z-index: -1"
          width="24"
          height="24"
          src="@/assets/plus.svg"
        />
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

function processError(e) {
  let errMes = e.toString();

  if (errMes && errMes.length > 0) {
    errMes = errMes.charAt(0).toUpperCase() + errMes.slice(1);
  }

  console.error(e);
  sender.showMessageBox({
    type: "error",
    buttons: ["OK"],
    message: errMes,
  });
}

export default {
  async mounted() {
    setTimeout(async () => {
      try {
        // request installed apps list (will be saved in store)
        await sender.GetInstalledApps();
      } catch (e) {
        console.error(e);
      }
    }, 0);
  },
  computed: {
    Description: function () {
      if (!this.IsSplitTunnelEnabled) return "";
      if (this.IsSplitTunnelEnabled && this.IsSplitTunnelInversed)
        return "Only specified applications utilize the VPN connection";
      return "Specified applications bypassing the VPN connection";
    },
    IsSplitTunnelEnabled: function () {
      return this.$store.state.vpnState.splitTunnelling?.IsEnabled;
    },
    IsSplitTunnelInversed: function () {
      return this.$store.state.vpnState.splitTunnelling?.IsInversed;
    },

    appToLaunch: {
      get() {
        return null;
      },
      async set(value) {
        try {
          let binaryPath = null;
          if (value.AppBinaryPath) binaryPath = value.AppBinaryPath;
          await sender.SplitTunnelAddApp(binaryPath);
        } catch (e) {
          processError(e);
        }
      },
    },

    sortedApps: function () {
      return this.$store.getters["settings/getAppsToSplitTunnel"];
    },
  },
  methods: {
    onShowSplitTunnelSettings() {
      sender.ShowSplitTunnelSettings();
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.large_text {
  font-size: 14px;
  line-height: 17px;
}

.small_text {
  font-size: 11px;
  line-height: 13px;
  color: var(--text-color-details);
}

#selectBtn {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  padding: 0px;

  display: flex;
  justify-content: space-between;
  align-items: center;

  width: 100%;
}
</style>
