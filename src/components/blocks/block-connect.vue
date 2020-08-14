<template>
  <div class="main">
    <div align="left">
      <div class="small_text">Your status is</div>
      <div class="large_text">{{ protectedText }}</div>
    </div>

    <div class="buttons">
      <div>
        <transition name="fade">
          <button
            class="settingsBtn"
            style="margin-right:17px;"
            v-if="isCanPause"
            v-on:click="onPauseMenu"
          >
            <img src="@/assets/pause.svg" />
          </button>
        </transition>

        <transition name="fade">
          <button
            class="settingsBtnResume"
            style="margin-right:17px;"
            v-if="isCanResume"
            v-on:click="onPauseResume"
          >
            <img src="@/assets/resume.svg" style="margin-left: 2px" />
          </button>
        </transition>

        <!-- Popup -->
        <div class="popup" style="margin-top: 38px; margin-left: 16px;">
          <div
            ref="pausePopup"
            class="popuptext"
            v-bind:class="{
              show: isCanShowPauseMenu
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
    protectedText: function() {
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Paused)
        return "paused";
      if (this.isChecked !== true || this.isCanResume) return "disconnected";
      return "connected";
    },
    isCanPause: function() {
      if (process.platform === "linux") return false;
      if (this.isChecked === false || this.isProgress === true) return false;
      if (this.$store.state.vpnState.pauseState !== PauseStateEnum.Resumed)
        return false;
      return true;
    },
    isCanResume: function() {
      if (process.platform === "linux") return false;
      if (this.isCanPause) return false;
      if (this.isChecked === false || this.isProgress === true) return false;
      if (this.$store.state.vpnState.pauseState !== PauseStateEnum.Paused)
        return false;
      return true;
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

// ============== POPUP =================
$popup-background: white; //green; //white; // #ffffff; // #f1f1f1;

.popup {
  position: absolute;
  z-index: 4;
  user-select: none;
}

// The actual popup

.popup .popuptext {
  visibility: hidden;
  background-color: $popup-background;
  text-align: center;
  border-radius: 14px;
  position: absolute;

  min-width: 216px;
  max-width: 216px;

  margin-left: -108px; // 216/2
  margin-top: 18px;

  box-shadow: 0px 0px 34px rgba(37, 51, 72, 0.15);
}

// Popup arrow
.popup .popuptext::after {
  content: "";
  position: absolute;
  top: -24px;
  margin-left: -12px;
  margin-top: 12px;
  border-width: 12px;
  border-style: solid;
  border-color: $popup-background transparent transparent $popup-background;
  transform: rotate(45deg);
}

// Toggle this class - hide and show the popup
.popup .show {
  visibility: visible;
  animation: fadeIn 0.5s;
}

.popup_menu_block {
  min-height: 41px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.popup_menu_block > * {
  // font
  font-size: 13px;
  line-height: 16px;
  text-align: center;
  letter-spacing: -0.078px;
  color: rgba(42, 57, 75, 0.85);
}

.popup_menu_block > button {
  @extend .noBordersBtn;
}

.popup_dividing_line {
  background: #e9e9e9;
  height: 1px;
  border: 0px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>
