<template>
  <div id="flexview">
    <div class="flexColumn">
      <div class="leftPanelTopSpace">
        <transition name="smooth-display">
          <div
            v-if="isMinimizedButtonsVisible"
            class="minimizedButtonsPanel leftPanelTopMinimizedButtonsPanel"
            v-bind:class="{
              minimizedButtonsPanelRightElements: isWindowHasFrame,
            }"
          >
            <button v-on:click="onAccountSettings()" title="Account settings">
              <img src="@/assets/user.svg" />
            </button>

            <button v-on:click="onSettings()" title="Settings">
              <img src="@/assets/settings.svg" />
            </button>

            <button v-on:click="onMaximize(true)" title="Show map">
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
            :onFirewallSettings="onFirewallSettings"
            :onAntiTrackerSettings="onAntitrackerSettings"
            :onDefaultView="onDefaultLeftView"
            id="left"
          ></component>
        </transition>
      </div>
    </div>
    <div id="right" v-if="!isMinimizedUI">
      <transition name="fade" appear>
        <TheMap
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
const sender = window.ipcSender;

import { DaemonConnectionType } from "@/store/types";
import { IsWindowHasFrame } from "@/platform/platform";
import Init from "@/components/Component-Init.vue";
import Login from "@/components/Component-Login.vue";
import Control from "@/components/Component-Control.vue";
import TheMap from "@/components/Component-Map.vue";
import ParanoidModePassword from "@/components/ParanoidModePassword.vue";

export default {
  components: {
    Init,
    Login,
    Control,
    TheMap,
    ParanoidModePassword,
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
      if (this.$store.state.uiState.isParanoidModePasswordView === true)
        return ParanoidModePassword;
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
    onFirewallSettings: function () {
      sender.ShowFirewallSettings();
    },
    onAntitrackerSettings: function () {
      sender.ShowAntitrackerSettings();
    },
    onDefaultLeftView: function (isDefaultView) {
      this.isCanShowMinimizedButtons = isDefaultView;
    },
    onMaximize: function (isMaximize) {
      this.$store.dispatch("settings/minimizedUI", !isMaximize);
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
