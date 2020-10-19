<template>
  <div id="main" class="row">
    <div id="leftPanel">
      <div style="margin: 20px">
        <div class="large_text ">Devices limit reached</div>
        <div style="height: 22px"></div>
        <div class="small_text ">
          According to your subscription plan you can use your IVPN account only
          on {{ devicesMaxLimit }} devices.
        </div>

        <div style="height: 24px"></div>

        <button class="master" v-if="isCanUpgrade" v-on:click="onUpgrade">
          Upgrade your subscription
        </button>

        <div style="height: 16px"></div>

        <button
          v-bind:class="{
            master: isCanUpgrade !== true,
            slave: isCanUpgrade === true
          }"
          v-if="isCanForceLogout"
          v-on:click="onForceLogout"
        >
          Log out from all devices
        </button>

        <div style="height: 16px"></div>
        <div class="centered">
          <button class="link linkFont" v-on:click="onTryAgain">
            Go back
          </button>
        </div>
      </div>

      <div class="elementFooter">
        <div class="small_text2 ">
          Do you think there is some issue?
        </div>
        <div style="height: 2px"></div>
        <button class="link linkFont" v-on:click="onContactSupport">
          Contact Support Team
        </button>
      </div>
    </div>

    <div id="rightPanel">
      <div>
        <img src="@/assets/devices-big.svg" />
      </div>
    </div>
  </div>
</template>

<script>
const { shell } = require("electron");
import { isValidURL } from "@/helpers/helpers";

export default {
  mounted() {
    this.accountID = this.$route.params.accountID;

    this.devicesMaxLimit = this.$route.params.devicesMaxLimit;

    this.CurrentPlan = this.$route.params.CurrentPlan;
    this.Upgradable = this.$route.params.Upgradable;
    this.UpgradeToPlan = this.$route.params.UpgradeToPlan;
    this.UpgradeToURL = this.$route.params.UpgradeToURL;
  },
  data: function() {
    return {
      accountID: null,
      devicesMaxLimit: 0,
      CurrentPlan: null,
      Upgradable: null,
      UpgradeToPlan: null,
      UpgradeToURL: null
    };
  },
  computed: {
    isCanUpgrade: function() {
      return this.Upgradable;
    },
    isCanForceLogout: function() {
      if (this.accountID == null || this.accountID === "") return false;
      return true;
    }
  },
  methods: {
    onTryAgain: function() {
      this.$router.push("/");
    },
    onForceLogout: async function() {
      this.$router.push({
        name: "Main",
        params: { forceLoginAccount: this.accountID }
      });
    },
    onUpgrade: function() {
      if (isValidURL(this.UpgradeToURL)) shell.openExternal(this.UpgradeToURL);
      else shell.openExternal(`https://www.ivpn.net/account`);
    },
    onContactSupport: function() {
      shell.openExternal(`https://www.ivpn.net/contactus`);
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

#main {
  height: 100%;
  display: flex;
  flex-direction: row;
}

#leftPanel {
  min-width: 320px;
  max-width: 320px;

  flex-direction: column;
  display: flex;
  justify-content: center;
  align-items: center;
}
#rightPanel {
  flex-direction: row;
  display: flex;
  align-items: center;
  justify-content: center;

  width: 100%;
  background: #f8c373;
}

.large_text {
  font-weight: 600;
  font-size: 18px;
  line-height: 120%;

  text-align: center;
}

.small_text {
  font-size: 15px;
  line-height: 18px;
  text-align: center;
  letter-spacing: -0.3px;

  color: var(--text-color-details);
}

.small_text2 {
  font-size: 14px;
  line-height: 17px;
  text-align: center;
  letter-spacing: -0.3px;

  color: var(--text-color-details);
}

.verticalSpace {
  margin-top: auto;
  margin-right: 0;
}
.linkFont {
  font-size: 12px;
  line-height: 18px;
  text-align: center;
  letter-spacing: -0.4px;
}

.centered {
  flex-direction: column;
  display: flex;
  justify-content: center;
  align-items: center;
}

.elementFooter {
  @extend .centered;
  position: fixed;
  bottom: 0%;
  margin-bottom: 36px;
}
</style>
