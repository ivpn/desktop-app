<template>
  <div>
    <div class="settingsTitle">ADVANCED SETTINGS</div>
    <div class="settingsBoldFont">Paranoid Mode</div>
    <div class="settingsParamProps settingsDescription">
      When Paranoid Mode is enabled, you will be asked for a secret password
      before using any IVPN functionality.
    </div>
    <div class="flexRow settingsParamProps">
      <div class="settingsDefaultTextColor paramName">Status:</div>
      <div style="font-weight: 500">
        <div v-if="IsPmEnabled" style="color: #ad6407">Enabled</div>
        <div v-else>Disabled</div>
      </div>
    </div>

    <!--Enable PM (new password)-->
    <div v-if="!IsPmEnabled" class="settingsParamProps">
      <div class="flexRow">
        <div class="settingsDefaultTextColor paramName">Password:</div>
        <input
          type="password"
          style="flex-grow: 1"
          class="settingsTextInput"
          placeholder="Define Paranoid Mode password"
          v-model="newPass"
          v-on:keyup.enter="onApplyNewPassword()"
        />
      </div>
      <div class="flexRow">
        <div class="settingsDefaultTextColor paramName">Repeat password:</div>
        <input
          type="password"
          style="flex-grow: 1"
          class="settingsTextInput"
          placeholder="Repeat Paranoid Mode password"
          v-model="newPassConfirm"
          v-on:keyup.enter="onApplyNewPassword()"
        />
      </div>

      <div class="flexRow" style="margin-top: 10px">
        <div style="flex-grow: 1"></div>
        <button
          class="settingsButton paramBlock"
          style="height: 24px"
          v-on:click="onApplyNewPassword()"
        >
          Enable Paranoid Mode
        </button>
      </div>
    </div>

    <!--Disable PM-->
    <div v-if="IsPmEnabled" class="settingsParamProps">
      <div class="flexRow">
        <div class="settingsDefaultTextColor paramName">Password:</div>
        <input
          type="password"
          style="flex-grow: 1"
          class="settingsTextInput"
          placeholder="Enter actual password"
          v-model="oldPass"
          v-on:keyup.enter="onApplyPasswordReset()"
        />
      </div>

      <div class="flexRow" style="margin-top: 10px">
        <div style="flex-grow: 1"></div>
        <button
          class="settingsButton paramBlock"
          style="height: 24px"
          v-on:click="onApplyPasswordReset()"
        >
          Disable Paranoid Mode
        </button>
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  data: function () {
    return {
      oldPass: "",
      newPass: "",
      newPassConfirm: "",
    };
  },

  computed: {
    IsPmEnabled: function () {
      return this.$store.state.paranoidModeStatus.IsEnabled;
    },
  },

  methods: {
    resetData() {
      this.oldPass = "";
      this.newPass = "";
      this.newPassConfirm = "";
    },
    async onApplyNewPassword() {
      if (!this.newPass) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Password not defined",
          detail: `Please, define password`,
        });
        return;
      }
      if (this.newPass != this.newPassConfirm) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Passwords does not match",
          detail: `Please, define password`,
        });
        return;
      }

      if (this.newPass != this.newPass.trim()) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Bad password",
          detail: `Please, avoid using space symbols`,
        });
        return;
      }

      try {
        await sender.setParanoidModePassword(this.newPass);
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to enable Paranoid Mode: `,
          detail: e,
        });
      }

      this.resetData();
    },

    async onApplyPasswordReset() {
      if (!this.oldPass) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Password not defined",
          detail: `Please, define actual password`,
        });
        return;
      }

      try {
        await sender.setParanoidModePassword("", this.oldPass);
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to disable Paranoid Mode`,
          detail: e,
        });
      }

      this.resetData();
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
