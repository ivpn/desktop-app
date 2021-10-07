<template>
  <div class="flexColumn">
    <!-- HEADER -->
    <div class="flexRow serversButtonsHeader">
      <div>
        <button v-on:click="goBack" class="stateButtonOff">
          <imgArrowLeft class="serversButtonsBack" />
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
              class="stateButtonOff"
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
        placeholder="Search for a server"
        v-model="filter"
        v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
      />

      <div class="buttonWithPopup">
        <button
          class="noBordersBtn sortBtn sortBtnPlatform"
          v-on:click="onSortMenuClicked()"
          v-click-outside="onSortMenuClickedOutside"
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
    <div
      ref="scrollArea"
      @scroll="recalcScrollButtonVisiblity()"
      class="commonMargins flexColumn scrollableColumnContainer"
    >
      <!-- FASTEST & RANDOMM SERVER -->
      <div v-if="isFavoritesView == false && isFastestServerConfig === false">
        <div class="flexRow" v-if="!isMultihop">
          <button
            class="serverSelectBtn flexRow"
            v-on:click="onFastestServerClicked()"
          >
            <serverNameControl class="serverName" :isFastestServer="true" />
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
          <serverNameControl class="serverName" :isRandomServer="true" />
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
            class="serverName"
            :server="server"
            :isCountryFirst="sortTypeStr === 'Country'"
          />

          <div
            class="flexColumn"
            v-if="isFastestServerConfig === false"
            style="margin-top: 11px"
          >
            <div class="flexRow">
              <serverPingInfoControl
                class="pingInfo"
                :server="server"
                :isShowPingTime="true"
              />

              <img
                :src="favoriteImage(server)"
                v-on:click="favoriteClicked($event, server)"
              />
            </div>
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

      <!-- SCROOL DOWN BUTTON -->
      <transition name="fade">
        <button
          class="btnScrollDown"
          v-if="isShowScrollButton"
          v-on:click="onScrollDown()"
        >
          <img src="@/assets/arrow-bottom.svg" />
        </button>
      </transition>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;
import serverNameControl from "@/components/controls/control-server-name.vue";
import serverPingInfoControl from "@/components/controls/control-server-ping.vue";
import SwitchProgress from "@/components/controls/control-switch-small.vue";
import imgArrowLeft from "@/components/images/arrow-left.vue";
import { isStrNullOrEmpty } from "@/helpers/helpers";
import { Platform, PlatformEnum } from "@/platform/platform";
import { enumValueName, getDistanceFromLatLonInKm } from "@/helpers/helpers";
import { ServersSortTypeEnum } from "@/store/types";

import Image_arrow_left_windows from "@/assets/arrow-left-windows.svg";
import Image_arrow_left_macos from "@/assets/arrow-left-macos.svg";
import Image_arrow_left_linux from "@/assets/arrow-left-linux.svg";
import Image_search_windows from "@/assets/search-windows.svg";
import Image_search_macos from "@/assets/search-macos.svg";
import Image_search_linux from "@/assets/search-linux.svg";
import Image_settings_windows from "@/assets/settings-windows.svg";
import Image_settings_macos from "@/assets/settings-macos.svg";
import Image_settings_linux from "@/assets/settings-linux.svg";
import Image_sort from "@/assets/sort.svg";
import Image_check_thin from "@/assets/check-thin.svg";
import Image_star_active from "@/assets/star-active.svg";
import Image_star_inactive from "@/assets/star-inactive.svg";

import ClickOutside from "vue-click-outside";

export default {
  directives: {
    ClickOutside
  },
  props: [
    "onBack",
    "onServerChanged",
    "isExitServer",
    "onFastestServer",
    "onRandomServer"
  ],
  components: {
    serverNameControl,
    serverPingInfoControl,
    SwitchProgress,
    imgArrowLeft
  },
  data: function() {
    return {
      filter: "",
      isFastestServerConfig: false,
      isSortMenu: false,
      isShowScrollButton: false
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
  mounted() {
    this.recalcScrollButtonVisiblity();
    const resizeObserver = new ResizeObserver(this.recalcScrollButtonVisiblity);
    resizeObserver.observe(this.$refs.scrollArea);
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
        switch (sType) {
          case ServersSortTypeEnum.City:
            return a.city.localeCompare(b.city);

          case ServersSortTypeEnum.Country: {
            if (!a.country && !b.country) return 0;
            if (!a.country) return 1;

            let ret = 0;
            ret = a.country.localeCompare(b.country);
            if (ret != 0) return ret;
            // If countries are the same - compare cities
            if (a.city && b.city) return a.city.localeCompare(b.city);
            return ret;
          }

          case ServersSortTypeEnum.Latency:
            if (a.ping && b.ping) return a.ping - b.ping;
            if (a.ping && !b.ping) return -1;
            if (!a.ping && b.ping) return 1;
            return 0;

          case ServersSortTypeEnum.Proximity: {
            const l = store.getters["getLastRealLocation"];
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
          (s.city && s.city.toLowerCase().includes(filter)) ||
          (s.country && s.country.toLowerCase().includes(filter)) ||
          (s.country_code && s.country_code.toLowerCase().includes(filter))
      );

      return filtered.slice().sort(compare);
    },

    arrowLeftImagePath: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return Image_arrow_left_windows;
        case PlatformEnum.macOS:
          return Image_arrow_left_macos;
        default:
          return Image_arrow_left_linux;
      }
    },
    searchImage: function() {
      if (!isStrNullOrEmpty(this.filter)) return null;

      switch (Platform()) {
        case PlatformEnum.Windows:
          return Image_search_windows;
        case PlatformEnum.macOS:
          return Image_search_macos;
        default:
          return Image_search_linux;
      }
    },
    settingsImage: function() {
      switch (Platform()) {
        case PlatformEnum.Windows:
          return Image_settings_windows;
        case PlatformEnum.macOS:
          return Image_settings_macos;
        default:
          return Image_settings_linux;
      }
    },
    sortImage: function() {
      return Image_sort;
    },
    selectedImage: function() {
      return Image_check_thin;
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
        sender.showMessageBoxSync({
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
    onSortMenuClickedOutside: function() {
      this.isSortMenu = false;
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
        return Image_star_active;
      return Image_star_inactive;
    },
    favoriteImageActive: function() {
      return Image_star_active;
    },
    onFastestServerConfig() {
      this.isFastestServerConfig = true;
      this.filter = "";
    },
    isInaccessibleServer: function(server) {
      if (this.$store.state.settings.isMultiHop === false) return false;
      let ccSkip = "";

      let connected = !this.$store.getters["vpnState/isDisconnected"];
      if (
        // ENTRY SERVER
        !this.isExitServer &&
        this.$store.state.settings.serverExit &&
        (connected || !this.$store.state.settings.isRandomExitServer)
      )
        ccSkip = this.$store.state.settings.serverExit.country_code;
      else if (
        // EXIT SERVER
        this.isExitServer &&
        this.$store.state.settings.serverEntry &&
        (connected || !this.$store.state.settings.isRandomServer)
      )
        ccSkip = this.$store.state.settings.serverEntry.country_code;
      if (server.country_code === ccSkip) return true;
      return false;
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
        sender.showMessageBoxSync({
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
    },
    recalcScrollButtonVisiblity() {
      let sa = this.$refs.scrollArea;
      if (sa == null) {
        this.isShowScrollButton = false;
        return;
      }

      const show = sa.scrollHeight > sa.clientHeight + sa.scrollTop;

      // hide - imadiately; show - with 1sec delay
      if (!show) this.isShowScrollButton = false;
      else {
        setTimeout(() => {
          this.isShowScrollButton =
            sa.scrollHeight > sa.clientHeight + sa.scrollTop;
        }, 1000);
      }
    },
    onScrollDown() {
      let sa = this.$refs.scrollArea;
      if (sa == null) return;
      sa.scrollTo({
        top: sa.clientHeight * 0.9 + sa.scrollTop, //sa.scrollHeight,
        behavior: "smooth"
      });
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
  max-width: 195px;
  width: 195px;
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
  color: var(--text-color-details);
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
