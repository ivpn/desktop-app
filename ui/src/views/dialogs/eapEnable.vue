<template>
  <div class="defaultMainDiv">
    <div class="settingsBoldFont" style="margin-top: 0px; margin-bottom: 12px">
      Please define shared secret to enable EAP
    </div>
    <div class="flexRow">
      <div class="settingsDefaultTextColor paramName">Secret:</div>
      <input
        type="password"
        style="flex-grow: 1"
        class="settingsTextInput"
        placeholder=""
        v-model="newPass"
        v-on:keyup.enter="onApplyNewPassword()"
      />
    </div>
    <div class="flexRow">
      <div class="settingsDefaultTextColor paramName">Re-enter secret:</div>
      <input
        type="password"
        style="flex-grow: 1"
        class="settingsTextInput"
        placeholder=""
        v-model="newPassConfirm"
        v-on:keyup.enter="onApplyNewPassword()"
      />
    </div>

    <div class="flexRow" style="margin-top: 10px">
      <div style="flex-grow: 1"></div>
      <div class="flexRow">
        <button
          class="master"
          style="height: 28px; min-width: 100px"
          v-on:click="onApplyNewPassword()"
        >
          Enable
        </button>
        <button
          class="slave"
          style="height: 28px; min-width: 100px; margin-left: 12px"
          v-on:click="onCancel()"
        >
          Cancel
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
      newPass: "",
      newPassConfirm: "",
    };
  },
  created() {
    window.onkeydown = function (event) {
      console.log(onkeydown);
      if (event.keyCode == 27) {
        window.close();
      }
    };
  },
  computed: {
    IsPmEnabled: function () {
      return this.$store.state.paranoidModeStatus.IsEnabled;
    },
  },

  methods: {
    onCancel() {
      window.close();
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
        window.close();
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to enable Paranoid Mode`,
          detail: e,
        });
      }
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

div.paramName {
  min-width: 100px;
  max-width: 100px;
}
</style>
