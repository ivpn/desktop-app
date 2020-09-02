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

if (Platform() === PlatformEnum.macOS) {
  const electron = require("electron");
  const remote = electron.remote;
  const Menu = remote.Menu;

  // Default COPY/PASTE contect menu for all imput elements (macOS only)
  const InputMenu = Menu.buildFromTemplate([
    {
      label: "Undo",
      role: "undo"
    },
    {
      label: "Redo",
      role: "redo"
    },
    {
      type: "separator"
    },
    {
      label: "Cut",
      role: "cut"
    },
    {
      label: "Copy",
      role: "copy"
    },
    {
      label: "Paste",
      role: "paste"
    },
    {
      type: "separator"
    },
    {
      label: "Select all",
      role: "selectall"
    }
  ]);

  document.body.addEventListener("contextmenu", e => {
    e.preventDefault();
    e.stopPropagation();

    let node = e.target;

    while (node) {
      if (
        node.nodeName.match(/^(input|textarea)$/i) ||
        node.isContentEditable
      ) {
        InputMenu.popup(remote.getCurrentWindow());
        break;
      }
      node = node.parentNode;
    }
  });

  // Ability to get working Copy\Paste to 'input' elements
  // without modification application menu (which is required for macOS)
  const { clipboard } = require("electron");
  const keyCodes = {
    V: 86,
    C: 67,
    X: 88,
    A: 65
  };
  document.onkeydown = function(event) {
    let toReturn = true;
    if (event.ctrlKey || event.metaKey) {
      console.log(event);
      // detect ctrl or cmd
      if (event.which == keyCodes.A) {
        const field = document.activeElement;
        if (field != null) field.select();
        toReturn = false;
      } else if (event.which == keyCodes.V) {
        const field = document.activeElement;
        if (field != null) {
          const startPos = field.selectionStart;
          const endPos = field.selectionEnd;

          const text = clipboard.readText();

          field.value =
            field.value.substring(0, startPos) +
            text +
            field.value.substring(endPos, field.value.length);

          field.focus();
          field.setSelectionRange(
            startPos + text.length,
            startPos + text.length
          );

          toReturn = false;
        }
      } else if (event.which == keyCodes.C) {
        clipboard.writeText(getSelection().toString());
        toReturn = false;
      } else if (event.which == keyCodes.X) {
        const field = document.activeElement;
        if (field != null) {
          let selection = getSelection();
          clipboard.writeText(selection.toString());

          const startPos = field.selectionStart;
          const endPos = field.selectionEnd;

          field.value =
            field.value.slice(0, startPos) + field.value.slice(endPos);

          field.focus();
          field.setSelectionRange(startPos, startPos);

          toReturn = false;
        }
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
