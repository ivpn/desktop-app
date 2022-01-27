<template>
  <div id="flexview">
    <div class="flexColumn">
      <div class="leftPanelTopSpace">
        <transition :name="minimizedButtonsTransition">
          <div
            v-if="isMinimizedButtonsVisible"
            class="minimizedButtonsPanel leftPanelTopMinimizedButtonsPanel"
            :class="{
              minimizedButtonsPanelRightElements: isWindowHasFrame,
            }"
          >
            <button @click="onAccountSettings()">
              <img src="@/assets/user.svg" />
            </button>

            <button @click="onSettings()">
              <img src="@/assets/settings.svg" />
            </button>

            <button @click="onMaximize(true)">
              <img src="@/assets/maximize.svg" />
            </button>
          </div>
        </transition>
      </div>
      <div class="flexColumn" style="min-height: 0px">
        <transition name="fade" mode="out-in">
          <component
            :is="currentViewComponent"
            id="left"
            :on-connection-settings="onConnectionSettings"
            :on-wifi-settings="onWifiSettings"
            :on-default-view="onDefaultLeftView"
          ></component>
        </transition>
      </div>
    </div>
    <div v-if="!isMinimizedUI" id="right">
      <transition name="fade" appear>
        <Map
          :is-blured="isMapBlured"
          :on-account-settings="onAccountSettings"
          :on-settings="onSettings"
          :on-minimize="() => onMaximize(false)"
        />
      </transition>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

import { DaemonConnectionType } from "@/store/types";
import { Platform, PlatformEnum, IsWindowHasFrame } from "@/platform/platform";
import Init from "@/components/Init.vue";
import Login from "@/components/Login.vue";
import Control from "@/components/Control.vue";
import Map from "@/components/Map.vue";

export default {
  components: {
    Init,
    Login,
    Control,
    Map,
  },
  data: function () {
    return {
      isCanShowMinimizedButtons: true,
    };
  },
  computed: {
    isWindowHasFrame: function () {
      return IsWindowHasFrame();
    },
    isLoggedIn: function () {
      return this.$store.getters["account/isLoggedIn"];
    },
    currentViewComponent: function () {
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
    isMapBlured: function () {
      if (this.currentViewComponent !== Control) return "true";
      return "false";
    },
    isMinimizedButtonsVisible: function () {
      if (this.currentViewComponent !== Control) return false;
      if (this.isCanShowMinimizedButtons !== true) return false;
      return this.isMinimizedUI;
    },
    isMinimizedUI: function () {
      return this.$store.state.settings.minimizedUI;
    },
    minimizedButtonsTransition: function () {
      if (Platform() === PlatformEnum.Linux) return "smooth-display";
      return "fade";
    },
  },
  watch: {
    isMinimizedUI() {
      this.updateUIState();
    },
  },
  methods: {
    onAccountSettings: function () {
      //if (this.$store.state.settings.minimizedUI)
      sender.ShowAccountSettings();
      //else this.$router.push({ name: "settings", params: { view: "account" } });
    },
    onSettings: function () {
      sender.ShowSettings();
    },
    onConnectionSettings: function () {
      sender.ShowConnectionSettings();
    },
    onWifiSettings: function () {
      sender.ShowWifiSettings();
    },
    onDefaultLeftView: function (isDefaultView) {
      this.isCanShowMinimizedButtons = isDefaultView;
    },
    onMaximize: function (isMaximize) {
      this.$store.dispatch("settings/minimizedUI", !isMaximize);
      this.updateUIState();
    },
    updateUIState: function () {
      sender.uiMinimize(this.isMinimizedUI);
    },
  },
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

div.minimizedButtonsPanelRightElements {
  display: flex;
  justify-content: flex-end;
}

div.minimizedButtonsPanel {
  display: flex;

  margin-left: 10px;
  margin-right: 10px;
  margin-top: 10px;
}

div.minimizedButtonsPanel button {
  @extend .noBordersBtn;

  -webkit-app-region: no-drag;
  z-index: 101;
  cursor: pointer;

  padding: 0px;
  margin-left: 6px;
  margin-right: 6px;
}

div.minimizedButtonsPanel img {
  height: 18px;
}
</style>
