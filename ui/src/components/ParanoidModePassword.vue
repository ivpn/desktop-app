<template>
  <div class="flexColumn">
    <div class="flexColumn" style="padding: 15px">
      <div class="main" style="padding-top: 15px">
        <div
          style="text-align: center; margin-left: 0px; margin-right: 0px"
          class="large_text"
        >
          Enhanced App Authentication
        </div>

        <div class="small_text flexRow" style="">
          Please enter EAA password:
        </div>
        <input
          type="password"
          class="styled"
          style="width: calc(100% - 30px); text-align: center"
          placeholder=""
          v-model="pmPassword"
          ref="passwordField"
          v-on:keyup.enter="setPassword()"
        />
        <button class="btn" v-on:click="setPassword()">OK</button>
      </div>

      <div class="small_text">
        Please note: You can disable Enhanced App Authentication mode in the
        advanced settings.
      </div>
      <!--
      <button
        class="noBordersTextBtn settingsLinkText"
        v-on:click="visitWebsite"
      >
        www.ivpn.net
      </button>-->
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  components: {},
  data: function () {
    return {
      pmPassword: "",
    };
  },
  mounted() {
    if (this.$refs.passwordField) this.$refs.passwordField.focus();
  },
  methods: {
    //
    async setPassword() {
      try {
        let pass = this.pmPassword;
        if (!pass) {
          await sender.showMessageBoxSync({
            type: "error",
            buttons: ["OK"],
            message: `Enhanced App Authentication`,
            detail: "The password is incorrect. Please try again.",
          });
          return;
        }

        if (pass != pass.trim()) {
          await sender.showMessageBoxSync({
            type: "warning",
            buttons: ["OK"],
            message: `Enhanced App Authentication`,
            detail: "The password is incorrect. Please try again.",
          });
          return;
        }

        await sender.setLocalParanoidModePassword(this.pmPassword);
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Enhanced App Authentication`,
          detail: e,
        });
      }
    },
    visitWebsite() {
      sender.shellOpenExternal(`https://www.ivpn.net`);
    },
  },
  computed: {},
  watch: {},
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.main {
  margin-top: -100px;
  height: 100%;

  display: flex;
  flex-flow: column;
  justify-content: center;
  align-items: center;
}

.large_text {
  margin: 12px;
  font-weight: 600;
  font-size: 18px;
  line-height: 120%;
}

.small_text {
  margin: 12px;
  margin-top: 0px;

  font-size: 13px;
  line-height: 17px;
  letter-spacing: -0.208px;

  color: #98a5b3;
}

.btn {
  margin: 30px 0 0 0;
  width: 90%;
  height: 28px;
  background: #ffffff;
  border-radius: 10px;
  border: 1px solid #7d91a5;

  font-size: 15px;
  line-height: 20px;
  text-align: center;
  letter-spacing: -0.4px;
  color: #6d849a;

  cursor: pointer;
}
</style>
