<template>
  <div>
    <div class="hopButtons">
      <div />
      <button
        class="hopButton"
        v-bind:class="{
          hopButtonActive: !this.$store.state.settings.isMultiHop,
        }"
        v-on:click="ChangeHop(false)"
      >
        SINGLE-HOP
      </button>

      <div />

      <button
        class="hopButton"
        v-bind:class="{
          hopButtonActive: this.$store.state.settings.isMultiHop,
        }"
        v-on:click="ChangeHop(true)"
      >
        MULTI-HOP
      </button>

      <div />
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  computed: {},

  methods: {
    async ChangeHop(isMultihop) {
      if (this.$store.state.settings.isMultiHop === isMultihop) return;

      this.$store.dispatch(
        `settings/isMultiHop`,
        !this.$store.state.settings.isMultiHop
      );

      if (
        this.$store.getters["vpnState/isConnected"] ||
        this.$store.getters["vpnState/isConnecting"]
      ) {
        // Re-connect
        try {
          await sender.Connect();
        } catch (e) {
          console.error(e);
          sender.showMessageBoxSync({
            type: "error",
            buttons: ["OK"],
            message: `Failed to connect: ` + e,
          });
        }
      }
    },

    showServersList(isExitServer) {
      this.onShowServersPressed(isExitServer);
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss"></style>
