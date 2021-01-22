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

      <div v-if="isCanBePersistent">
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
          boot time before any other process. IVPN Firewall will always be
          active even when IVPN Client is not running
        </div>
      </div>
    </div>

    <!-- On-demand Firewall -->
    <div class="settingsBoldFont">
      On-demand Firewall:
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="firewallActivateOnConnect"
        :disabled="firewallPersistent === true"
        v-model="firewallActivateOnConnect"
      />
      <label class="defColor" for="firewallActivateOnConnect"
        >Activate IVPN Firewall on connect to VPN</label
      >
    </div>
    <div class="param">
      <input
        type="checkbox"
        id="firewallDeactivateOnDisconnect"
        :disabled="firewallPersistent === true"
        v-model="firewallDeactivateOnDisconnect"
      />
      <label class="defColor" for="firewallDeactivateOnDisconnect"
        >Deactivate IVPN Firewall on disconnect from VPN</label
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
import { Platform, PlatformEnum } from "@/platform/platform";
const sender = window.ipcSender;

export default {
  data: function() {
    return {};
  },
  methods: {},
  computed: {
    isCanBePersistent: function() {
      var release = sender.osRelease();
      if (release) {
        // TODO: persistant firewall not working properly on Linux Fedora
        // Here we are disabling this functionality in UI
        // Looking for a string like: "5.6.0-0.rc5.git0.2.fc32.x86_64" <- Fedora 32
        if (
          Platform() === PlatformEnum.Linux &&
          release.match(/.+\.fc[0-9]{2,3}\..+/)
        )
          return false;
      }
      return true;
    },
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

    firewallActivateOnConnect: {
      get() {
        return this.$store.state.settings.firewallActivateOnConnect;
      },
      set(value) {
        this.$store.dispatch("settings/firewallActivateOnConnect", value);
      }
    },
    firewallDeactivateOnDisconnect: {
      get() {
        return this.$store.state.settings.firewallDeactivateOnDisconnect;
      },
      set(value) {
        this.$store.dispatch("settings/firewallDeactivateOnDisconnect", value);
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
