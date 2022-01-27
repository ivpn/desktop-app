<template>
  <div class="main">
    <div align="left">
      <div class="small_text">Your status is</div>
      <div>
        <div class="large_text">
          {{ protectedText }}
        </div>

        <div
          v-if="isCanResume"
          class="buttonWithPopup"
          style="position: absolute"
        >
          <button
            v-click-outside="onPauseMenuClickOutside"
            class="noBordersBtn"
            style="padding: 0"
            @click="onAddPauseTimeMenu"
          >
            <div class="small_text" align="left" style="min-width: 80px">
              {{ pauseTimeLeftText }}
            </div>
          </button>

          <!-- Popup -->
          <div
            style="background: red; margin-top: -5px"
            class="popup"
            :class="{
              popupMinShiftedRight: true,
            }"
          >
            <div
              class="popuptext"
              :class="{
                show: isPauseExtendMenuShow,
                popuptextMinShiftedRight: true,
              }"
            >
              <div class="popup_menu_block">
                <button @click="onPauseMenuItem(null)">Resume now</button>
              </div>
              <div class="popup_dividing_line" />
              <div class="popup_menu_block">
                <button @click="onPauseMenuItem(5 * 60)">
                  Resume in 5 min
                </button>
              </div>
              <div class="popup_dividing_line" />
              <div class="popup_menu_block">
                <button @click="onPauseMenuItem(30 * 60)">
                  Resume in 30 min
                </button>
              </div>
              <div class="popup_dividing_line" />
              <div class="popup_menu_block">
                <button @click="onPauseMenuItem(1 * 60 * 60)">
                  Resume in 1 hour
                </button>
              </div>
              <div class="popup_dividing_line" />
              <div class="popup_menu_block">
                <button @click="onPauseMenuItem(3 * 60 * 60)">
                  Resume in 3 hours
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="buttons">
      <div class="buttonWithPopup" style="margin-right: 17px">
        <transition name="fade">
          <button
            v-if="isCanPause"
            v-click-outside="onPauseMenuClickOutside"
            class="settingsBtn"
            style="background: var(--background-color)"
            @click="onPauseMenu"
          >
            <imgPause />
          </button>

          <button
            v-else-if="isCanResume"
            class="settingsBtnResume"
            @click="onPauseResume(null)"
          >
            <img src="@/assets/resume.svg" style="margin-left: 2px" />
          </button>
        </transition>

        <!-- Popup -->
        <div
          class="popup"
          :class="{
            popupMin: isMinimizedUI,
          }"
        >
          <div
            class="popuptext"
            :class="{
              show: isCanShowPauseMenu,
              popuptextMin: isMinimizedUI,
            }"
          >
            <div class="popup_menu_block">
              <button @click="onPauseMenuItem(5 * 60)">Pause for 5 min</button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button @click="onPauseMenuItem(30 * 60)">
                Pause for 30 min
              </button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button @click="onPauseMenuItem(1 * 60 * 60)">
                Pause for 1 hour
              </button>
            </div>
            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <button @click="onPauseMenuItem(3 * 60 * 60)">
                Pause for 3 hours
              </button>
            </div>
          </div>
        </div>
      </div>

      <div style="min-width: 50px; margin-left: auto; margin-right: 0">
        <SwitchProgress
          :class="{ lowOpacity: isCanResume }"
          :on-checked="onChecked"
          :is-checked="isChecked"
          :is-progress="isProgress"
        />
      </div>
    </div>
  </div>
</template>

<script>
import SwitchProgress from "@/components/controls/control-switch.vue";
import imgPause from "@/components/images/pause.vue";
import { PauseStateEnum } from "@/store/types";
import { GetTimeLeftText } from "@/helpers/renderer";
import ClickOutside from "vue-click-outside";

export default {
  directives: {
    ClickOutside,
  },
  components: {
    SwitchProgress,
    imgPause,
  },
  props: [
    "onChecked",
    "onPauseResume",
    "pauseState",
    "isChecked",
    "isProgress",
  ],
  data: () => ({
    isPauseMenuAllowed: false,
    isPauseExtendMenuShow: false,
    pauseTimeUpdateTimer: null,
    pauseTimeLeftText: "",
  }),
  computed: {
    isMinimizedUI: function () {
      return this.$store.state.settings.minimizedUI;
    },
    protectedText: function () {
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Paused)
        return "paused";
      if (this.isChecked !== true || this.isCanResume) return "disconnected";
      return "connected";
    },
    isConnected: function () {
      return this.$store.getters["vpnState/isConnected"];
    },
    pauseConnectionTill: function () {
      return this.$store.state.uiState.pauseConnectionTill;
    },
    isPaused: function () {
      if (this.$store.state.vpnState.pauseState !== PauseStateEnum.Paused)
        return false;
      return this.pauseConnectionTill != null;
    },
    isCanPause: function () {
      if (!this.isConnected) return false;
      if (this.isProgress === true) return false;

      var connInfo = this.$store.state.vpnState.connectionInfo;
      if (connInfo === null || connInfo.IsCanPause === false) return false;
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Resumed)
        return true;
      return false;
    },
    isCanResume: function () {
      if (this.isCanPause) return false;
      if (!this.isConnected) return false;
      if (this.isProgress === true) return false;
      if (this.$store.state.vpnState.pauseState === PauseStateEnum.Paused)
        return true;
      return false;
    },
    isCanShowPauseMenu: function () {
      return this.isCanPause && this.isPauseMenuAllowed;
    },
  },
  watch: {
    isPaused() {
      this.startPauseTimer();
    },
  },
  mounted() {
    this.startPauseTimer();
  },
  created: function () {
    let self = this;
    window.addEventListener("click", function (e) {
      // close dropdown when clicked outside
      if (!self.$el.contains(e.target)) {
        self.isPauseMenuAllowed = false;
      }
    });
  },
  methods: {
    onPauseMenuClickOutside() {
      this.isPauseExtendMenuShow = false;
      this.isPauseMenuAllowed = false;
    },
    onPauseMenu() {
      if (this.isPauseMenuAllowed != true) this.onPauseResume(null);
      this.isPauseMenuAllowed = !this.isPauseMenuAllowed;
    },
    onAddPauseTimeMenu() {
      if (this.isCanResume != true) this.isPauseExtendMenuShow = false;
      else this.isPauseExtendMenuShow = !this.isPauseExtendMenuShow;
    },
    onPauseMenuItem(seconds) {
      this.isPauseMenuAllowed = false;
      this.isPauseExtendMenuShow = false;
      if (this.onPauseResume != null) this.onPauseResume(seconds);
    },
    startPauseTimer() {
      if (this.pauseTimeUpdateTimer) return;
      if (!this.$store.state.uiState.pauseConnectionTill) return;

      this.pauseTimeUpdateTimer = setInterval(() => {
        this.pauseTimeLeftText = GetTimeLeftText(
          this.$store.state.uiState.pauseConnectionTill
        );

        if (this.$store.state.vpnState.pauseState !== PauseStateEnum.Paused) {
          clearInterval(this.pauseTimeUpdateTimer);
          this.pauseTimeUpdateTimer = null;
        }
      }, 1000);

      this.pauseTimeLeftText = GetTimeLeftText(
        this.$store.state.uiState.pauseConnectionTill
      );
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/popup";
$shadow: 0px 3px 1px rgba(0, 0, 0, 0.06),
  0px 3px 8px rgba(0, 0, 0, var(--shadow-opacity-koef));

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

.lowOpacity {
  opacity: 0.5;
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
  color: var(--text-color-details);
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
