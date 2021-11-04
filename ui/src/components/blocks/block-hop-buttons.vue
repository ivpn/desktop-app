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

import { VpnStateEnum } from "@/store/types";

export default {
  computed: {},

  methods: {
    ChangeHop(isMultihop) {
      if (this.$store.state.settings.isMultiHop === isMultihop) return;
      if (
        this.$store.state.vpnState.connectionState !== VpnStateEnum.DISCONNECTED
      ) {
        sender.showMessageBox({
          type: "info",
          buttons: ["OK"],
          message: "You are now connected to IVPN",
          detail:
            "You can change Multi-Hop settings only when IVPN is disconnected.",
        });
        return;
      }

      this.$store.dispatch(
        `settings/isMultiHop`,
        !this.$store.state.settings.isMultiHop
      );
    },
    showServersList(isExitServer) {
      this.onShowServersPressed(isExitServer);
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss"></style>
