<template>
  <div class="defaultMainDiv">
    <div class="settingsBoldFont" style="margin-top: 0px; margin-bottom: 12px">
      Please enter the shared secret to disable EAP:
    </div>
    <div class="flexRow">
      <input
        ref="passwordField"
        type="password"
        style="flex-grow: 1"
        class="settingsTextInput"
        placeholder=""
        v-model="oldPass"
        v-on:keyup.enter="onApplyPasswordReset()"
      />
    </div>

    <div class="flexRow" style="margin-top: 10px">
      <div style="flex-grow: 1"></div>
      <div class="flexRow">
        <button
          class="master"
          style="height: 28px; min-width: 100px"
          v-on:click="onApplyPasswordReset()"
        >
          Disable
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
  mounted() {
    if (this.$refs.passwordField) this.$refs.passwordField.focus();
  },
  data: function () {
    return {
      oldPass: "",
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
        window.close();
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to disable Paranoid Mode`,
          detail: e,
        });
      }
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";
</style>
