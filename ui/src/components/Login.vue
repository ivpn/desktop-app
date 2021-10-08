<template>
  <div class="flexColumn">
    <div class="flexRow flexRowRestSpace">
      <spinner :loading="isProcessing" />

      <div class="column">
        <div class="centered" style="margin-top: -50px; margin-bottom:50px">
          <img src="@/assets/logo.svg" />
        </div>

        <div v-if="isCaptchaRequired">
          <!-- CAPTCHA -->
          <div class="centered">
            <div class="large_text">Captcha Required</div>
            <div style="height: 12px" />
            <div class="small_text">
              Please enter number you see below
            </div>
          </div>

          <div style="height: 21px" />
          <img :style="capchaImageStyle" :src="captchaImage" />
          <div style="height: 12px" />
          <input
            class="styledBig"
            ref="captcha"
            style="text-align: center"
            placeholder="xxxxxx"
            v-model="captcha"
            v-on:keyup="keyup($event)"
          />
        </div>
        <div v-else-if="is2FATokenRequired">
          <!-- 2FA TOKEN -->
          <div class="centered">
            <div class="large_text">2-Factor Authentication</div>
            <div style="height: 12px" />
            <div class="small_text">
              Account has two-factor authentication enabled. Please enter TOTP
              token to login
            </div>
          </div>

          <div style="height: 21px" />

          <input
            class="styledBig"
            ref="accountid"
            style="text-align: center"
            placeholder="xxxxxx"
            v-model="confirmation2FA"
            v-on:keyup="keyup($event)"
          />
        </div>
        <div v-else>
          <!-- ACCOUNT ID -->
          <div class="centered">
            <div class="large_text">Enter your Account ID</div>
            <div style="height: 12px" />
          </div>

          <div style="height: 21px" />

          <input
            class="styledBig"
            ref="accountid"
            style="text-align: center"
            placeholder="i-XXXX-... or ivpnXXXXXXXX"
            v-model="accountID"
            v-on:keyup="keyup($event)"
          />
        </div>

        <div style="height: 24px" />
        <button class="master" v-on:click="Login">Log In</button>
        <div style="height: 12px" />

        <button
          v-if="!isCaptchaRequired && !is2FATokenRequired"
          class="slave"
          v-on:click="CreateAccount"
        >
          Create an account
        </button>
        <button v-else class="slave" v-on:click="Cancel">
          Cancel
        </button>
      </div>
    </div>

    <div class="flexRow leftright_margins" style="margin-bottom: 20px;">
      <div
        class="flexRow flexRowRestSpace switcher_small_text"
        style="margin-right: 10px"
      >
        {{ firewallStatusText }}
      </div>

      <SwitchProgress
        :onChecked="firewallOnChecked"
        :isChecked="this.$store.state.vpnState.firewallState.IsEnabled"
        :isProgress="firewallIsProgress"
      />
    </div>
  </div>
</template>

<script>
import spinner from "@/components/controls/control-spinner.vue";
import SwitchProgress from "@/components/controls/control-switch-small2.vue";

import { IsOsDarkColorScheme } from "@/helpers/renderer";
import { ColorTheme } from "@/store/types";

const sender = window.ipcSender;
import {
  API_SUCCESS,
  API_SESSION_LIMIT,
  API_CAPTCHA_REQUIRED,
  API_CAPTCHA_INVALID,
  API_2FA_REQUIRED,
  API_2FA_TOKEN_NOT_VALID
} from "@/api/statuscode";

function processError(e) {
  console.error(e);
  sender.showMessageBox({
    type: "error",
    buttons: ["OK"],
    message: e.toString()
  });
}

export default {
  props: {
    forceLoginAccount: {
      type: String,
      default: null
    }
  },
  components: {
    spinner,
    SwitchProgress
  },
  data: function() {
    return {
      firewallIsProgress: false,

      accountID: "",
      isProcessing: false,

      rawResponse: null,
      apiResponseStatus: 0,

      capchaImageStyle: "",

      isForceLogoutRequested: false,
      captcha: "",
      confirmation2FA: ""
    };
  },
  mounted() {
    // COLOR SCHEME
    window.matchMedia("(prefers-color-scheme: dark)").addListener(() => {
      this.updateColorScheme();
    });
    this.updateColorScheme();

    if (this.$refs.accountid) this.$refs.accountid.focus();

    if (this.$route.params.forceLoginAccount != null) {
      this.accountID = this.$route.params.forceLoginAccount;

      let confirmation2FA = null;
      if (this.$route.params.extraArgs) {
        confirmation2FA = this.$route.params.extraArgs.confirmation2FA;
      }

      const force = true;
      this.Login(force, confirmation2FA);
    } else {
      if (this.$store.state.settings.isExpectedAccountToBeLoggedIn === true) {
        this.$store.dispatch("settings/isExpectedAccountToBeLoggedIn", false);
        setTimeout(() => {
          sender.showMessageBox({
            type: "info",
            buttons: ["OK"],
            message: `You are logged out.\n\nYou have been redirected to the login page to re-enter your credentials.`
          });
        }, 0);
      }
    }
  },
  methods: {
    async Login(isForceLogout, confirmation2FA) {
      try {
        // check accoundID
        var pattern = new RegExp("^(i-....-....-....)|(ivpn[a-zA-Z0-9]{7,8})$"); // fragment locator
        if (this.accountID) this.accountID = this.accountID.trim();
        if (pattern.test(this.accountID) !== true) {
          throw new Error(
            "Your account ID has to be in 'i-XXXX-XXXX-XXXX' or 'ivpnXXXXXXXX' format. You can find it on other devices where you are logged in and in the client area of the IVPN website."
          );
        }

        if (this.is2FATokenRequired && !this.confirmation2FA) {
          sender.showMessageBoxSync({
            type: "warning",
            buttons: ["OK"],
            message: "Failed to login",
            detail: `Please enter 6-digit verification code`
          });
          return;
        }

        this.isProcessing = true;
        const resp = await sender.Login(
          this.accountID,
          isForceLogout === true || this.isForceLogoutRequested === true
            ? true
            : false,
          this.captchaID,
          this.captcha,
          confirmation2FA ? confirmation2FA : this.confirmation2FA
        );
        this.isForceLogoutRequested = false;

        const oldConfirmation2FA = this.confirmation2FA;
        this.captcha = "";
        this.confirmation2FA = "";
        this.apiResponseStatus = resp.APIStatus;
        this.rawResponse = JSON.parse(resp.RawResponse);

        if (resp.APIStatus !== API_SUCCESS) {
          if (resp.APIStatus === API_CAPTCHA_INVALID) {
            throw new Error(`Invalid captcha, please try again`);
          } else if (resp.APIStatus === API_CAPTCHA_REQUIRED) {
            // UI should be updated automatically based on data from 'resp.RawResponse'
            this.isForceLogoutRequested = isForceLogout;
          } else if (resp.APIStatus === API_2FA_TOKEN_NOT_VALID) {
            throw new Error(
              `Specified two-factor authentication token is not valid`
            );
          } else if (resp.APIStatus === API_2FA_REQUIRED) {
            // UI should be updated automatically based on data from 'resp.RawResponse'
            this.isForceLogoutRequested = isForceLogout;
          } else if (
            resp.APIStatus === API_SESSION_LIMIT &&
            resp.Account != null
          ) {
            this.$router.push({
              name: "AccountLimit",
              params: {
                accountID: this.accountID,
                devicesMaxLimit: resp.Account.Limit,
                CurrentPlan: resp.Account.CurrentPlan,
                Upgradable: resp.Account.Upgradable,
                UpgradeToPlan: resp.Account.UpgradeToPlan,
                UpgradeToURL: resp.Account.UpgradeToURL,
                extraArgs: {
                  confirmation2FA: oldConfirmation2FA
                }
              }
            });
          } else throw new Error(`[${resp.APIStatus}] ${resp.APIErrorMessage}`);
        } else {
          try {
            await sender.GeoLookup();
          } catch (e) {
            console.error(e);
          }
        }
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: "Failed to login",
          detail: `${e}`
        });
      } finally {
        this.isProcessing = false;
      }
    },
    CreateAccount() {
      sender.shellOpenExternal(`https://www.ivpn.net/signup`);
    },
    Cancel() {
      this.rawResponse = null;
      this.apiResponseStatus = 0;
      this.captcha = "";
      this.confirmation2FA = "";
      this.isForceLogoutRequested = false;
    },
    keyup(event) {
      if (event.keyCode === 13) {
        // Cancel the default action, if needed
        event.preventDefault();
        this.Login();
      }
    },
    updateColorScheme() {
      let isDarkTheme = false;
      let scheme = sender.ColorScheme();
      if (scheme === ColorTheme.system) {
        isDarkTheme = IsOsDarkColorScheme();
      } else isDarkTheme = scheme === ColorTheme.dark;

      if (isDarkTheme)
        this.capchaImageStyle =
          "filter: grayscale(100%) brightness(0%) invert(100%); display: block; margin-left: auto; margin-right: auto; max-width:240px; max-height:80px";
      else
        this.capchaImageStyle =
          "filter: grayscale(100%) brightness(0%); display: block; margin-left: auto; margin-right: auto; max-width:240px; max-height:80px";
    },
    async firewallOnChecked(isEnabled) {
      this.firewallIsProgress = true;
      try {
        if (
          isEnabled === false &&
          this.$store.state.vpnState.firewallState.IsPersistent
        ) {
          let ret = await sender.showMessageBoxSync(
            {
              type: "question",
              message:
                "The always-on firewall is enabled. If you disable the firewall the 'always-on' feature will be disabled.",
              buttons: ["Disable Always-on firewall", "Cancel"]
            },
            true
          );

          if (ret == 1) return; // cancel
          await sender.KillSwitchSetIsPersistent(false);
        }

        this.firewallIsProgress = true;
        await sender.EnableFirewall(isEnabled);
      } catch (e) {
        processError(e);
      } finally {
        this.firewallIsProgress = false;
      }
    }
  },
  computed: {
    isCaptchaRequired: function() {
      return (
        (this.apiResponseStatus === API_CAPTCHA_REQUIRED ||
          this.apiResponseStatus === API_CAPTCHA_INVALID) &&
        this.captchaImage &&
        this.captchaID &&
        this.accountID
      );
    },
    isCaptchaInvalid: function() {
      return this.apiResponseStatus === API_CAPTCHA_INVALID;
    },
    is2FATokenRequired: function() {
      return (
        (this.apiResponseStatus === API_2FA_REQUIRED ||
          this.apiResponseStatus === API_2FA_TOKEN_NOT_VALID) &&
        this.accountID
      );
    },
    captchaImage: function() {
      return this.rawResponse?.captcha_image;
    },
    captchaID: function() {
      return this.rawResponse?.captcha_id;
    },
    firewallStatusText: function() {
      if (this.$store.state.vpnState.firewallState.IsEnabled)
        return "Firewall enabled and blocking all traffic";
      return "Firewall disabled";
    }
  },
  watch: {
    isCaptchaRequired() {
      if (!this.$refs.captcha || !this.$refs.accountid) return;
      if (this.isCaptchaRequired) this.$refs.captcha.focus();
      else this.$refs.accountid.focus();
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
.leftright_margins {
  margin-left: 20px;
  margin-right: 20px;
}

.column {
  @extend .leftright_margins;
  width: 100%;
}

.centered {
  margin-top: auto;
  margin-bottom: auto;
  text-align: center;
}

.large_text {
  font-weight: 600;
  font-size: 18px;
  line-height: 120%;
}

.small_text {
  font-size: 13px;
  line-height: 17px;
  letter-spacing: -0.208px;
  color: #98a5b3;
}

.switcher_small_text {
  font-size: 11px;
  line-height: 13px;
  color: var(--text-color-details);
}
</style>
