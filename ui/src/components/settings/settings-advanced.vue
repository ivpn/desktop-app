<template>
  <div>
    <ComponentDialog ref="eaaEnableDlg" noCloseButtons>
      <ComponentEaaEnable style="margin: 0px" :onClose="onCloseEaaDlg" />
    </ComponentDialog>

    <ComponentDialog ref="eaaDisableDlg" noCloseButtons>
      <ComponentEaaDisable style="margin: 0px" :onClose="onCloseEaaDlg" />
    </ComponentDialog>

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
import ComponentDialog from "@/components/component-dialog.vue";
import ComponentEaaDisable from "@/views/dialogs/eaaDisable.vue";
import ComponentEaaEnable from "@/views/dialogs/eaaEnable.vue";

const sender = window.ipcSender;

export default {
  components: {
    ComponentDialog,
    ComponentEaaDisable,
    ComponentEaaEnable,
  },

  computed: {
    IsPmEnabled: function () {
      return this.$store.state.paranoidModeStatus.IsEnabled;
    },
  },

  methods: {
    onCloseEaaDlg() {
      try {
        this.$refs.eaaDisableDlg.close();
        this.$refs.eaaEnableDlg.close();
      } catch (e) {
        console.error(e);
      }
    },
    async onChangeState() {
      if (!this.IsPmEnabled) {
        let warningMessage = "";

        if (
          true ===
          this.$store.state.settings.daemonSettings.IsAutoconnectOnLaunch
        )
          warningMessage =
            "On application start 'Autoconnect on application launch' will not be applied until the EAA password is entered.";

        if (true === this.$store.state.settings?.wifi?.trustedNetworksControl) {
          if (warningMessage) warningMessage += "\n\n";
          warningMessage +=
            "On application start Trusted WiFi will be disabled until the EAA password is entered.";
        }

        if (warningMessage) {
          let ret = await sender.showMessageBoxSync({
            type: "warning",
            message: `Enhanced App Authentication`,
            detail: "Warning!\n\n" + warningMessage,
            buttons: ["Enable", "Cancel"],
          });
          if (ret == 1) return; // cancel
        }
      }

      try {
        if (this.IsPmEnabled) this.$refs.eaaDisableDlg.showModal();
        else this.$refs.eaaEnableDlg.showModal();
      } catch (e) {
        console.error(e);
      }
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
