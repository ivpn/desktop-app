<template>
  <div class="titleBar" v-show="!isWindowHasFrame">
    <button style="margin-top: 1px"
      v-if="minimizable"
      class="noBordersBtn winBtns winBtnMinimize"
      v-on:click="onMinimize()"
    >
      <svg width="11" height="1" viewBox="0 0 11 1">
        <path d="m11 0v1h-11v-1z" stroke-width=".26208" fill="#888888" />
      </svg>
    </button>

    <button style="margin-top: 1px; margin-right: 1px"
      v-if="closable"
      class="noBordersBtn winBtns winBtnClose"
      v-on:click="onClose()"
    >
      <svg width="11" height="11" viewBox="0 0 12 12" color="red">
        <path
          d="m6.8496 6 5.1504 5.1504-0.84961 0.84961-5.1504-5.1504-5.1504 5.1504-0.84961-0.84961 5.1504-5.1504-5.1504-5.1504 0.84961-0.84961 5.1504 5.1504 5.1504-5.1504 0.84961 0.84961z"
          stroke-width=".3"
          fill="#888888"
        />
      </svg>
    </button>
  </div>
</template>

<script>
import { IsWindowHasFrame } from "@/platform/platform";
const sender = window.ipcSender;

export default {
  data: function() {
    return {
      closable: true,
      maximizable: true,
      minimizable: true
    };
  },
  mounted() {
    let t = this;
    setTimeout(() => {
      const props = sender.getCurrentWindowProperties();
      if (!props) return;
      if (props.closable !== undefined) t.closable = props.closable;
      if (props.maximizable !== undefined) t.maximizable = props.maximizable;
      if (props.minimizable !== undefined) t.minimizable = props.minimizable;
    }, 0);
  },
  computed: {
    isWindowHasFrame: function() {
      return IsWindowHasFrame();
    }
  },
  watch: {},
  methods: {
    onMinimize: function() {
      sender.minimizeCurrentWindow();
    },
    onClose: function() {
      sender.closeCurrentWindow();
    }
  }
};
</script>

<style lang="scss">
@import "@/components/scss/constants";

.titleBar {
  // Panel can be dragable by mouse
  // (we need this because using no title style for main window (for  macOS))
  -webkit-app-region: drag;

  height: 24px;
  width: 100%;

  position: absolute;

  z-index: 100;

  display: flex;
  justify-content: flex-end;
}

.winBtns {
  z-index: 101;
  -webkit-app-region: no-drag;
  cursor: pointer;
  width: 46px;
}

.winBtnMinimize:hover {
  background: #e5e5e5;
  @media (prefers-color-scheme: dark) {
    background: #404040;
  }
}

.winBtnClose:hover {
  background: #e81123;
}
</style>
