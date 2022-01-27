<template>
  <transition name="fade-quick" appear>
    <div id="main" class="row">
      <div id="leftPanel" class="settingsLeftPanel">
        <div class="flexColumn">
          <div id="leftPanelHeader" class="row settingsLeftPanelHeader">
            <button id="backBtn" class="noBordersBtn" @click="goBack">
              <!-- ARROW LEFT -->
              <imgArrowLeft />
            </button>
            <div class="Header settingsHeader">Settings</div>
          </div>

          <!-- TABS -->
          <div class="row" style="flex-grow: 1">
            <div id="tabsTitle">
              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'account',
                }"
                @click="onView('account')"
              >
                Account
              </button>

              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'general',
                }"
                @click="onView('general')"
              >
                General
              </button>

              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'connection',
                }"
                @click="onView('connection')"
              >
                Connection
              </button>
              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'firewall',
                }"
                @click="onView('firewall')"
              >
                IVPN Firewall
              </button>
              <button
                v-if="isLoggedIn && isSplitTunnelVisible"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'splittunnel',
                }"
                @click="onView('splittunnel')"
              >
                Split Tunnel
              </button>
              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'networks',
                }"
                @click="onView('networks')"
              >
                WiFi control
              </button>

              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'antitracker',
                }"
                @click="onView('antitracker')"
              >
                AntiTracker
              </button>
              <button
                v-if="isLoggedIn"
                class="noBordersBtn tabTitleBtn"
                :class="{
                  activeBtn: view === 'dns',
                }"
                @click="onView('dns')"
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
            </div>
          </div>

          <!-- VERSION -->
          <div class="flexRow" style="flex-grow: 1">
            <div class="flexRow" style="margin: 20px; flex-grow: 1">
              <div style="flex-grow: 1; text-align: center">
                <div v-if="versionSingle" class="version">
                  <!-- single version -->
                  {{ versionSingle }}
                </div>

                <div v-else>
                  <!-- daemon and UI versions different-->
                  <div class="version">{{ versionUI }}</div>
                  <div class="version">daemon {{ versionDaemon }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="rightPanel">
        <div v-if="view === 'connection'" class="flexColumn">
          <connectionView />
        </div>
        <div v-else-if="view === 'account'" class="flexColumn">
          <accountView />
        </div>
        <div v-else-if="view === 'general'" class="flexColumn">
          <generalView />
        </div>
        <div v-else-if="view === 'firewall'" class="flexColumn">
          <firewallView />
        </div>
        <div v-else-if="view === 'splittunnel'" class="flexColumn">
          <splittunnelView />
        </div>
        <div v-else-if="view === 'networks'" class="flexColumn">
          <networksView />
        </div>
        <div v-else-if="view === 'antitracker'" class="flexColumn">
          <antitrackerView />
        </div>
        <div v-else-if="view === 'dns'" class="flexColumn">
          <dnsView />
        </div>
        <div v-else class="flexColumn">
          <!-- no view defined -->
        </div>
      </div>
    </div>
  </transition>
</template>

<script>
const sender = window.ipcSender;

import connectionView from "@/components/settings/settings-connection.vue";
import accountView from "@/components/settings/settings-account.vue";
import generalView from "@/components/settings/settings-general.vue";
import firewallView from "@/components/settings/settings-firewall.vue";
import splittunnelView from "@/components/settings/settings-splittunnel.vue";
import networksView from "@/components/settings/settings-networks.vue";
import antitrackerView from "@/components/settings/settings-antitracker.vue";
import dnsView from "@/components/settings/settings-dns.vue";
import imgArrowLeft from "@/components/images/arrow-left.vue";

export default {
  components: {
    connectionView,
    accountView,
    generalView,
    firewallView,
    splittunnelView,
    networksView,
    antitrackerView,
    dnsView,
    imgArrowLeft,
  },
  data: function () {
    return {
      view: "general",
    };
  },
  computed: {
    isLoggedIn: function () {
      return this.$store.getters["account/isLoggedIn"];
    },
    isSplitTunnelVisible() {
      return this.$store.getters["isSplitTunnelEnabled"];
    },
    versionSingle: function () {
      if (this.versionDaemon === this.versionUI) return this.versionDaemon;
      return null;
    },
    versionDaemon: function () {
      let v = this.$store.state.daemonVersion;
      if (!v) return "version unknown";
      return `v${v}`;
    },
    versionUI: function () {
      let v = sender.appGetVersion();
      if (!v) return "version unknown";
      return `v${v}`;
    },
  },
  mounted() {
    if (this.$route.params.view != null) this.view = this.$route.params.view;
    this.$store.dispatch("uiState/currentSettingsViewName", this.view);
  },
  methods: {
    goBack: function () {
      if (this.$store.state.settings.minimizedUI) {
        sender.closeCurrentWindow();
      } else this.$router.push("/");
      this.$store.dispatch("uiState/currentSettingsViewName", null);
    },
    onView: function (viewName) {
      this.view = viewName;
      this.$store.dispatch("uiState/currentSettingsViewName", this.view);
    },
  },
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
  max-width: 232px;
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
div.version {
  color: gray;
}
</style>
