<template>
  <div class="flexColumn">
    <div class="settingsTitle">ACCOUNT SETTINGS</div>

    <div class="flexColumn">
      <spinner :loading="isProcessing" />

      <div class="flexRowSpace">
        <div class="flexColumn">
          <div class="settingsGrayDescriptionFont">
            Account ID
          </div>

          <div class="settingsBigBoldFont" id="accountID">
            <label class="settingsBigBoldFont selectable">
              {{ this.$store.state.account.session.AccountID }}
            </label>
          </div>
          <div>
            <div
              class="statusButtonActive"
              v-if="IsAccountStateExists && IsActive"
            >
              ACTIVE
            </div>
            <div
              class="statusButtonNotActive"
              v-if="IsAccountStateExists && !IsActive"
            >
              NOT ACTIVE
            </div>
          </div>
        </div>

        <div ref="qrcode" class="qrcode"></div>
      </div>

      <!-- ACCOUNT EXPIRATION TEXT -->
      <div
        style="margin-bottom: 12px; color: darkorange"
        v-if="$store.getters['account/messageAccountExpiration']"
      >
        {{ $store.getters["account/messageAccountExpiration"] }}
      </div>
      <!-- FREE TRIAL EXPIRATION TEXT -->
      <div
        style="margin-bottom: 12px; color: darkorange"
        v-if="$store.getters['account/messageFreeTrial']"
      >
        {{ $store.getters["account/messageFreeTrial"] }}
      </div>

      <div class="subscriptionDetails" v-if="IsAccountStateExists">
        <div class="settingsBoldFont" style="margin-bottom: 16px">
          Subscription details:
        </div>

        <div class="flexRowAlignTop">
          <div style="min-width: 170px; margin-right:17px">
            <div class="settingsGrayDescriptionFont">Subscription</div>
            <div class="defColor" style="margin-top: 5px; margin-bottom:4px;">
              {{ CurrentPlan }}
            </div>

            <button
              class="noBordersTextBtn settingsLinkText"
              v-if="IsCanUpgradeToPro"
              v-on:click="upgrade"
            >
              Upgrade
            </button>
          </div>

          <div v-if="IsActive">
            <div class="settingsGrayDescriptionFont">Active until</div>
            <div class="defColor" style="margin-top: 5px; margin-bottom:4px;">
              {{ ActiveUntil }}
            </div>

            <button
              class="noBordersTextBtn settingsLinkText"
              v-on:click="addMoreTime"
            >
              Add more time
            </button>
          </div>
        </div>
      </div>

      <div class="proAcountDescriptionBlock" v-if="IsCanUpgradeToPro">
        <p>
          <strong>IVPN PRO</strong> gives you more possibilities to stay safe
          and protected:
        </p>

        <div>
          <div class="i">*</div>
          Connect up to <strong>7 devices</strong>
        </div>
        <div>
          <div class="i">*</div>
          Use <strong>Multi-Hop</strong> connections
        </div>
        <div>
          <div class="i">*</div>
          Turn on <strong>Port forwarding</strong>
        </div>
        <p>
          Login to the website to change subscription plan
        </p>
      </div>
    </div>

    <div class="flexRow">
      <button id="logoutButton" v-on:click="logOut()">LOG OUT</button>
    </div>
  </div>
</template>

<script>
import spinner from "@/components/controls/control-spinner.vue";
import { dateDefaultFormat } from "@/helpers/helpers";

import qrcode from "qrcode-generator";

const sender = window.ipcSender;

export default {
  components: {
    spinner
  },
  data: function() {
    return {
      isProcessing: false
    };
  },
  mounted() {
    // generating QRcode
    const typeNumber = 2;
    const errorCorrectionLevel = "M";
    const qr = qrcode(typeNumber, errorCorrectionLevel);

    let accId = "";
    if (
      this.$store.state.account != null &&
      this.$store.state.account.session != null &&
      this.$store.state.account.session.AccountID != null
    ) {
      accId = this.$store.state.account.session.AccountID;
    }

    qr.addData(accId);
    qr.make();
    this.$refs.qrcode.innerHTML = qr.createSvgTag(3, 10);

    // request account status (if not exists)
    if (this.$store.getters["account/isAccountStateExists"] !== true)
      this.accountStatusRequest();
  },
  methods: {
    async logOut() {
      // check: is it is necessary to warn user about enabled firewall?
      let isNeedPromptFirewallStatus = false;
      if (this.$store.state.vpnState.firewallState.IsEnabled == true) {
        isNeedPromptFirewallStatus = true;
        if (
          this.$store.state.vpnState.firewallState.IsPersistent === false &&
          this.$store.state.settings.firewallDeactivateOnDisconnect === true &&
          this.$store.getters["vpnState/isDisconnected"] === false
        ) {
          isNeedPromptFirewallStatus = false;
        }
      }

      // show dialog ("confirm to logout")
      let needToDisableFirewall = true;
      let needToResetSettings = false;
      const mes = "Do you really want to log out IVPN account?";
      const mesResetSettings = "Reset application settings to defaults";

      if (isNeedPromptFirewallStatus == true) {
        // LOGOUT message: Firewall is enabled
        let ret = await sender.showMessageBox(
          {
            type: "question",
            message: mes,
            detail:
              "The Firewall is enabled. All network access will be blocked.",
            checkboxLabel: mesResetSettings,
            buttons: ["Turn Firewall off and log out", "Log out", "Cancel"]
          },
          true
        );
        if (ret.response == 2) return; // cancel
        if (ret.response != 0) needToDisableFirewall = false;
        needToResetSettings = ret.checkboxChecked;
      } else {
        // LOGOUT message: Firewall is disabled
        let ret = await sender.showMessageBox(
          {
            type: "question",
            message: mes,
            checkboxLabel: mesResetSettings,
            buttons: ["Log out", "Cancel"]
          },
          true
        );
        if (ret.response == 1) return; // cancel
        needToResetSettings = ret.checkboxChecked;
      }

      // LOGOUT
      try {
        this.isProcessing = true;

        const isCanDeleteSessionLocally = false;
        await sender.Logout(
          needToResetSettings,
          needToDisableFirewall,
          isCanDeleteSessionLocally
        );
      } catch (e) {
        this.isProcessing = false;
        console.error(e);

        try {
          let ret = sender.showMessageBoxSync({
            type: "error",
            message:
              "Unable to contact server to log out. Please check Internet connectivity.\nDo you want to force log out?",
            detail:
              "This device will continue to count towards your device limit.",
            buttons: ["Force log out", "Cancel"]
          });
          if (ret == 1) return; // Cancel

          this.isProcessing = true;
          // FORCE LOGOUT
          const isCanDeleteSessionLocally = true;
          await sender.Logout(
            needToResetSettings,
            needToDisableFirewall,
            isCanDeleteSessionLocally
          );
        } catch (e) {
          sender.showMessageBoxSync({
            type: "error",
            message: "Failed to log out.",
            detail: e,
            buttons: ["OK"]
          });
        }
      } finally {
        this.isProcessing = false;
      }
    },
    async accountStatusRequest() {
      await sender.AccountStatus();
    },
    upgrade() {
      sender.shellOpenExternal(`https://www.ivpn.net/account`);
    },
    addMoreTime() {
      sender.shellOpenExternal(`https://www.ivpn.net/account`);
    }
  },
  computed: {
    IsAccountStateExists: function() {
      return this.$store.getters["account/isAccountStateExists"];
    },
    CurrentPlan: function() {
      return this.$store.state.account.accountStatus.CurrentPlan;
    },
    ActiveUntil: function() {
      return dateDefaultFormat(
        new Date(this.$store.state.account.accountStatus.ActiveUntil * 1000)
      );
    },
    IsActive: function() {
      return this.$store.state.account.accountStatus.Active;
    },
    IsCanUpgradeToPro: function() {
      return (
        this.IsAccountStateExists &&
        this.$store.state.account.accountStatus.Upgradable &&
        this.$store.state.account.accountStatus.CurrentPlan.toLowerCase() !=
          "ivpn pro"
      );
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}

.statusButton {
  border-radius: 4px;

  display: inline-block;

  font-weight: 500;
  font-size: 10px;
  line-height: 12px;
  letter-spacing: 1px;

  padding-top: 4px;
  padding-bottom: 4px;
  padding-left: 8px;
  padding-right: 8px;
}

.statusButtonActive {
  @extend .statusButton;
  background: rgba(177, 228, 125, 0.27);
  color: #64ad07;
}

.statusButtonNotActive {
  @extend .statusButton;
  background: rgba(228, 177, 125, 0.27);
  color: #ad6407;
}

.subscriptionDetails {
  margin-bottom: 40px;
}

.accountDescription * {
  font-size: 12px;
  line-height: 14px;
  letter-spacing: -0.4px;

  color: #3e6894;
}

.proAcountDescriptionBlock {
  @extend .accountDescription;
  background: rgba(57, 143, 230, 0.1);
  border-radius: 8px;
  padding-left: 14px;
  padding-right: 14px;
  padding-top: 7px;
  padding-bottom: 6px;
}

.accountDescription strong {
  font-weight: 600;
}

.accountDescription .i {
  color: #398fe6;
  display: inline;

  margin-left: 2px;
  margin-right: 4px;
}

.accountDescription div {
  margin-top: 6px;
}

#accountID {
  margin-top: 3px;
  margin-bottom: 7px;
}

#logoutButton {
  @extend .noBordersBtn;
  padding: 5px;
  margin-right: auto;
  margin-left: auto;

  font-weight: 500;
  font-size: 10px;
  line-height: 12px;

  letter-spacing: 1px;

  color: #8b9aab;
}
</style>
