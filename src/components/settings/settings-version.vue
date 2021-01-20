<template>
  <div class="flexColumn">
    <div class="settingsTitle">VERSION</div>
    <div class="flexRow">
      <div>
        <div class="flexRow">
          <div>
            <div class="settingsGrayDescriptionFont verInfoCell bottomMargin">
              IVPN Daemon
            </div>
            <div class="settingsGrayDescriptionFont verInfoCell">
              IVPN Client
            </div>
          </div>

          <div style="margin-left: 20px">
            <div class="verInfoCell bottomMargin">
              <label class="selectable">{{ daemonVersionInfo }}</label>
            </div>
            <div class="verInfoCell">
              <label class="selectable">{{ uiVersionInfo }}</label>
            </div>
          </div>

          <div
            style="margin-left: 10px"
            v-if="
              isAbleToCheckUpdate &&
                latestDaemonVersionInfo &&
                latestUiVersionInfo
            "
          >
            <div class="verInfoCell settingsGrayDescriptionFont bottomMargin">
              {{ latestDaemonVersionInfo }}
            </div>
            <div class="verInfoCell settingsGrayDescriptionFont">
              {{ latestUiVersionInfo }}
            </div>
          </div>
        </div>
      </div>

      <div class="flexRowRestSpace"></div>
    </div>

    <div>
      <button
        class="slave btn"
        v-if="isAbleToCheckUpdate && !isCanUpgrade"
        v-on:click="onCheckUpdatesPressed"
      >
        {{ updateBtnText }}
      </button>
    </div>

    <div
      v-if="
        isAbleToCheckUpdate &&
          isCanUpgrade &&
          (latestUiVersionHasUpdate || latestDaemonVersionHasUpdate)
      "
      class="scrollableColumnContainer"
      style="min-height: 0px; margin-top: 20px"
    >
      <div v-if="daemonReleaseNotes && latestDaemonVersionHasUpdate">
        <div class="boldFont">IVPN Daemon release notes</div>
        <div v-for="note of daemonReleaseNotes" v-bind:key="note.description">
          <div class="flexRow rnItem">
            <div
              class="badge"
              v-bind:class="{
                'badge-grey':
                  note.type && note.type.toLowerCase().startsWith('fix'),
                'badge-green':
                  note.type && note.type.toLowerCase().startsWith('new'),
                'badge-blue':
                  note.type && note.type.toLowerCase().startsWith('improve')
              }"
            >
              {{ note.type }}
            </div>
            {{ note.description }}
          </div>
        </div>
      </div>
      <div v-if="uiReleaseNotes && latestUiVersionHasUpdate">
        <div class="boldFont">IVPN Client release notes</div>
        <div v-for="note of uiReleaseNotes" v-bind:key="note.description">
          <div class="flexRow rnItem">
            <div
              class="badge"
              v-bind:class="{
                'badge-grey':
                  note.type && note.type.toLowerCase().startsWith('fix'),
                'badge-green':
                  note.type && note.type.toLowerCase().startsWith('new'),
                'badge-blue':
                  note.type && note.type.toLowerCase().startsWith('improve')
              }"
            >
              {{ note.type }}
            </div>
            {{ note.description }}
          </div>
        </div>
      </div>
    </div>

    <div class="flexRowSpace">
      <button
        class="master btn"
        v-if="isAbleToCheckUpdate && isCanUpgrade"
        v-on:click="onUpgradePressed"
      >
        Upgrade ...
      </button>
    </div>
  </div>
</template>

<script>
import sender from "@/ipc/renderer-sender";

import { IsNewerVersion } from "@/app-updater";

export default {
  data: function() {
    return {
      isChecking: false
    };
  },
  mounted() {},
  methods: {
    onCheckUpdatesPressed: async function() {
      this.isChecking = true;
      try {
        await sender.AppUpdatesCheck();
      } finally {
        this.isChecking = false;
      }
    },
    onUpgradePressed: async function() {
      await sender.AppUpdatesUpgrade();
    }
  },
  computed: {
    updateBtnText: function() {
      if (this.isChecking === true) return "Checking for updates ...";
      return "Check for updates";
    },
    isAbleToCheckUpdate: function() {
      let ret = sender.AppUpdatesIsAbleToUpdate();
      return ret;
    },

    uiVersion: function() {
      return require("electron").remote.app.getVersion();
    },
    daemonVersion: function() {
      return this.$store.state.daemonVersion;
    },

    latestUiVersion: function() {
      try {
        return this.$store.state.latestVersionInfo.uiClient.version;
      } catch {
        return null;
      }
    },
    latestDaemonVersion: function() {
      try {
        return this.$store.state.latestVersionInfo.daemon.version;
      } catch {
        return null;
      }
    },

    latestUiVersionHasUpdate: function() {
      if (!this.latestUiVersion || !this.uiVersion) return false;
      if (IsNewerVersion(this.uiVersion, this.latestUiVersion)) return true;
      return false;
    },
    latestDaemonVersionHasUpdate: function() {
      if (!this.latestDaemonVersion || !this.daemonVersion) return false;
      if (IsNewerVersion(this.daemonVersion, this.latestDaemonVersion))
        return true;
      return false;
    },

    uiVersionInfo: function() {
      let version = this.uiVersion;
      if (!version) return "unknown";
      return `v${version}`;
    },
    daemonVersionInfo: function() {
      let version = this.daemonVersion;
      if (!version) return "unknown";
      return `v${version}`;
    },

    latestUiVersionInfo: function() {
      if (!this.latestUiVersion || !this.uiVersion) return "";
      if (this.latestUiVersionHasUpdate)
        return `New version available (v${this.latestUiVersion})`;
      return `You already have latest version!`;
    },
    latestDaemonVersionInfo: function() {
      if (!this.latestDaemonVersion || !this.daemonVersion) return "";
      if (this.latestDaemonVersionHasUpdate)
        return `New version available (v${this.latestDaemonVersion})`;
      return `You already have latest version!`;
    },

    uiReleaseNotes: function() {
      try {
        const rn = this.$store.state.latestVersionInfo.uiClient.releaseNotes;
        if (rn.length == 0) return null;
        return rn;
      } catch {
        return null;
      }
    },
    daemonReleaseNotes: function() {
      try {
        const rn = this.$store.state.latestVersionInfo.daemon.releaseNotes;
        if (rn.length == 0) return null;
        return rn;
      } catch {
        return null;
      }
    },

    isCanUpgrade: function() {
      if (!this.latestUiVersion && !this.latestDaemonVersion) return false;

      return this.latestUiVersionHasUpdate || this.latestDaemonVersionHasUpdate;
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}
.verInfoCell {
  margin-right: 20px;
}

.bottomMargin {
  margin-bottom: 20px;
}

.btn {
  width: 280px;
  margin-top: 32px;
}

.rnItem {
  margin-top: 4px;
  margin-bottom: 4px;
}

.badge {
  border-radius: 4pt;
  height: 16pt;
  width: 60pt;
  font-size: 9pt;
  color: white;
  text-align: center;
  line-height: 17pt;
  margin-right: 15px;
  text-transform: capitalize;

  min-width: 80px;
}

.badge-grey {
  background-color: rgb(152, 165, 179);
}

.badge-green {
  background-color: rgb(33, 208, 116);
}

.badge-blue {
  background-color: rgb(57, 158, 230);
}

.boldFont {
  @extend .settingsBoldFont;
  margin-top: 20px;
}
</style>
