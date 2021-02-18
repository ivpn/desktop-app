<template>
  <div class="flexColumn">
    <div class="settingsTitle">VERSION</div>
    <spinner :loading="isChecking" />
    <!-- VERSION INFO -->
    <div v-if="isGenericUpdater" class="flexRow">
      <!-- generic -->
      <div class="settingsGrayDescriptionFont verInfoCell">
        IVPN

        <label class="selectable" style="margin-left: 20px"
          >{{ daemonVersionInfo }}
        </label>

        <label
          style="margin-left: 10px"
          class="verInfoCell settingsGrayDescriptionFont"
        >
          {{ latestGenericVersionInfo }}
        </label>
      </div>
    </div>
    <div v-else class="flexRow">
      <!-- saparate daemon and UI -->
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

    <div
      v-if="isAbleToCheckUpdate"
      style="margin-top: 20px; margin-buttom:20px;"
    >
      <div class="flexRow">
        <div>
          <!-- CHECK UPDATES BUTTON -->
          <button
            v-if="!isHasUpgrade"
            class="slave btn"
            v-on:click="onCheckUpdatesPressed"
          >
            {{ updateBtnText }}
          </button>

          <div v-else>
            <!-- UPGRADE BUTTON -->
            <button
              v-if="isHasUpgrade && !isHasDownloadState"
              class="master btn"
              v-on:click="onUpgradePressed"
            >
              Update
            </button>

            <div v-else>
              <!-- CANCEL DOWNLOAD BUTTON -->
              <button
                v-if="isDownloading"
                class="slave btn"
                v-on:click="onCancelDownloadPressed"
              >
                Cancel
              </button>
              <!-- INSTALL BUTTON -->
              <button
                v-else-if="isReadyToInstall"
                class="master btn"
                v-on:click="onInstallPressed"
              >
                Install
              </button>
              <!-- INSTALLING BUTTON -->
              <button
                v-if="isInstalling"
                class="slave btn"
                style="background: transparent; color: grey; birder: 0px"
              >
                Installing ...
              </button>
            </div>
          </div>
        </div>

        <UpdateProgress
          v-if="isHasDownloadState || isErrorState"
          class="flexRowRestSpace"
          style="margin-left: 10px;"
        />
      </div>
    </div>

    <!-- RELEASE NOTES -->
    <div
      v-if="isAbleToCheckUpdate && isHasUpgrade"
      class="scrollableColumnContainer"
      style="min-height: 0px; margin-top: 20px"
    >
      <div v-if="isGenericUpdater">
        <!-- generic -->

        <div v-if="genericReleaseNotes && isHasUpgrade">
          <div class="boldFont">Release notes</div>
          <div
            v-for="note of genericReleaseNotes"
            v-bind:key="note.description"
          >
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
      <div v-else>
        <!-- saparate daemon and UI -->
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
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;
import spinner from "@/components/controls/control-spinner.vue";
import UpdateProgress from "@/components/UpdateProgress.vue";
import { IsNewVersion } from "@/app-updater/helper";
import { IsGenericUpdater } from "@/app-updater";
import { AppUpdateStage } from "@/store/types";

export default {
  components: {
    UpdateProgress,
    spinner
  },
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
        // You already have the latest version installed (v3.2.40)
        this.isChecking = false;

        if (!this.isHasUpgrade) {
          sender.showMessageBox({
            type: "info",
            buttons: ["OK"],
            message: "Nothing to update.",
            detail: `You already have the latest version installed!\n\nDaemon: ${this.daemonVersionInfo}\nUI: ${this.uiVersionInfo}`
          });
        }
      }
    },
    onUpgradePressed: async function() {
      await sender.AppUpdatesUpgrade();
    },
    onCancelDownloadPressed: async function() {
      await sender.AppUpdatesCancelDownload();
    },
    onInstallPressed: async function() {
      await sender.AppUpdatesInstall();
    }
  },
  computed: {
    isGenericUpdater: function() {
      return IsGenericUpdater();
    },
    updateBtnText: function() {
      if (this.isChecking === true) return "Checking ...";
      return "Check for Updates";
    },
    isAbleToCheckUpdate: function() {
      let ret = sender.AppUpdatesIsAbleToUpdate();
      return ret;
    },
    isHasUpgrade: function() {
      if (this.isGenericUpdater) return this.latestVersionGenericHasUpdate;
      return this.latestUiVersionHasUpdate || this.latestDaemonVersionHasUpdate;
    },

    updateProgress: function() {
      if (!this.$store.state.uiState) return null;
      return this.$store.state.uiState.appUpdateProgress;
    },
    state: function() {
      if (!this.updateProgress) return null;
      return this.updateProgress.state;
    },
    isDownloading: function() {
      return (
        this.state == AppUpdateStage.Downloading ||
        this.state == AppUpdateStage.CheckingSignature
      );
    },
    isReadyToInstall: function() {
      return this.state == AppUpdateStage.ReadyToInstall;
    },
    isInstalling: function() {
      return this.state == AppUpdateStage.Installing;
    },
    isHasDownloadState: function() {
      if (!this.state || this.isErrorState) return false;
      return true;
    },
    isErrorState: function() {
      if (this.state == AppUpdateStage.Error) return true;
      return false;
    },
    // CIRRENT VERSIONS
    uiVersion: function() {
      return sender.appGetVersion();
    },
    daemonVersion: function() {
      return this.$store.state.daemonVersion;
    },
    // CIRRENT VERSION NOTIFICATION TEXT
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

    // LATEST VERSIONS
    latestVersionGeneric: function() {
      try {
        return this.$store.state.latestVersionInfo.generic.version;
      } catch {
        return null;
      }
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

    // LATEST VERSIONS 'IS HAS UPDATE?' true/fase
    latestVersionGenericHasUpdate: function() {
      if (!this.latestVersionGeneric || !this.uiVersion) return false;
      if (IsNewVersion(this.uiVersion, this.latestVersionGeneric)) return true;
      return false;
    },
    latestUiVersionHasUpdate: function() {
      if (!this.latestUiVersion || !this.uiVersion) return false;
      if (IsNewVersion(this.uiVersion, this.latestUiVersion)) return true;
      return false;
    },
    latestDaemonVersionHasUpdate: function() {
      if (!this.latestDaemonVersion || !this.daemonVersion) return false;
      if (IsNewVersion(this.daemonVersion, this.latestDaemonVersion))
        return true;
      return false;
    },

    // LATEST VERSIONS NOTIFICATION TEXT
    latestGenericVersionInfo: function() {
      if (!this.latestVersionGeneric) return "";
      if (this.latestVersionGenericHasUpdate)
        return `New version available (v${this.latestVersionGeneric})`;
      return `You already have latest version!`;
    },
    latestUiVersionInfo: function() {
      if (!this.latestUiVersion) return "";
      if (this.latestVersionGenericHasUpdate)
        return `New version available (v${this.latestUiVersion})`;
      return `You already have latest version!`;
    },
    latestDaemonVersionInfo: function() {
      if (!this.latestDaemonVersion) return "";
      if (this.latestDaemonVersionHasUpdate)
        return `New version available (v${this.latestDaemonVersion})`;
      return `You already have latest version!`;
    },

    genericReleaseNotes: function() {
      try {
        const rn = this.$store.state.latestVersionInfo.generic.releaseNotes;
        if (rn.length == 0) return null;
        return rn;
      } catch {
        return null;
      }
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
  width: 150px;
  height: 30px;
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
