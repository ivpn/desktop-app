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
      class="triangle"
      style="margin-left: 18px"
      title="Launch application in Split Tunnel environment ..."
    >
      <select
        v-model="appToLaunch"
        id="appsList"
        style="
          cursor: pointer;
          position: absolute;
          opacity: 0;
          width: 22px;
          height: 22px;
          left: -10px;
        "
      >
        <option>[ Custom application... ]</option>
        <option v-for="item in sortedApps" :key="item" :value="item">
          {{ item.AppName ? item.AppName : item.AppBinaryPath }}
        </option>
      </select>
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
  data: function () {
    return {
      allInstalledApps: null,
    };
  },
  async mounted() {
    setTimeout(async () => {
      try {
        this.allInstalledApps = await sender.GetInstalledApps();
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
        // launch app ...
        if (value.AppBinaryPath) {
          try {
            await sender.SplitTunnelAddApp(value.AppBinaryPath);
          } catch (e) {
            processError(e);
          }
          return;
        }
        // manuall app ...
        try {
          let dlgFilters = [{ name: "All files", extensions: ["*"] }];

          var diagConfig = {
            properties: ["openFile"],
            filters: dlgFilters,
          };
          var ret = await sender.showOpenDialog(diagConfig);
          if (!ret || ret.canceled || ret.filePaths.length == 0) return;

          await sender.SplitTunnelAddApp(ret.filePaths[0]);
        } catch (e) {
          processError(e);
        }
      },
    },

    sortedApps: function () {
      let getApps = this.$store.getters["settings/FuncGetAppsToSplitTunnel"];
      if (!getApps) return null;
      return getApps(this.allInstalledApps);
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

.triangle {
  width: 0px;
  height: 0;
  border-left: 10px solid transparent;
  border-right: 10px solid transparent;
  border-bottom: 20px solid transparent; /* You can change the color */
  transform: rotate(90deg);
  border-bottom-color: #8b9aab;
}

.btn {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;
}

#selectBtn {
  @extend .btn;
  padding: 0px;

  display: flex;
  justify-content: space-between;
  align-items: center;

  width: 100%;
}
</style>
