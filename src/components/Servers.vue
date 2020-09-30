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

      <div class="buttonWithPopup">
        <button
          class="noBordersBtn sortBtn sortBtnPlatform"
          v-on:click="onSortMenuClicked()"
        >
          <img :src="sortImage" />
        </button>

        <!-- Popup -->
        <div
          class="popup popupMinShifted"
          v-bind:class="{
            popupMinShifted: isMinimizedUI
          }"
        >
          <div
            ref="pausePopup"
            class="popuptext"
            v-bind:class="{
              show: isSortMenu,
              popuptextMinShifted: isMinimizedUI
            }"
          >
            <div class="popup_menu_block">
              <div class="sortSelectedImg">
                <img :src="selectedImage" v-if="sortTypeStr === 'City'" />
              </div>
              <button class="flexRowRestSpace" v-on:click="onSortType('City')">
                City
              </button>
            </div>

            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <div class="sortSelectedImg">
                <img :src="selectedImage" v-if="sortTypeStr === 'Country'" />
              </div>
              <button
                class="flexRowRestSpace"
                v-on:click="onSortType('Country')"
              >
                Country
              </button>
            </div>

            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <div class="sortSelectedImg">
                <img :src="selectedImage" v-if="sortTypeStr === 'Latency'" />
              </div>
              <button
                class="flexRowRestSpace"
                v-on:click="onSortType('Latency')"
              >
                Latency
              </button>
            </div>

            <div class="popup_dividing_line" />
            <div class="popup_menu_block">
              <div class="sortSelectedImg">
                <img :src="selectedImage" v-if="sortTypeStr === 'Proximity'" />
              </div>
              <button
                class="flexRowRestSpace"
                v-on:click="onSortType('Proximity')"
              >
                Proximity
              </button>
            </div>
          </div>
        </div>
      </div>
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
      <div v-if="isFavoritesView == false && isFastestServerConfig === false">
        <div class="flexRow" v-if="!isMultihop">
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
        <!-- RANDOM -->
        <button
          class="serverSelectBtn flexRow"
          v-on:click="onRandomServerClicked()"
        >
          <serverNameControl
            class="serverName"
            :isRandomServer="true"
            :isShowSelected="
              isMultihop && isExitServer
                ? $store.getters['settings/isRandomExitServer']
                : $store.getters['settings/isRandomServer']
            "
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
              (value, event) => {
                configFastestSvrClicked(server, event);
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
import { enumValueName, getDistanceFromLatLonInKm } from "@/helpers/helpers";
import { ServersSortTypeEnum } from "@/store/types";

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
      isFastestServerConfig: false,
      isSortMenu: false
    };
  },
  created: function() {
    let self = this;
    window.addEventListener("click", function(e) {
      // close dropdown when clicked outside
      if (!self.$el.contains(e.target)) {
        self.isSortMenu = false;
      }
    });
  },
  computed: {
    isMinimizedUI: function() {
      return this.$store.state.settings.minimizedUI;
    },
    isFavoritesView: function() {
      return this.$store.state.uiState.serversFavoriteView;
    },
    isMultihop: function() {
      return this.$store.state.settings.isMultiHop;
    },
    isShowFavoriteDescriptionBlock: function() {
      if (!this.isFavoritesView) return false;
      let favSvrs = this.favoriteServers;
      return favSvrs == null || favSvrs.length == 0;
    },
    servers: function() {
      return this.$store.getters["vpnState/activeServers"];
    },

    sortTypeStr: function() {
      return enumValueName(
        ServersSortTypeEnum,
        this.$store.state.settings.serversSortType
      );
    },

    favoriteServers: function() {
      let favorites = this.$store.state.settings.serversFavoriteList;
      return this.servers.filter(s => favorites.includes(s.gateway));
    },

    filteredServers: function() {
      let store = this.$store;
      let sType = store.state.settings.serversSortType;
      function compare(a, b) {
        let ret = 0;
        switch (sType) {
          case ServersSortTypeEnum.City:
            return a.city.localeCompare(b.city);

          case ServersSortTypeEnum.Country:
            ret = a.country_code.localeCompare(b.country_code);
            if (ret != 0) return ret;
            return a.city.localeCompare(b.city);

          case ServersSortTypeEnum.Latency:
            if (a.ping === b.ping) return 0;
            if (a.ping < b.ping) return -1;
            return 1;

          case ServersSortTypeEnum.Proximity: {
            let l = store.state.location;
            if (l == null) return 0;

            var distA = getDistanceFromLatLonInKm(
              l.latitude,
              l.longitude,
              a.latitude,
              a.longitude
            );
            var distB = getDistanceFromLatLonInKm(
              l.latitude,
              l.longitude,
              b.latitude,
              b.longitude
            );

            if (distA === distB) return 0;
            if (distA < distB) return -1;

            return 1;
          }
        }
      }

      let servers = this.servers;
      if (this.isFavoritesView) servers = this.favoriteServers;

      if (this.filter == null || this.filter.length == 0)
        return servers.slice().sort(compare);
      let filter = this.filter.toLowerCase();
      let filtered = servers.filter(
        s =>
          s.city.toLowerCase().includes(filter) ||
          //s.country.toLowerCase().includes(filter) ||
          s.country_code.toLowerCase().includes(filter)
      );

      return filtered.slice().sort(compare);
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
      return require("@/assets/sort.svg");
    },
    selectedImage: function() {
      return require("@/assets/check-thin.svg");
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
    onSortMenuClicked: function() {
      this.isSortMenu = !this.isSortMenu;
    },
    onSortType: function(sortTypeStr) {
      this.$store.dispatch(
        "settings/serversSortType",
        ServersSortTypeEnum[sortTypeStr]
      );
      this.isSortMenu = false;
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
        return require("@/assets/star-active.svg");
      return require("@/assets/star-inactive.svg");
    },
    favoriteImageActive: function() {
      return require("@/assets/star-active.svg");
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
      if (this.$store.state.settings.isFastestServer === true) return false;

      if (this.isExitServer) {
        if (this.$store.state.settings.isRandomExitServer === true)
          return false;
        return this.$store.state.settings.serverExit.gateway === server.gateway;
      }
      if (this.$store.state.settings.isRandomServer === true) return false;
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
    configFastestSvrClicked(server, event) {
      if (server == null || server.gateway == null) return;
      let excludeSvrs = this.$store.state.settings.serversFastestExcludeList.slice();

      if (excludeSvrs.includes(server.gateway))
        excludeSvrs = excludeSvrs.filter(gw => gw != server.gateway);
      else excludeSvrs.push(server.gateway);

      const activeServers = this.servers.slice();
      const notExcludedActiveServers = activeServers.filter(
        s => !excludeSvrs.includes(s.gateway)
      );

      if (notExcludedActiveServers.length < 1) {
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "info",
          buttons: ["OK"],
          message: "Please, keep at least one server",
          detail: "Not allowed to exclude all servers."
        });
        event.preventDefault();
        return;
      } else
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
@import "@/components/scss/popup";

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

div.sortSelectedImg {
  margin-left: 11px;
  position: absolute;
  left: 0px;
  min-width: 13px;
}
</style>
