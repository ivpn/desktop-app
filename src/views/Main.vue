<template>
  <div class="main" id="flexview">
    <div class="leftPanel windowContentPaddingTop">
      <transition name="component-fade" mode="out-in">
        <component v-bind:is="currentViewComponent" id="left"></component>
      </transition>
    </div>
    <div id="right">
      <Map
        :isBlured="isMapBlured"
        :onAccountSettings="onAccountSettings"
        :onSettings="onSettings"
      />
    </div>
  </div>
</template>

<script>
import Init from "@/components/Init.vue";
import Login from "@/components/Login.vue";
import Control from "@/components/Control.vue";
import Map from "@/components/Map.vue";

export default {
  components: {
    Init,
    Login,
    Control,
    Map
  },
  computed: {
    isLoggedIn: function() {
      return this.$store.getters["account/isLoggedIn"];
    },
    currentViewComponent: function() {
      if (this.$store.state.isDaemonConnected === false) return Init;
      if (!this.isLoggedIn) return Login;
      return Control;
    },
    isMapBlured: function() {
      if (this.currentViewComponent !== Control) return "true";
      return "false";
    }
  },
  methods: {
    onAccountSettings: function() {
      this.$router.push({ name: "settings", params: { view: "account" } });
    },
    onSettings: function() {
      this.$router.push("settings");
    }
  }
};
</script>

<style scoped lang="scss">
.leftPanel {
  display: flex;
  flex-flow: column;
  height: 100%;
  color: #2a394b;
  overflow: auto;
}

.leftPanel::-webkit-scrollbar { 
    display: none;  // do not show scrollbars
}

#flexview {
  display: flex;
  flex-direction: row;
  height: 100vh;
}

#left {
  width: 320px;
  min-width: 320px;
  max-width: 320px;
}
#right {
  width: 0%; // ???
  flex-grow: 1;
}
</style>
