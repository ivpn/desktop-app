<template>
  <div>
    <div class="settingsTitle">ADVANCED SETTINGS</div>
    <div class="settingsBoldFont">Enhanced App Authentication</div>
    <div class="settingsParamProps settingsDescription">
      Enhanced App Authentication (EAA) implements an additional authentication
      factor between the IVPN app (UI) and the daemon that manages the VPN
      tunnel. This prevents a malicious app from being able to manipulate the
      VPN tunnel without the users permission. You will be required to manually
      enter the EAA password when starting the app.
    </div>

    <div class="flexRowAlignTop settingsParamProps">
      <div class="settingsDefaultTextColor paramName">Status:</div>
      <div style="font-weight: 500">
        <div v-if="IsPmEnabled">
          <div style="color: #64ad07; min-width: 80px">Enabled</div>
          <button
            class="settingsButton paramBlock"
            style="height: 24px; margin-top: 6px"
            v-on:click="onChangeState()"
          >
            Disable
          </button>
        </div>
        <div v-else>
          <div style="min-width: 80px">Disabled</div>
          <button
            class="settingsButton paramBlock"
            style="height: 24px; margin-top: 6px"
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

      if (!this.IsPmEnabled) {
        if (
          true ===
          this.$store.state.settings.daemonSettings.IsAutoconnectOnLaunch
        ) {
          sender.showMessageBoxSync({
            type: "info",
            buttons: ["OK"],
            message: `Enhanced App Authentication`,
            detail:
              "EAA cannot be enabled whilst 'Autoconnect on application launch' is enabled.",
          });
          return;
        }

        if (true === this.$store.state.settings.wifi.trustedNetworksControl) {
          let ret = await sender.showMessageBoxSync(
            {
              type: "warning",
              message: `Enhanced App Authentication`,
              detail:
                "Warning: On application start Trusted WiFi will be disabled until the EAA password is entered",
              buttons: ["Enable", "Cancel"],
            },
            true
          );
          if (ret == 1) return; // cancel
        }
      }

      let dlgType = IpcModalDialogType.EnableEAA;
      if (Platform() !== PlatformEnum.macOS) cfg.height = 142;

      if (this.IsPmEnabled) {
        dlgType = IpcModalDialogType.DisableEAA;

        cfg.height = 150;
        if (Platform() !== PlatformEnum.macOS) cfg.height = 122;
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
