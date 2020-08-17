<template>
  <div id="app">
    <div class="title" v-if="isShowDragableTitle">
      <!--
      <img
        src="@/assets/logo_grey.svg"
        style="margin-top: 12px; margin-left:78px"
      /> -->
    </div>
    <router-view />
  </div>
</template>

<script>
import sender from "@/ipc/renderer-sender";
import { IsWindowHasTitle } from "@/platform/platform";
import { Platform, PlatformEnum } from "@/platform/platform";

export default {
  mounted() {
    // function using to re-apply all mutations
    // This is required to send to renderer processes current storage state
    sender.RefreshStorage();
  },
  computed: {
    isLoggedIn: function() {
      return this.$store.getters["account/isLoggedIn"];
    },
    isShowDragableTitle: function() {
      // macOS UI has no standart movable header (we are adding transparent movable line at the window top)
      return !IsWindowHasTitle();
    }
  },
  watch: {
    isLoggedIn() {
      if (this.isLoggedIn === false) this.$router.push("/");
    }
  }
};

// Ability to get working Copy\Paste to 'input' elements
// without modification application menu (which is required for macOS)
if (Platform() === PlatformEnum.macOS) {
  const { clipboard } = require("electron");
  const keyCodes = {
    V: 86
  };
  document.onkeydown = function(event) {
    let toReturn = true;
    if (event.ctrlKey || event.metaKey) {
      // detect ctrl or cmd
      if (event.which == keyCodes.V) {
        document.activeElement.value += clipboard.readText();
        document.activeElement.dispatchEvent(new Event("input"));
        toReturn = false;
      }
    }
    return toReturn;
  };
}
</script>

<style lang="scss">
@import "@/components/scss/constants";

html * {
  // disable elements\text selelection
  -webkit-user-select: none;
  // Window can be dragable by mouse from any place
  //-webkit-app-region: drag;

  // assign default properties globally for all elements
  color: $base-text-color;

  font-family: $base-font-family; // !important;
}

#app {
  position: absolute;
  left: 0;
  top: 0;
  width: 100vw;
  height: 100vh;

  // disable scroolbars (Windows)
  overflow-y: hidden;
}

.title {
  // Panel can be dragable by mouse
  // (we need this because using no title style for main window (for  macOS))
  -webkit-app-region: drag;
  height: 24px;
  width: 100%;

  position: absolute;
  z-index: 5;
}
</style>
