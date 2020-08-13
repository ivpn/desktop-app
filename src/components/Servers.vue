<template>
  <div>
    <div class="flexRow serversButtonsHeader">
      <div>
        <button v-on:click="onBack" class="stateButtonOff">
          <img :src="arrowLeftImagePath" class="serversButtonsBack" />
        </button>
      </div>

      <div class="serversButtonsSpace" />

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

    <div class="flexRow">
      <input
        id="filter"
        class="styled"
        placeholder="Serach for a server"
        v-model="filter"
        v-bind:style="{ backgroundImage: 'url(' + searchImage + ')' }"
      />
    </div>

    <div id="list">
      <div v-for="server of filteredServers" v-bind:key="server.gateway">
        <button
          class="serverSelectBtn flexRow"
          v-on:click="onServerSelected(server)"
          v-bind:class="{ disabledButton: isInaccessibleServer(server) }"
        >
          <serverNameControl
            :server="server"
            class="serverName"
            :isShowSelected="isSelectedServer(server)"
          />
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
        </button>
      </div>
    </div>
  </div>
</template>

<script>
const { dialog, getCurrentWindow } = require("electron").remote;

import serverNameControl from "@/components/controls/control-server-name.vue";
import { isStrNullOrEmpty } from "@/helpers/helpers";
import { Platform, PlatformEnum } from "@/platform/platform";

export default {
  props: ["onBack", "onServerChanged", "isExitServer"],
  components: {
    serverNameControl
  },
  data: function() {
    return {
      filter: ""
    };
  },

  computed: {
    isFavoritesView: function() {
      return this.$store.state.uiState.serversFavoriteView;
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
    }
  },

  watch: {},

  methods: {
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
    favoriteImage: function(server) {
      if (
        this.$store.state.settings.serversFavoriteList.includes(server.gateway)
      )
        return require("@/assets/star-active.png");
      return require("@/assets/star-inactive.png");
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
      if (this.isExitServer)
        return this.$store.state.settings.serverExit.gateway === server.gateway;
      return this.$store.state.settings.serverEntry.gateway === server.gateway;
    },
    favoriteClicked: function(evt, server) {
      evt.stopPropagation();

      let favorites = this.$store.state.settings.serversFavoriteList;
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
#list {
  overflow: auto;
  padding-left: 20px;
  padding-right: 20px;
}

input#filter {
  background-position: 97% 50%; //right
  background-repeat: no-repeat;
  margin: 20px;
}

.disabledButton {
  opacity: 0.5;
}

.flexRow {
  display: flex;
  align-items: center;
}

.flexRowSpace {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
</style>
