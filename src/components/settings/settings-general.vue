<template>
  <div>
    <div class="settingsTitle">GENERAL SETTINGS</div>
    <div class="settingsBoldFont">
      Autoconnect:
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="connectOnLaunch"
        v-model="autoConnectOnLaunch"
      />
      <label class="defColor" for="connectOnLaunch">On launch</label>
    </div>

    <div class="settingsBoldFont">
      On exit:
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="quitWithoutConfirmation"
        v-model="quitWithoutConfirmation"
      />
      <label class="defColor" for="quitWithoutConfirmation"
        >Quit without confirmation when closing application</label
      >
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="disconnect"
        v-model="disconnectOnQuit"
        :disabled="quitWithoutConfirmation === false"
      />
      <label class="defColor" for="disconnect"
        >Disconnect when closing application</label
      >
    </div>

    <div class="settingsBoldFont">
      Diagnostics:
    </div>
    <div class="param">
      <input type="checkbox" id="logging" v-model="logging" />
      <label class="defColor" for="logging">Create log files</label>
    </div>
  </div>
</template>

<script>
import sender from "@/ipc/renderer-sender";

export default {
  data: function() {
    return {};
  },
  methods: {},
  computed: {
    autoConnectOnLaunch: {
      get() {
        return this.$store.state.settings.autoConnectOnLaunch;
      },
      set(value) {
        this.$store.dispatch("settings/autoConnectOnLaunch", value);
      }
    },
    disconnectOnQuit: {
      get() {
        return this.$store.state.settings.disconnectOnQuit;
      },
      set(value) {
        this.$store.dispatch("settings/disconnectOnQuit", value);
      }
    },
    quitWithoutConfirmation: {
      get() {
        return this.$store.state.settings.quitWithoutConfirmation;
      },
      set(value) {
        this.$store.dispatch("settings/quitWithoutConfirmation", value);
      }
    },
    logging: {
      get() {
        return this.$store.state.settings.logging;
      },
      set(value) {
        this.$store.dispatch("settings/logging", value);
        sender.SetLogging();
      }
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

input:disabled {
  opacity: 0.5;
}
input:disabled + label {
  opacity: 0.5;
}

label {
  margin-left: 1px;
}
</style>
