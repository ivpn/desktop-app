<template>
  <div id="app">
    <!--
    For no-bordered windows: print border manually -
    show transparent but bordered div top of the window
    Div should be 'transparent' for mouse events -->
    <div v-if="!isWindowHasFrame" class="border"></div>

    <!-- ability to move by mouse when no title for window -->
    <div class="title">
      <CustomTitleBar />
    </div>

    <router-view />
  </div>
</template>

<script>
import { IsWindowHasFrame } from "@/platform/platform";

const sender = window.ipcSender;
import { InitDefaultCopyMenus } from "@/context-menu/renderer";
import CustomTitleBar from "@/views/CustomTitleBar.vue";

export default {
  components: {
    CustomTitleBar,
  },
  beforeCreate() {
    // function using to re-apply all mutations
    // This is required to send to renderer processes current storage state
    sender.RefreshStorage();
  },
  computed: {
    isWindowHasFrame: function () {
      return IsWindowHasFrame();
    },
    isLoggedIn: function () {
      return this.$store.getters["account/isLoggedIn"];
    },
  },
  watch: {
    isLoggedIn() {
      if (this.isLoggedIn === false) this.$router.push("/");
    },
  },
};

InitDefaultCopyMenus();
</script>

<style lang="scss">
@import "@/components/scss/constants";

html * {
  // disable elements\text selelection
  -webkit-user-select: none;

  // assign default properties globally for all elements
  color: var(--text-color);

  font-family: $base-font-family; // !important;
}

input {
  background: var(--input-background);
}
textarea {
  background: var(--input-background);
}

body {
  background: var(--background-color);
}
/*
button:hover {
  opacity: 80%;
}
*/
#app {
  position: absolute;
  left: 0;
  top: 0;
  width: 100vw;
  height: 100vh;

  // disable scroolbars
  overflow-y: hidden;
  overflow-x: hidden;
}

.title {
  // Panel can be dragable by mouse
  // (we need this because using no title style for main window (for  macOS))
  -webkit-app-region: drag;
  height: 24px;
  width: 100%;

  position: absolute;
}
.border {
  // For no-bordered windows: print border manually -
  // show transparent but bordered div top of the window
  // Div should be 'transparent' for mouse events

  // top of all other elements
  z-index: 100;
  // full window
  position: absolute;
  left: 0px;
  right: 0px;
  top: 0px;
  bottom: 0px;
  // allow elements located under this div to receive mouse events
  pointer-events: none;
  // border style
  border: 1px solid rgba(128, 128, 128, 0.5);
}
</style>
