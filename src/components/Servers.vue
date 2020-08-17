<template>
  <div class="flexColumn">
    <!-- HEADER -->
    <div class="flexRow serversButtonsHeader">
      <div>
        <button v-on:click="goBack" class="stateButtonOff">
          <img :src="arrowLeftImagePath" class="serversButtonsBack" />
        </button>
      </div>

      <div class="serversButtonsSpace" />

      <div style="width: 100%" v-if="isFastestServerConfig === false">
        <div class="flexRow" style="flex-grow: 1">
          <div style="flex-grow: 1">
            <button
              style="width: 100%"
              v-on:click="showAll"
              class="stateButtonOff stateButtonLeft"
              v-bind:class="{ stateButtonOn: !isFavoritesView }"
            >
              all servers
            </button>
          </div>

          <div style="flex-grow: 1">
            <button
              style="width: 100%"
              v-on:click="showFavorites"
              class="stateButtonOff stateButtonRight"
              v-bind:class="{ stateButtonOn: isFavoritesView }"
            >
              favorites
            </button>
          </div>
        </div>
      </div>

      <div style="width: 100%" v-if="isFastestServerConfig">
        <div class="flexRow" style="flex-grow: 1">
          <div style="flex-grow: 1">
            <button
              style="width: 100%"
              v-on:click="showAll"
              class="stateButtonOff stateButtonLeft"
              v-bind:class="{ stateButtonOn: !isFavoritesView }"
            >
              fastest server settings
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- EMPTY FAVORITE SERVERS DESCRIPTION BLOCK -->
    <div v-if="isShowFavoriteDescriptionBlock">
      <div class="text">
        Your favorite (<img :src="favoriteImageActive()" />) servers will be
        displayed here
      </div>
    </div>

    <!-- FILTER -->
    <div class="commonMargins flexRow" v-if="!isShowFavoriteDescriptionBlock">
      <input
        id="filter"
        class="styled"
        placeholder="Serach for a server"
        v-model="filter"
        v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
      />

      <!--
      <button class="noBordersBtn sortBtn sortBtnPlatform">
        <img :src="sortImage" />
      </button>
      -->
    </div>

    <div
      v-if="isFastestServerConfig"
      class="small_text"
      style="margin-bottom: 5px;"
    >
      Disable servers you do not want to be choosen as the fastest server
    </div>

    <!-- SERVERS LIST BLOCK -->
    <div class="commonMargins flexColumn scrollableColumnContainer">
      <!-- FASTEST & RANDOMM SERVER -->
      <div
        v-if="
          !isMultihop &&
            isFavoritesView == false &&
            isFastestServerConfig === false
        "
      >
        <div class="flexRow">
          <button
            class="serverSelectBtn flexRow"
            v-on:click="onFastestServerClicked()"
          >
            <serverNameControl
              class="serverName"
              :isFastestServer="true"
              :isShowSelected="$store.getters['settings/isFastestServer']"
            />
          </button>
          <button class="noBordersBtn" v-on:click="onFastestServerConfig()">
            <img :src="settingsImage" />
          </button>
        </div>
        <button
          class="serverSelectBtn flexRow"
          v-on:click="onRandomServerClicked()"
        >
          <serverNameControl
            class="serverName"
            :isRandomServer="true"
            :isShowSelected="$store.getters['settings/isRandomServer']"
          />
        </button>
      </div>

      <!-- SERVERS LIST -->
      <div
        class="flexRow"
        v-for="server of filteredServers"
        v-bind:key="server.gateway"
      >
        <button
          class="serverSelectBtn flexRow"
          v-on:click="onServerSelected(server)"
          v-bind:class="{ disabledButton: isInaccessibleServer(server) }"
        >
          <serverNameControl
            :server="server"
            class="serverName"
            :isShowSelected="
              isFastestServerConfig === false && isSelectedServer(server)
            "
          />

          <div class="flexRow" v-if="isFastestServerConfig === false">
            <!-- NO CONFIG -->
            <serverNameControl
              class="pingInfo"
              :server="server"
              isHideName="true"
              isHideFlag="true"
              isShowPingPicture="true"
              isShowPingTime="true"
            />

            <img
              :src="favoriteImage(server)"
              v-on:click="favoriteClicked($event, server)"
            />
          </div>
        </button>

        <div class="flexRow" v-if="isFastestServerConfig">
          <!-- CONFIG -->
          <SwitchProgress
            :onChecked="
              () => {
                configFastestSvrClicked(server);
              }
            "
            :isChecked="!isSvrExcludedFomFastest(server)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
const { dialog, getCurrentWindow } = require("electron").remote;

import serverNameControl from "@/components/controls/control-server-name.vue";
import SwitchProgress from "@/components/controls/control-switch-small.vue";
import { isStrNullOrEmpty } from "@/helpers/helpers";
import { Platform, PlatformEnum } from "@/platform/platform";

export default {
  props: [
    "onBack",
    "onServerChanged",
    "isExitServer",
    "onFastestServer",
    "onRandomServer"
  ],
  components: {
    serverNameControl,
    SwitchProgress
  },
  data: function() {
    return {
      filter: "",
      isFastestServerConfig: false
    };
  },

  computed: {
    isFavoritesView: function() {
      return this.$store.state.uiState.serversFavoriteView;
    },
    isMultihop: function() {
      return this.$store.state.settings.isMultiHop;
    },
    isShowFavoriteDescriptionBlock: function() {
      if (!this.isFavoritesView) return false;
      let favSvrs = this.$store.state.settings.serversFavoriteList;
      return favSvrs == null || favSvrs.length == 0;
    },
    servers: function() {
      return this.$store.getters["vpnState/activeServers"];
    },
    favoriteServers: function() {
      let favorites = this.$store.state.settings.serversFavoriteList;
      return this.servers.filter(s => favorites.includes(s.gateway));
    },

    filteredServers: function() {
      let servers = this.servers;
      if (this.isFavoritesView) servers = this.favoriteServers;

      if (this.filter == null || this.filter.length == 0) return servers;
      let filter = this.filter.toLowerCase();
      return servers.filter(
        s =>
          s.city.toLowerCase().includes(filter) ||
          //s.country.toLowerCase().includes(filter) ||
          s.country_code.toLowerCase().includes(filter)
      );
    },
    arrowLeftImagePath: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return require("@/assets/arrow-left-windows.svg");
        case PlatformEnum.macOS:
          return require("@/assets/arrow-left-macos.svg");
        default:
          return require("@/assets/arrow-left-linux.svg");
      }
    },
    searchImage: function() {
      if (!isStrNullOrEmpty(this.filter)) return null;

      switch (Platform()) {
        case PlatformEnum.Windows:
          return require("@/assets/search-windows.svg");
        case PlatformEnum.macOS:
          return require("@/assets/search-macos.svg");
        default:
          return require("@/assets/search-linux.svg");
      }
    },
    settingsImage: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return require("@/assets/settings-windows.svg");
        case PlatformEnum.macOS:
          return require("@/assets/settings-macos.svg");
        default:
          return require("@/assets/settings-linux.svg");
      }
    },
    sortImage: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return require("@/assets/sort-windows.svg");
        case PlatformEnum.macOS:
          return require("@/assets/sort-macos.svg");
        default:
          return require("@/assets/sort-linux.svg");
      }
    }
  },

  methods: {
    goBack: function() {
      if (this.isFastestServerConfig) {
        this.filter = "";
        this.isFastestServerConfig = false;
        return;
      }
      if (this.onBack != null) this.onBack();
    },
    onServerSelected: function(server) {
      if (this.isInaccessibleServer(server)) {
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "info",
          buttons: ["OK"],
          message: "Entry and exit servers cannot be in the same country",
          detail:
            "When using multihop you must select entry and exit servers in different countries. Please select a different entry or exit server."
        });
        return;
      }

      this.onServerChanged(server, this.isExitServer != null);
      this.onBack();
    },
    onFastestServerClicked() {
      if (this.onFastestServer != null) this.onFastestServer();
      this.onBack();
    },
    onRandomServerClicked() {
      if (this.onRandomServer != null) this.onRandomServer();
      this.onBack();
    },
    isSvrExcludedFomFastest: function(server) {
      return this.$store.state.settings.serversFastestExcludeList.includes(
        server.gateway
      );
    },
    favoriteImage: function(server) {
      if (
        this.$store.state.settings.serversFavoriteList.includes(server.gateway)
      )
        return require("@/assets/star-active.png");
      return require("@/assets/star-inactive.png");
    },
    favoriteImageActive: function() {
      return require("@/assets/star-active.png");
    },
    onFastestServerConfig() {
      this.isFastestServerConfig = true;
      this.filter = "";
    },
    isInaccessibleServer: function(server) {
      if (this.$store.state.settings.isMultiHop === false) return false;
      let ccSkip = "";
      if (!this.isExitServer)
        ccSkip = this.$store.state.settings.serverExit.country_code;
      else ccSkip = this.$store.state.settings.serverEntry.country_code;
      if (server.country_code === ccSkip) return true;
      return false;
    },
    isSelectedServer: function(server) {
      if (server == null) return false;
      if (
        this.$store.state.settings.isFastestServer === true ||
        this.$store.state.settings.isRandomServer === true
      )
        return false;
      if (this.isExitServer)
        return this.$store.state.settings.serverExit.gateway === server.gateway;
      return this.$store.state.settings.serverEntry.gateway === server.gateway;
    },
    favoriteClicked: function(evt, server) {
      evt.stopPropagation();

      let favorites = this.$store.state.settings.serversFavoriteList.slice();
      let serversHashed = this.$store.state.vpnState.serversHashed;
      let gateway = server.gateway;

      if (favorites.includes(gateway)) {
        // remove
        console.log(`Removing favorite ${gateway}`);
        favorites = favorites.filter(gw => gw != gateway);
      } else {
        // add
        console.log(`Adding favorite ${gateway}`);
        if (serversHashed[gateway] == null) return;
        favorites.push(gateway);
      }

      this.$store.dispatch("settings/serversFavoriteList", favorites);
    },
    configFastestSvrClicked(server) {
      console.log("configFastestSvrClicked", server);
      if (server == null || server.gateway == null) return;
      let excludeSvrs = this.$store.state.settings.serversFastestExcludeList.slice();

      if (excludeSvrs.includes(server.gateway)) {
        excludeSvrs = excludeSvrs.filter(gw => gw != server.gateway);
      } else {
        excludeSvrs.push(server.gateway);
      }
      this.$store.dispatch("settings/serversFastestExcludeList", excludeSvrs);
    },

    showFavorites: function() {
      this.$store.dispatch("uiState/serversFavoriteView", true);
      this.filter = "";
    },
    showAll: function() {
      this.$store.dispatch("uiState/serversFavoriteView", false);
      this.filter = "";
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
$paddingLeftRight: 20px;

.commonMargins {
  margin-left: $paddingLeftRight;
  margin-right: $paddingLeftRight;
}

input#filter {
  background-position: 97% 50%; //right
  background-repeat: no-repeat;
  margin-top: $paddingLeftRight;
  margin-bottom: $paddingLeftRight;
}

.disabledButton {
  opacity: 0.5;
}

.serverSelectBtn {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  height: 48px;
  width: 100%;

  padding: 0px;
}

.serverName {
  max-width: 196px;
  width: 196px;
}

.pingInfo {
  max-width: 72px;
  width: 72px;
}

.pingtext {
  margin-left: 8px;
}

.text {
  margin: $paddingLeftRight;
  margin-top: 60px;
  text-align: center;
}

.small_text {
  margin-left: $paddingLeftRight;
  margin-right: $paddingLeftRight;
  font-size: 11px;
  line-height: 13px;
  color: $base-text-color-details;
}

button.sortBtn {
  margin-left: 5px;
}
</style>
