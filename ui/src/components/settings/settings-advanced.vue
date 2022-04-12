<template>
  <div>
    <div class="settingsTitle">ADVANCED SETTINGS</div>
    <div class="settingsBoldFont">Enhanced App Protection</div>
    <div class="settingsParamProps settingsDescription">
      Enhanced App Protection (EAP) implements an additional authentication
      factor between the IVPN app (UI) and the daemon that manages the VPN
      tunnel. This prevents a malicious app from being able to manipulate the
      VPN tunnel without the users permission. You will be required to manually
      enter the shared secret when starting the app..
    </div>

    <div class="flexRow settingsParamProps">
      <div class="settingsDefaultTextColor paramName">Status:</div>
      <div style="font-weight: 500">
        <div v-if="IsPmEnabled" class="flexRow">
          <div style="color: #64ad07; min-width: 80px">Enabled</div>
          <button
            class="settingsButton paramBlock"
            style="height: 24px"
            v-on:click="onChangeState()"
          >
            Disable
          </button>
        </div>
        <div v-else class="flexRow">
          <div style="min-width: 80px">Disabled</div>
          <button
            class="settingsButton paramBlock"
            style="height: 24px"
            v-on:click="onChangeState()"
          >
            Enable
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { IpcModalDialogType, IpcOwnerWindowType } from "@/ipc/types.js";
import { Platform, PlatformEnum } from "@/platform/platform";

const sender = window.ipcSender;

export default {
  computed: {
    IsPmEnabled: function () {
      return this.$store.state.paranoidModeStatus.IsEnabled;
    },
  },

  methods: {
    async onChangeState() {
      let cfg = {
        width: 400,
        height: 170,
      };
      let dlgType = IpcModalDialogType.EnableEAP;
      if (Platform() === PlatformEnum.Windows) cfg.height = 142;

      if (this.IsPmEnabled) {
        dlgType = IpcModalDialogType.DisableEAP;

        cfg.height = 150;
        if (Platform() === PlatformEnum.Windows) cfg.height = 122;
      }
      await sender.showModalDialog(
        dlgType,
        IpcOwnerWindowType.SettingsWindow,
        cfg
      );
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

div.paramName {
  min-width: 150px;
  max-width: 150px;
}
</style>
