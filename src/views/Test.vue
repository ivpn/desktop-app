<template>
  <div class="about">
    <h1>This is a test page</h1>
    <Spinner />
    <div id="demo">
      <button v-on:click="doLogout">
        Logout
      </button>
      <button v-on:click="$router.push('/')">
        To main view...
      </button>

      <button v-on:click="doTestComonentsChange">
        Toggle
      </button>
      <transition name="fade">
        <p v-if="show">hello</p>
      </transition>
    </div>

    <div>
      <button v-on:click="doChangeVPNType">
        Change VPN type
      </button>
      {{ this.$store.state.settings.vpnType }}
    </div>

    <transition name="component-fade" mode="out-in">
      <component v-bind:is="currentViewComponent"></component>
    </transition>

    <pre align="left">GETTERS {{ theGetters }}</pre>
    <pre align="left">STATE {{ theState }}</pre>

    <button v-on:click="doTest">
      Test
    </button>
  </div>
</template>

<script>
import Login from "@/components/Login.vue";
import Control from "@/components/Control.vue";
import Spinner from "@/components/controls/control-spinner.vue";
import sender from "./../ipc/renderer-sender";

import { VpnTypeEnum } from "@/store/types";

function removeServers(key, value) {
  if (key == "vpnState/activeServers")
    return { DEBUG: `DEBUG: excluded from output (array len=${value.length})` };
  if (key == "serversHashed") return { DEBUG: `DEBUG: excluded from output` };
  if (key == "servers")
    return {
      DEBUG: `DEBUG: excluded from output (ovpn ${value.wireguard.length}; wg ${value.openvpn.length} )`
    };
  else return value;
}

export default {
  components: {
    Login,
    Control,
    Spinner
  },
  data: function() {
    return {
      show: true,
      currentViewComponent: Control
    };
  },

  computed: {
    theGetters: function() {
      return JSON.stringify(this.$store.getters, removeServers, 2);
    },
    theState: function() {
      return JSON.stringify(this.$store.state, removeServers, 2);
    }
  },

  methods: {
    async doLogout() {
      await sender.Logout();
    },
    doTest() {
      console.debug("button CLICKED!");
      this.$store.dispatch("testRenderInc");
    },
    doTestComonentsChange() {
      // this.show = !this.show;
      if (this.currentViewComponent === Control) {
        this.currentViewComponent = Login;
      } else {
        this.currentViewComponent = Control;
      }
    },
    doChangeVPNType() {
      console.log("Change VPN type");
      let type = this.$store.state.settings.vpnType;
      if (type === VpnTypeEnum.OpenVPN) type = VpnTypeEnum.WireGuard;
      else type = VpnTypeEnum.OpenVPN;
      this.$store.dispatch("settings/vpnType", type);
    }
  }
};
</script>

<style scoped lang="scss">
.about {
  -webkit-user-select: text;
  user-select: text;
  overflow-y: scroll;
  height: 100%;
}

.component-fade-enter-active,
.component-fade-leave-active {
  transition: opacity 0.3s ease;
}
.component-fade-enter, .component-fade-leave-to
/* .component-fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter, .fade-leave-to /* .fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}
</style>
