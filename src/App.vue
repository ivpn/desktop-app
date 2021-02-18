<template>
  <div id="app">
    <!-- ability to move by mouse when no title for window (macOS) -->
    <div class="title" v-if="isShowDragableTitle"></div>

    <router-view />
  </div>
</template>

<script>
const sender = window.ipcSender;
import { IsWindowHasTitle } from "@/platform/platform";
import { InitDefaultCopyMenus } from "@/context-menu/renderer";

export default {
  beforeCreate() {
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
</style>
