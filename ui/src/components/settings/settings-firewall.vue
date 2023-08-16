<template>
  <div>
    <div class="settingsTitle">FIREWALL SETTINGS</div>

    <div class="settingsBoldFont">Non-VPN traffic blocking:</div>

    <div>
      <div class="settingsRadioBtn">
        <input
          ref="radioFWOnDemand"
          type="radio"
          id="onDemand"
          name="firewall"
          value="false"
          v-on:click="onPersistentFWChange(false)"
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
            ref="radioFWPersistent"
            type="radio"
            id="alwaysOn"
            name="firewall"
            value="true"
            v-on:click="onPersistentFWChange(true)"
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
        type="checkbox"
        id="firewallAllowApiServers"
        v-model="firewallAllowApiServers"
      />
      <label class="defColor" for="firewallAllowApiServers"
        >Allow access to IVPN servers when Firewall is enabled</label
      >
    </div>

    <!-- On-demand Firewall -->
    <div class="settingsBoldFont">On-demand Firewall:</div>

    <div class="param">
      <input
        type="checkbox"
        id="firewallActivateOnConnect"
        :disabled="IsPersistent === true"
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
        :disabled="IsPersistent === true"
        v-model="firewallDeactivateOnDisconnect"
      />
      <label class="defColor" for="firewallDeactivateOnDisconnect"
        >Deactivate IVPN Firewall on disconnect from VPN</label
      >
    </div>

    <!-- TAB-view Extra config header -->

    <div class="flexRow" style="margin-top: 15px">
      <button
        v-on:click="onExtraCfgViewLan"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: extraCfgViewIsLan }"
      >
        LAN settings
      </button>
      <button
        v-on:click="onExtraCfgViewExceptions"
        class="selectableButtonOff"
        v-bind:class="{ selectableButtonOn: extraCfgViewIsExceptions }"
      >
        Exceptions
      </button>
      <button
        style="cursor: auto; flex-grow: 1"
        class="selectableButtonSeparator"
      ></button>
    </div>

    <!-- TAB: LAN settings -->
    <div style="margin-top: 12px">
      <div v-if="extraCfgViewIsLan">
        <div class="param">
          <input
            type="checkbox"
            id="firewallAllowLan"
            v-model="firewallAllowLan"
          />
          <label class="defColor" for="firewallAllowLan"
            >Allow LAN traffic when IVPN Firewall is enabled</label
          >

          <button
            class="noBordersBtn flexRow"
            title="Help"
            v-on:click="$refs.helpAllowLAN.showModal()"
          >
            <img src="@/assets/question.svg" />
          </button>
          <ComponentDialog ref="helpAllowLAN" header="Info">
            <div>
              <p>
                This includes traffic to all private address spaces in RFC 1918,
                3927, 4291, 4193.
              </p>
              <div class="settingsGrayLongDescriptionFont">
                'WiFi control' actions for untrusted networks will override this
                option.
              </div>
            </div>
          </ComponentDialog>
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
      <!-- TAB: Exceptions -->
      <div v-if="extraCfgViewIsExceptions">
        <div>
          <input
            class="settingsTextInput"
            style="width: calc(100% - 5px)"
            v-bind:class="{ badData: isExceptionsStringError === true }"
            placeholder="192.0.2.0/24, 198.51.100.1"
            v-model="firewallExceptions"
          />
          <div class="fwDescription" style="margin-left: 0px; margin-top: 4px">
            Enter a comma-separated list of IP addresses or subnets (using CIDR
            notation) that will be allowed through the firewall when enabled.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { isValidIpOrMask } from "@/helpers/helpers";
import ComponentDialog from "@/components/component-dialog.vue";

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
  components: {
    ComponentDialog,
  },
  props: { registerBeforeCloseHandler: Function },
  created() {
    // We have to call applyChanges() even when Settings window was closed by user
    // (the 'beforeUnmount()' is not called in this case)
    window.addEventListener("beforeunload", this.applyUserExceptions);

    if (this.registerBeforeCloseHandler != null) {
      // Register handler which will be called before closing current view
      // Handler MUST be 'async' function and MUST return 'true' to allow to switch current view
      this.registerBeforeCloseHandler(this.applyUserExceptions);
    }
  },
  async beforeUnmount() {
    window.removeEventListener("beforeunload", this.applyUserExceptions);
    await this.applyUserExceptions();
  },

  data: function () {
    return {
      extraCfgViewName: "", // possible values: "" or "lan" (LAN settings), "exceptions" (firewall exceptions)
      isExceptionsValueChanged: false,
    };
  },
  mounted() {
    this.updatePersistentFwUiState();
  },

  methods: {
    async applyUserExceptions(e) {
      // when component closing ->  update changed user exceptions (if necessary)

      if (this.isExceptionsStringError) {
        // activate 'exceptions' view
        this.extraCfgViewName = "exceptions";

        sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Error in firewall exceptions configuration",
          detail: `User exceptions will not be applied.`,
        });

        if (e && typeof e.preventDefault === "function") {
          // it is 'beforeunload' handler. Prevent closing window.
          e.preventDefault();
          e.returnValue = "";
        }
        return false;
      }

      if (this.isExceptionsValueChanged !== true) return true;
      this.isExceptionsValueChanged = false;
      await sender.KillSwitchSetUserExceptions(this.firewallExceptions);
      return true;
    },

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

    async onExtraCfgViewLan() {
      let isOK = true;
      if (this.extraCfgViewIsExceptions) {
        isOK = await this.applyUserExceptions();
      }
      if (isOK) this.extraCfgViewName = "lan";
    },
    onExtraCfgViewExceptions() {
      this.extraCfgViewName = "exceptions";
    },
  },
  watch: {
    IsPersistent() {
      this.updatePersistentFwUiState();
    },
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

    isExceptionsStringError() {
      if (!this.firewallExceptions) return false;

      let masks = this.firewallExceptions.split(/[\s,;]+/);
      for (const m of masks) {
        if (!m) continue;
        if (!isValidIpOrMask(m)) return true;
      }
      return false;
    },

    firewallExceptions: {
      get() {
        return this.$store.state.settings.firewallCfg.userExceptions;
      },
      async set(value) {
        this.isExceptionsValueChanged = true;
        let newFirewallCfg = Object.assign(
          {},
          this.$store.state.settings.firewallCfg
        );
        newFirewallCfg.userExceptions = value;

        this.$store.dispatch("settings/firewallCfg", newFirewallCfg);
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

    extraCfgViewIsLan() {
      return this.extraCfgViewName === "" || this.extraCfgViewName === "lan";
    },
    extraCfgViewIsExceptions() {
      return this.extraCfgViewName === "exceptions";
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
