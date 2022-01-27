<template>
  <div>
    <div class="settingsTitle">FIREWALL SETTINGS</div>

    <div class="settingsBoldFont">Non-VPN traffic blocking:</div>

    <div>
      <div class="settingsRadioBtn">
        <input
          id="onDemand"
          ref="radioFWOnDemand"
          type="radio"
          name="firewall"
          value="false"
          @click="onPersistentFWChange(false)"
        />
        <label class="defColor" for="onDemand">On-demand</label>
      </div>
      <div class="fwDescription">
        When this option is enabled the IVPN Firewall can be either manually
        activated or automatically activated when the VPN connection is
        established - see On-demand Firewall options below
      </div>

      <div>
        <div class="settingsRadioBtn">
          <input
            id="alwaysOn"
            ref="radioFWPersistent"
            type="radio"
            name="firewall"
            value="true"
            @click="onPersistentFWChange(true)"
          />
          <label class="defColor" for="alwaysOn">Always-on firewall</label>
        </div>
        <div class="fwDescription">
          When the option is enabled the IVPN Firewall is started during system
          boot time before any other process. IVPN Firewall will always be
          active even when IVPN Client is not running
        </div>
      </div>
    </div>

    <div class="param">
      <input
        id="firewallAllowApiServers"
        v-model="firewallAllowApiServers"
        type="checkbox"
      />
      <label class="defColor" for="firewallAllowApiServers"
        >Allow access to IVPN servers when Firewall is enabled</label
      >
    </div>

    <!-- On-demand Firewall -->
    <div class="settingsBoldFont">On-demand Firewall:</div>

    <div class="param">
      <input
        id="firewallActivateOnConnect"
        v-model="firewallActivateOnConnect"
        type="checkbox"
        :disabled="IsPersistent === true"
      />
      <label class="defColor" for="firewallActivateOnConnect"
        >Activate IVPN Firewall on connect to VPN</label
      >
    </div>
    <div class="param">
      <input
        id="firewallDeactivateOnDisconnect"
        v-model="firewallDeactivateOnDisconnect"
        type="checkbox"
        :disabled="IsPersistent === true"
      />
      <label class="defColor" for="firewallDeactivateOnDisconnect"
        >Deactivate IVPN Firewall on disconnect from VPN</label
      >
    </div>

    <!-- LAN settings -->
    <div class="settingsBoldFont">LAN settings:</div>
    <div class="param">
      <input id="firewallAllowLan" v-model="firewallAllowLan" type="checkbox" />
      <label class="defColor" for="firewallAllowLan"
        >Allow LAN traffic when IVPN Firewall is enabled</label
      >
    </div>
    <div class="param">
      <input
        id="firewallAllowMulticast"
        v-model="firewallAllowMulticast"
        type="checkbox"
        :disabled="firewallAllowLan === false"
      />
      <label class="defColor" for="firewallAllowMulticast"
        >Allow Multicast when LAN traffic is allowed</label
      >
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

function processError(e) {
  console.error(e);
  sender.showMessageBox({
    type: "error",
    buttons: ["OK"],
    message: e.toString(),
  });
}

export default {
  data: function () {
    return {};
  },
  computed: {
    IsPersistent: function () {
      return this.$store.state.vpnState.firewallState.IsPersistent;
    },
    firewallAllowApiServers: {
      get() {
        return this.$store.state.vpnState.firewallState.IsAllowApiServers;
      },
      async set(value) {
        await sender.KillSwitchSetAllowApiServers(value);
      },
    },
    firewallAllowLan: {
      get() {
        return this.$store.state.vpnState.firewallState.IsAllowLAN;
      },
      async set(value) {
        await sender.KillSwitchSetAllowLAN(value);
      },
    },
    firewallAllowMulticast: {
      get() {
        return this.$store.state.vpnState.firewallState.IsAllowMulticast;
      },
      async set(value) {
        await sender.KillSwitchSetAllowLANMulticast(value);
      },
    },

    firewallActivateOnConnect: {
      get() {
        return this.$store.state.settings.firewallActivateOnConnect;
      },
      set(value) {
        this.$store.dispatch("settings/firewallActivateOnConnect", value);
      },
    },
    firewallDeactivateOnDisconnect: {
      get() {
        return this.$store.state.settings.firewallDeactivateOnDisconnect;
      },
      set(value) {
        this.$store.dispatch("settings/firewallDeactivateOnDisconnect", value);
      },
    },
  },
  watch: {
    IsPersistent() {
      this.updatePersistentFwUiState();
    },
  },
  mounted() {
    this.updatePersistentFwUiState();
  },

  methods: {
    updatePersistentFwUiState() {
      if (this.$store.state.vpnState.firewallState.IsPersistent) {
        this.$refs.radioFWPersistent.checked = true;
        this.$refs.radioFWOnDemand.checked = false;
      } else {
        this.$refs.radioFWPersistent.checked = false;
        this.$refs.radioFWOnDemand.checked = true;
      }
    },
    async onPersistentFWChange(value) {
      try {
        await sender.KillSwitchSetIsPersistent(value);
      } catch (e) {
        processError(e);
      }
      this.updatePersistentFwUiState();
    },
  },
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

div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-bottom: 17px;
  margin-left: 22px;
  max-width: 425px;
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
