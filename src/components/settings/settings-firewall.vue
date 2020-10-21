<template>
  <div>
    <div class="settingsTitle">FIREWALL</div>

    <div class="settingsBoldFont">
      Non-VPN traffic blocking:
    </div>

    <div>
      <div class="settingsRadioBtn">
        <input
          type="radio"
          id="onDemand"
          name="firewall"
          :value="false"
          v-model="firewallPersistent"
          :checked="firewallPersistent !== true"
        />
        <label class="defColor" for="onDemand">On-demand</label>
      </div>
      <div class="fwDescription">
        When this option is enabled the IVPN Firewall can be either manually
        activated or automatically activated when the VPN connection is
        established - see On-demand Firewall options below
      </div>

      <div class="settingsRadioBtn">
        <input
          type="radio"
          id="alwaysOn"
          name="firewall"
          :value="true"
          v-model="firewallPersistent"
          :checked="firewallPersistent === true"
        />
        <label class="defColor" for="alwaysOn">Always-on firewall</label>
      </div>
      <div class="fwDescription">
        When the option is enabled the IVPN Firewall is started during system
        boot time before any other process. IVPN Firewall will always be active
        even when IVPN Client is not running
      </div>
    </div>

    <!-- On-demand Firewall -->
    <div class="settingsBoldFont">
      On-demand Firewall:
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="firewallOnOffOnConnect"
        :disabled="firewallPersistent === true"
        v-model="firewallOnOffOnConnect"
      />
      <label class="defColor" for="firewallOnOffOnConnect"
        >Activate on connect to VPN and deactivate on disconnect</label
      >
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="firewallOffOnExit"
        :disabled="firewallPersistent === true"
        v-model="firewallOffOnExit"
      />
      <label class="defColor" for="firewallOffOnExit"
        >Deactivate IVPN Firewall on IVPN Client exit (when VPN
        disconnected)</label
      >
    </div>

    <!-- LAN settings -->
    <div class="settingsBoldFont">
      LAN settings:
    </div>
    <div class="param">
      <input type="checkbox" id="firewallAllowLan" v-model="firewallAllowLan" />
      <label class="defColor" for="firewallAllowLan"
        >Allow LAN traffic when IVPN Firewall is enabled</label
      >
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="firewallAllowMulticast"
        :disabled="firewallAllowLan === false"
        v-model="firewallAllowMulticast"
      />
      <label class="defColor" for="firewallAllowMulticast"
        >Allow Multicast when LAN traffic is allowed</label
      >
    </div>
  </div>
</template>

<script>
import sender from "@/ipc/renderer-sender";

export default {
  data: function() {
    return {};
  },
  methods: {},
  computed: {
    firewallPersistent: {
      get() {
        return this.$store.state.vpnState.firewallState.IsPersistent;
      },
      async set(value) {
        await sender.KillSwitchSetIsPersistent(value);
      }
    },
    firewallAllowLan: {
      get() {
        return this.$store.state.vpnState.firewallState.IsAllowLAN;
      },
      async set(value) {
        await sender.KillSwitchSetAllowLAN(value);
      }
    },
    firewallAllowMulticast: {
      get() {
        return this.$store.state.vpnState.firewallState.IsAllowMulticast;
      },
      async set(value) {
        await sender.KillSwitchSetAllowLANMulticast(value);
      }
    },

    firewallOnOffOnConnect: {
      get() {
        return this.$store.state.settings.firewallOnOffOnConnect;
      },
      set(value) {
        this.$store.dispatch("settings/firewallOnOffOnConnect", value);
      }
    },
    firewallOffOnExit: {
      get() {
        return this.$store.state.settings.firewallOffOnExit;
      },
      set(value) {
        this.$store.dispatch("settings/firewallOffOnExit", value);
      }
    }
  }
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
