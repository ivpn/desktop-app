<template>
  <div id="main" class="row">
    <div id="leftPanel" class="settingsLeftPanel">
      <div class="row settingsLeftPanelHeader" id="leftPanelHeader">
        <button id="backBtn" class="noBordersBtn" v-on:click="goBack">
          <img :src="arrowLeftImagePath" />
        </button>
        <div class="Header settingsHeader">Settings</div>
      </div>

      <div class="row">
        <div id="tabsTitle">
          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('account')"
            v-bind:class="{
              activeBtn: view === 'account'
            }"
          >
            Account
          </button>

          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('general')"
            v-bind:class="{
              activeBtn: view === 'general'
            }"
          >
            General
          </button>

          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('connection')"
            v-bind:class="{
              activeBtn: view === 'connection'
            }"
          >
            Connection
          </button>
          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('firewall')"
            v-bind:class="{
              activeBtn: view === 'firewall'
            }"
          >
            IVPN Firewall
          </button>
          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('networks')"
            v-bind:class="{
              activeBtn: view === 'networks'
            }"
          >
            Networks
          </button>

          <button
            class="noBordersBtn tabTitleBtn"
            v-on:click="onView('antitracker')"
            v-bind:class="{
              activeBtn: view === 'antitracker'
            }"
          >
            AntiTracker
          </button>
          <button
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
      <div class="flexColumn" v-else>
        <img src="@/assets/temp/under-construction.jpg" />
      </div>
    </div>
  </div>
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

export default {
  components: {
    connectionView,
    accountView,
    generalView,
    firewallView,
    networksView,
    antitrackerView,
    dnsView
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
    arrowLeftImagePath: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return require("@/assets/arrow-left-windows.svg");
        case PlatformEnum.macOS:
          return require("@/assets/arrow-left-macos.svg");
        default:
          return require("@/assets/arrow-left-linux.svg");
      }
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
  background: #f2f3f6;
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
  margin-bottom: 30px;

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
}
button.activeBtn {
  font-weight: 500;
  color: #3b99fc;
}
</style>
