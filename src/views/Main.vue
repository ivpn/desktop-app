<template>
  <div id="flexview">
    <div class="flexColumn">
      <div class="leftPanelTopSpace"></div>

      <div class="flexColumn" style="min-height: 0px">
        <transition name="fade" mode="out-in">
          <component
            v-bind:is="currentViewComponent"
            :onConnectionSettings="onConnectionSettings"
            :onWifiSettings="onWifiSettings"
            :onSettings="onSettings"
            :onAccountSettings="onAccountSettings"
            :onMaximize="onMaximize"
            id="left"
          ></component>
        </transition>
      </div>
    </div>
    <div id="right" v-if="!isMinimizedUI">
      <Map
        :isBlured="isMapBlured"
        :onAccountSettings="onAccountSettings"
        :onSettings="onSettings"
        :onMinimize="() => onMaximize(false)"
      />
    </div>
  </div>
</template>

<script>
const { remote, ipcRenderer } = require("electron");
import { DaemonConnectionType } from "@/store/types";

import Init from "@/components/Init.vue";
import Login from "@/components/Login.vue";
import Control from "@/components/Control.vue";
import Map from "@/components/Map.vue";

import config from "@/config";

export default {
  components: {
    Init,
    Login,
    Control,
    Map
  },
  computed: {
    isLoggedIn: function() {
      return this.$store.getters["account/isLoggedIn"];
    },
    currentViewComponent: function() {
      const daemonConnection = this.$store.state.daemonConnectionState;
      if (
        daemonConnection == null ||
        daemonConnection === DaemonConnectionType.NotConnected ||
        daemonConnection === DaemonConnectionType.Connecting
      )
        return Init;
      if (!this.isLoggedIn) return Login;
      return Control;
    },
    isMapBlured: function() {
      if (this.currentViewComponent !== Control) return "true";
      return "false";
    },
    isMinimizedButtonsVisible: function() {
      if (this.currentViewComponent !== Control) return false;
      if (this.$store.state.uiState.isDefaultControlView !== true) return false;
      return this.isMinimizedUI;
    },
    isMinimizedUI: function() {
      return this.$store.state.settings.minimizedUI;
    }
  },
  watch: {
    isMinimizedUI() {
      this.updateUIState();
    }
  },
  methods: {
    onAccountSettings: function() {
      //if (this.$store.state.settings.minimizedUI)
      ipcRenderer.send("renderer-request-show-settings-account");
      //else this.$router.push({ name: "settings", params: { view: "account" } });
    },
    onSettings: function() {
      ipcRenderer.send("renderer-request-show-settings-general");
    },
    onConnectionSettings: function() {
      ipcRenderer.send("renderer-request-show-settings-connection");
    },
    onWifiSettings: function() {
      ipcRenderer.send("renderer-request-show-settings-networks");
    },
    onMaximize: function(isMaximize) {
      this.$store.dispatch("settings/minimizedUI", !isMaximize);
      this.updateUIState();
    },
    updateUIState: function() {
      const win = remote.getCurrentWindow();
      const animate = false;
      if (this.isMinimizedUI)
        win.setBounds({ width: config.MinimizedUIWidth }, animate);
      else win.setBounds({ width: config.MaximizedUIWidth }, animate);
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

#flexview {
  display: flex;
  flex-direction: row;
  height: 100%;
}

#left {
  width: 320px;
  min-width: 320px;
  max-width: 320px;
}
#right {
  width: 0%; // ???
  flex-grow: 1;
}
</style>
