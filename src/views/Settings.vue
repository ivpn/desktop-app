<template>
  <transition name="fade-quick" appear>
    <div id="main" class="row">
      <div id="leftPanel" class="settingsLeftPanel">
        <div class="row settingsLeftPanelHeader" id="leftPanelHeader">
          <button id="backBtn" class="noBordersBtn" v-on:click="goBack">
            <!-- ARROW LEFT -->
            <div>
              <svg
                v-if="platform == PlatformEnum.Linux"
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M16 8.5H1.95312L8.10156 14.6484L7.39844 15.3516L0.046875 8L7.39844 0.648438L8.10156 1.35156L1.95312 7.5H16V8.5Z"
                  fill="var(--text-color)"
                />
              </svg>
              <svg
                v-else-if="platform == PlatformEnum.Windows"
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M16.001 8.5H1.9541L8.10254 14.6484L7.39941 15.3516L0.0478516 8L7.39941 0.648438L8.10254 1.35156L1.9541 7.5H16.001V8.5Z"
                  fill="var(--text-color)"
                />
              </svg>
              <svg
                v-else
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M12.6666 8H3.33331"
                  stroke="var(--text-color)"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <path
                  d="M7.99998 12.6663L3.33331 7.99968L7.99998 3.33301"
                  stroke="var(--text-color)"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
          </button>
          <div class="Header settingsHeader">Settings</div>
        </div>

        <div class="row">
          <div id="tabsTitle">
            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('account')"
              v-bind:class="{
                activeBtn: view === 'account'
              }"
            >
              Account
            </button>

            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('general')"
              v-bind:class="{
                activeBtn: view === 'general'
              }"
            >
              General
            </button>

            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('connection')"
              v-bind:class="{
                activeBtn: view === 'connection'
              }"
            >
              Connection
            </button>
            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('firewall')"
              v-bind:class="{
                activeBtn: view === 'firewall'
              }"
            >
              IVPN Firewall
            </button>
            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('networks')"
              v-bind:class="{
                activeBtn: view === 'networks'
              }"
            >
              Networks
            </button>

            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('antitracker')"
              v-bind:class="{
                activeBtn: view === 'antitracker'
              }"
            >
              AntiTracker
            </button>
            <button
              v-if="isLoggedIn"
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('dns')"
              v-bind:class="{
                activeBtn: view === 'dns'
              }"
            >
              DNS
            </button>
            <!--
          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('openvpn')"
            v-bind:class="{
              activeBtn: view === 'openvpn'
            }"
          >
            OpenVPN
          </button>
          -->
            <button
              class="noBordersBtn tabTitleBtn"
              v-on:click="onView('version')"
              v-bind:class="{
                activeBtn: view === 'version'
              }"
            >
              Version
            </button>
          </div>
        </div>
      </div>

      <div class="rightPanel ">
        <div class="flexColumn" v-if="view === 'connection'">
          <connectionView />
        </div>
        <div class="flexColumn" v-else-if="view === 'account'">
          <accountView />
        </div>
        <div class="flexColumn" v-else-if="view === 'general'">
          <generalView />
        </div>
        <div class="flexColumn" v-else-if="view === 'firewall'">
          <firewallView />
        </div>
        <div class="flexColumn" v-else-if="view === 'networks'">
          <networksView />
        </div>
        <div class="flexColumn" v-else-if="view === 'antitracker'">
          <antitrackerView />
        </div>
        <div class="flexColumn" v-else-if="view === 'dns'">
          <dnsView />
        </div>
        <div class="flexColumn" v-else-if="view === 'version'">
          <versionView />
        </div>
        <div class="flexColumn" v-else>
          <img src="@/assets/temp/under-construction.jpg" />
        </div>
      </div>
    </div>
  </transition>
</template>

<script>
const remote = require("electron").remote;
import { Platform, PlatformEnum } from "@/platform/platform";

import connectionView from "@/components/settings/settings-connection.vue";
import accountView from "@/components/settings/settings-account.vue";
import generalView from "@/components/settings/settings-general.vue";
import firewallView from "@/components/settings/settings-firewall.vue";
import networksView from "@/components/settings/settings-networks.vue";
import antitrackerView from "@/components/settings/settings-antitracker.vue";
import dnsView from "@/components/settings/settings-dns.vue";
import versionView from "@/components/settings/settings-version.vue";

export default {
  components: {
    connectionView,
    accountView,
    generalView,
    firewallView,
    networksView,
    antitrackerView,
    dnsView,
    versionView
  },
  mounted() {
    if (this.$route.params.view != null) this.view = this.$route.params.view;
  },
  data: function() {
    return {
      view: "general"
    };
  },
  computed: {
    isLoggedIn: function() {
      return this.$store.getters["account/isLoggedIn"];
    },
    PlatformEnum: function() {
      return PlatformEnum;
    },
    platform: function() {
      return Platform();
    }
  },
  methods: {
    goBack: function() {
      if (this.$store.state.settings.minimizedUI) {
        var window = remote.getCurrentWindow();
        window.close();
      } else this.$router.push("/");
    },
    onView: function(viewName) {
      this.view = viewName;
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

$back-btn-width: 50px;
$min-title-height: 26px;

div.row {
  display: flex;
  flex-direction: row;
  width: 100%;
}

#main {
  height: 100%;

  font-size: 13px;
  line-height: 16px;
  letter-spacing: -0.58px;
}
#leftPanel {
  padding-top: 50px;
  background: var(--background-color-alternate);
  min-width: 232px;
  height: 100vh;
}
#leftPanelHeader {
  padding-bottom: 38px;
}
#tabsTitle {
  width: 100%;

  display: flex;
  flex-flow: column;
  overflow: auto;

  margin-left: $back-btn-width;
}
.rightPanel {
  margin-top: 58px;
  margin-left: 34px;
  margin-right: 51px;
  margin-bottom: 20px;

  width: 100vw;
}

.rightPanel * {
  @extend .settingsDefaultText;
}

#backBtn {
  min-width: $back-btn-width;
  max-width: $back-btn-width;

  display: flex;
  justify-content: center;
  align-items: center;
}

.Header {
  font-style: normal;
  font-weight: 800;
  font-size: 24px;
  line-height: 29px;

  letter-spacing: -0.3px;
  text-transform: capitalize;
}

button.noBordersBtn {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;
  width: 100%;
}
button.tabTitleBtn {
  display: flex;
  padding: 0px;

  margin-bottom: 19px;

  font-size: 14px;
  line-height: 17px;

  color: var(--text-color-settings-menu);
}
button.activeBtn {
  font-weight: 500;
  color: #3b99fc;
}
</style>
