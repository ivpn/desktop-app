<template>
  <div>
    <div class="settingsTitle">SPLIT TUNNEL SETTINGS</div>

    <div class="param">
      <input type="checkbox" id="isSTEnabled" v-model="isSTEnabled" />
      <label class="defColor" for="isSTEnabled">Split Tunnel</label>
    </div>
    <div class="fwDescription">
      By enabling this feature you can exclude traffic of specific applications
      from the VPN tunnel
    </div>

    <!-- APPS -->
    <div v-if="!isActionsView" class="flexColumn">
      <div class="flexRow" style="margin-top: 12px; margin-bottom:12px">
        <div
          class="flexRowRestSpace settingsBoldFont"
          style="margin-top: 0px; margin-bottom:0px"
        >
          Applications
        </div>
        <div>
          <button
            class="settingsButton"
            style="min-width: 60px"
            v-on:click="addNewApplication"
          >
            Add ...
          </button>
        </div>
      </div>

      <div class="horizontalLine" />

      <div
        class="scrollableColumnContainer"
        style="padding:1px; margin-top: 8px; margin-bottom:8px; max-height: 320px;  height: 320px;"
      >
        <div v-for="path of apps" v-bind:key="path">
          <div
            class="flexRow visibleOnHoverParent"
            style="margin-top: 2px; margin-bottom: 2px"
          >
            <div class="flexRowRestSpace">
              {{ path }}
            </div>
            <button
              class="settingsButton visibleOnHover"
              style="min-width: 60px"
              v-on:click="removeApp(path)"
            >
              remove
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  data: function() {
    return {};
  },
  methods: {
    async addNewApplication() {
      var diagConfig = {
        properties: ["openFile"],
        filters: [
          { name: "Executables", extensions: ["exe"] },
          { name: "All files", extensions: ["*"] }
        ]
      };
      var ret = await sender.showOpenDialog(diagConfig);

      if (!ret || ret.canceled || ret.filePaths.length == 0) return;

      var st = this.$store.state.settings.splitTunnelling;
      if (!st.apps) st.apps = [];

      ret.filePaths.forEach(appPath => {
        if (st.apps.includes(appPath) == false) st.apps.push(appPath);
      });

      await sender.SplitTunnelSetConfig(st.enabled, st.apps);
    },

    async removeApp(appPath) {
      var st = this.$store.state.settings.splitTunnelling;
      if (!st.apps) return;
      var index = st.apps.indexOf(appPath);
      if (index === -1) return;

      st.apps.splice(index, 1);
      await sender.SplitTunnelSetConfig(st.enabled, st.apps);
    }
  },
  computed: {
    isSTEnabled: {
      get() {
        return this.$store.state.settings.splitTunnelling.enabled;
      },
      async set(value) {
        var st = this.$store.state.settings.splitTunnelling;
        st.enabled = value;

        await sender.SplitTunnelSetConfig(st.enabled, st.apps);
      }
    },
    apps: function() {
      return this.$store.state.settings.splitTunnelling.apps;
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.visibleOnHoverParent > .visibleOnHover {
  visibility: hidden;
}
.visibleOnHoverParent:hover > .visibleOnHover {
  visibility: visible;
}

.defColor {
  @extend .settingsDefaultTextColor;
}

div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
  max-width: 425px;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

button.link {
  @extend .noBordersTextBtn;
  @extend .settingsLinkText;
  font-size: inherit;
}
label {
  margin-left: 1px;
  font-weight: 500;
}
</style>
