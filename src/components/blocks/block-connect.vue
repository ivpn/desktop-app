<template>
  <div class="main">
    <div align="left">
      <div class="small_text">Your status is</div>
      <div class="large_text">{{ protectedText }}</div>
    </div>

    <div class="buttons">
      <div class="buttonWithPopup" style="margin-right:17px;">
        <transition name="fade">
          <button
            class="settingsBtn"
            v-if="isCanPause"
            v-on:click="onPauseMenu"
          >
            <img src="@/assets/pause.svg" />
          </button>

          <button
            class="settingsBtnResume"
            v-else-if="isCanResume"
            v-on:click="onPauseResume"
          >
            <img src="@/assets/resume.svg" style="margin-left: 2px" />
          </button>
        </transition>

        <!-- Popup -->
        <div
          class="popup"
          v-bind:class="{
            popupMin: isMinimizedUI
          }"
        >
          <div
            ref="pausePopup"
            class="popuptext"
            v-bind:class="{
              show: isCanShowPauseMenu,
              popuptextMin: isMinimizedUI
            }"
          >
            <div class="popup_menu_block">
              <button v-on:click="onPauseMenuItem(5 * 60)">
                Pause for 5 min
              </button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button v-on:click="onPauseMenuItem(30 * 60)">
                Pause for 30 min
              </button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button v-on:click="onPauseMenuItem(1 * 60 * 60)">
                Pause for 1 hour
              </button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button v-on:click="onPauseMenuItem(3 * 60 * 60)">
                Pause for 3 hours
              </button>
            </div>
          </div>
        </div>
      </div>

      <div style="min-width: 50px; margin-left:auto; margin-right:0;">
        <SwitchProgress
          :onChecked="onChecked"
          :isChecked="isChecked"
          :isProgress="isProgress"
        />
      </div>
    </div>
  </div>
</template>

<script>
import SwitchProgress from "@/components/controls/control-switch.vue";
import { PauseStateEnum } from "@/store/types";

export default {
  components: {
    SwitchProgress
  },
  props: [
    "onChecked",
    "onPauseResume",
    "pauseState",
    "isChecked",
    "isProgress"
  ],
  data: () => ({
    isPauseMenuAllowed: false
  }),
  created: function() {
    let self = this;
    window.addEventListener("click", function(e) {
      // close dropdown when clicked outside
      if (!self.$el.contains(e.target)) {
        self.isPauseMenuAllowed = false;
      }
    });
  },
  computed: {
    isMinimizedUI: function() {
      return this.$store.state.settings.minimizedUI;
    },
    protectedText: function() {
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Paused)
        return "paused";
      if (this.isChecked !== true || this.isCanResume) return "disconnected";
      return "connected";
    },
    isConnected: function() {
      return this.$store.getters["vpnState/isConnected"];
    },
    isCanPause: function() {
      if (!this.isConnected) return false;
      if (this.isProgress === true) return false;
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Resumed)
        return true;
      return false;
    },
    isCanResume: function() {
      if (this.isCanPause) return false;
      if (!this.isConnected) return false;
      if (this.isProgress === true) return false;
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Paused)
        return true;
      return false;
    },
    isCanShowPauseMenu: function() {
      return this.isCanPause && this.isPauseMenuAllowed;
    }
  },
  methods: {
    onPauseMenu() {
      if (this.isPauseMenuAllowed != true) this.onPauseResume(null);
      this.isPauseMenuAllowed = !this.isPauseMenuAllowed;
    },
    onPauseMenuItem(seconds) {
      this.isPauseMenuAllowed = false;
      if (this.onPauseResume != null) this.onPauseResume(seconds);
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/popup";
$shadow: 0px 3px 1px rgba(0, 0, 0, 0.06), 0px 3px 8px rgba(0, 0, 0, 0.15);

.main {
  @extend .left_panel_block;
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 97px;
}

.buttons {
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 92px;
}

.large_text {
  font-style: normal;
  font-weight: 600;
  letter-spacing: -0.3px;
  font-size: 24px;
  line-height: 29px;
}

.small_text {
  font-size: 14px;
  line-height: 17px;
  letter-spacing: -0.3px;
  color: $base-text-color-details;
}

.settingsBtn {
  float: right;

  width: 32px;
  height: 32px;

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

.settingsBtn:hover {
  background-color: #f0f0f0;
}

.settingsBtnResume {
  @extend .settingsBtn;
  background-color: #449cf8;
}

.settingsBtnResume:hover {
  background-color: #3377ff;
}
</style>
