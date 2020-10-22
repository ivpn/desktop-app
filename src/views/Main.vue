<template>
  <div id="flexview">
    <div class="flexColumn">
      <div class="leftPanelTopSpace">
        <transition :name="minimizedButtonsTransition">
          <div
            v-if="isMinimizedButtonsVisible"
            class="minimizedButtonsPanel leftPanelTopMinimizedButtonsPanel"
          >
            <button v-on:click="onAccountSettings()">
              <img src="@/assets/user.svg" />
            </button>

            <button v-on:click="onSettings()">
              <img src="@/assets/settings.svg" />
            </button>

            <button v-on:click="onMaximize(true)">
              <img src="@/assets/maximize.svg" />
            </button>
          </div>
        </transition>
      </div>
      <div class="flexColumn" style="min-height: 0px">
        <transition name="fade" mode="out-in">
          <component
            v-bind:is="currentViewComponent"
            :onConnectionSettings="onConnectionSettings"
            :onWifiSettings="onWifiSettings"
            id="left"
          ></component>
        </transition>
      </div>
    </div>
    <div id="right" v-if="!isMinimizedUI">
      <transition name="fade" appear>
        <Map
          :isBlured="isMapBlured"
          :onAccountSettings="onAccountSettings"
          :onSettings="onSettings"
          :onMinimize="() => onMaximize(false)"
        />
      </transition>
    </div>
  </div>
</template>

<script>
const { remote, ipcRenderer } = require("electron");
import { DaemonConnectionType } from "@/store/types";

import { Platform, PlatformEnum } from "@/platform/platform";
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
      if (daemonConnection == null) return null;
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
    },
    minimizedButtonsTransition: function() {
      if (Platform() === PlatformEnum.Linux) return "smooth-display";
      return "fade";
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

div.minimizedButtonsPanel {
  display: flex;
  justify-content: flex-end;

  margin-right: 10px;
  margin-top: 10px;
}

div.minimizedButtonsPanel button {
  @extend .noBordersBtn;

  z-index: 1;
  cursor: pointer;

  padding: 0px;
  margin-left: 6px;
  margin-right: 6px;
}

div.minimizedButtonsPanel img {
  height: 18px;
}
</style>
