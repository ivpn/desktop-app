<template>
  <div class="login">
    <spinner :loading="isProcessing" />

    <div class="column">
      <div>
        <div class="centered" style="margin-top: -50px; margin-bottom:50px">
          <img src="@/assets/logo.svg" />
        </div>

        <div v-if="isCaptchaRequired">
          <!-- CAPTCHA -->
          <div class="centered">
            <div class="large_text">Captcha Required</div>
            <div style="height: 12px" />
            <div class="small_text">
              Please enter number you see above
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
  </div>
</template>

<script>
import spinner from "@/components/controls/control-spinner.vue";
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

export default {
  props: {
    forceLoginAccount: {
      type: String,
      default: null
    }
  },
  components: {
    spinner
  },
  data: function() {
    return {
      accountID: "",
      isProcessing: false,

      rawResponse: null,
      apiResponseStatus: 0,

      capchaImageStyle: "",

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

      const force = true;
      this.Login(force);
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
    async Login(isForceLogout) {
      try {
        // check accoundID
        var pattern = new RegExp("^(i-....-....-....)|(ivpn[a-zA-Z0-9]{7,8})$"); // fragment locator
        if (this.accountID) this.accountID = this.accountID.trim();
        if (pattern.test(this.accountID) !== true) {
          throw new Error(
            "Your account ID has to be in 'i-XXXX-XXXX-XXXX' or 'ivpnXXXXXXXX' format. You can find it on other devices where you are logged in and in the client area of the IVPN website."
          );
        }

        this.isProcessing = true;
        const resp = await sender.Login(
          this.accountID,
          isForceLogout === true,
          this.captchaID,
          this.captcha,
          this.confirmation2FA
        );

        this.captcha = "";
        this.confirmation2FA = "";
        this.apiResponseStatus = resp.APIStatus;
        this.rawResponse = JSON.parse(resp.RawResponse);

        console.log("apiResponseStatus:", this.apiResponseStatus);

        if (resp.APIStatus !== API_SUCCESS) {
          if (resp.APIStatus === API_CAPTCHA_INVALID) {
            throw new Error(`Invalid captcha, please try again`);
          } else if (resp.APIStatus === API_CAPTCHA_REQUIRED) {
            // UI should be updated automatically based on data from 'resp.RawResponse'
          } else if (resp.APIStatus === API_2FA_TOKEN_NOT_VALID) {
            throw new Error(
              `Specified two-factor authentication token is not valid`
            );
          } else if (resp.APIStatus === API_2FA_REQUIRED) {
            // UI should be updated automatically based on data from 'resp.RawResponse'
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
                UpgradeToURL: resp.Account.UpgradeToURL
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
      //return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAPAAAABQCAMAAAAQlwhOAAAAP1BMVEUAAAA7dEUsZTZlnm97tIU1bj+Fvo8MRRYcVSZ0rX43cEESSxxlnm9ZkmNDfE0nYDEiWyxlnm8rZDUQSRomXzDFtqbCAAAAAXRSTlMAQObYZgAABMBJREFUeJzsWm1v4yAMxlPUat0qVfv/P/Z0aTB+B5q3acV3H1YSjB8e2xhIGjJkyJAhQ1rlcbYBB8vj8XaIzzZgyHqBc4b9OWfYBAlOQfzzcwpiABcu7Ey9i3fPRAkOwbBMxRncPx47Igab4QUsnAJ6T8DgMAyLLKB3Gt2R/QAvFCo8DO02BHdB2I1gQCJ5c4nsrQjeNSw9yb57uVyWBgfvswlKP6bkpXX7HMCId0Fc/Fa96K5UGNV9oM8ADBLw02qfYktH8QroW6bPw0sAz9Q6MexomIGCEwu/Sah5SPAzLVkE2yowl3nZfdva7HNFX9O+pUXFsAuY5DI7e8esd3r152eIGPBfYKxsNB9hA1emVWiFIcPdeSvAW4mrHHq6G62p0Fbi6zQbVzdVlbjeMFEDEzXJAGny/FT0LUjlIt2Qo7YG/PX15YzDGTaCdZomm/pZKVCSy4rDtDUsvNXU3Y/XRYw2moH8n2FrGhLArFRyTFwFXYYEtoc3TCIviAuYF0tqludnFt7/wgGr4ODpIUVM+/61OWARXnKWkTTe5fl/AUw6GNGb54ysv0Fm3K4aCWNYjcyeKjPQc2el+nFS+YpXWALUxFPdWqQ1UfTppYQvKYIkZSSJZvI+z2X0/WmacNgDzkcChkvkQVRDmIjJ3h+ou1rnPhPWZ6czfBX1snk2p4KS5TB2AoIjOGXMIQdCAcPX6zXR6hipExqMGM9JuegUmaGjiN1a3CwNcBUEOyuUDDx8yctT0RHBMYBB/82TDiubq0aCJFdGvVtYH0AxBlheMUszWlca3CVUtnD1Oq156bg1T6+YF2uJpBSU9OqPbazESrvsYaprZVitk50iSlnp5YGFyWaF7QxtwB6SFutBOVG9DzVX+Z02N9BoPtZuLcfsUBYpb9hoWb19ihu0sSxAMEmTaI944xQbrEqh2maaG4tHxdgJDI/0jQMhRi/NQezTseVWMZDnvI6XvFpStTTNtQHA3UwzHhYV5HfAcIiY66HtCRous7K9rE6wItjD6zIsKpbytFpD1gHXwitCLHwBKc6/DByqt39eAvK8BzWFTESgXuOfvsEzJiECNC9G/9CDMGAQKuqL7I7Ga2OwsqbMf5AbFOWkrxc1/DwP/PLaNqoNzMV4K7Qo8UsyBXdFcavcxa6u2yx22vECl71Xu1Uo16CJ2bZ2/5JzgsIcxwIrghgMQy5Wf6faRiXi3rcoXwk4OWFSi0T5CUGfr+W3nb3O/Idzs9+P98NBoPJ1xeDiG6mctXSIM4i9fScO0TvOx4eNOGtvCxT0/pSZfcXTzF2PbCmRhj/7RgnxJnaTbvkcvbKgae6Vb0PMEaybB+CHPmtj2DYmW6RsZGCr62plFDDCmZZKdDafZ/d7fktnVr8Mrb1V6Buk5D7NK+U3387sBjeZW5Pi6+raZv0whW8g4bJA9K+jNhWVwdQB/kbD5Jmz6T0OsEgcgJD94sGUytfTnFVzZ+jev20vetKLYY0qGr4XB5zIrSKlWb61LdSBlwzTZ0/79/GQWGa63+9dA70g398KcUqS3KOm/34/ArH7BE8e9jYB5RDAv0neDvARMfxX5Xa7nW3CoXK7vRniAfjPy7vhHTJkyJAhQ95Q/gUAAP//0o0UuFkwDccAAAAASUVORK5CYII=";
    },
    captchaID: function() {
      return this.rawResponse?.captcha_id;
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
.login {
  height: 100%;

  display: flex;
  justify-content: center;
  align-items: center;
}

.column {
  width: 100%;
  margin-left: 20px;
  margin-right: 20px;
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
</style>
