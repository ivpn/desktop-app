<template>
  <div class="main flexColumn" style="margin: 0px">
    <div class="flexColumn" style="flex-grow: 1" />

    <div ref="contentdiv" class="flexColumn">
      <div class="main flexColumn">
        <div>
          <!-- CHECKING FOR UPDATE-->
          <div v-if="isCheckingUpdate">
            <div class="big_text">Checking for updates ...</div>
            <spinner :loading="isCheckingUpdate" />
          </div>
          <div v-else>
            <!-- UPDATE CHECK FAILED-->
            <div v-if="!latestVersionInfo">
              <div class="big_text">Check update failed</div>
              <div class="small_text">
                Check new version failed, please try again.
              </div>

              <div class="buttons">
                <button class="master btn" v-on:click="onCheckUpdates">
                  Retry
                </button>
                <button class="slave btn" v-on:click="onCancel">Close</button>
              </div>
            </div>
            <div v-else>
              <!-- NOTHING TO UPDATE-->
              <div v-if="!isHasUpgrade">
                <div>
                  <div class="big_text">
                    You already have the latest version installed!
                  </div>
                  <div class="small_text">
                    <div v-if="versionDaemon == versionUI">
                      v{{ versionDaemon }}
                    </div>
                    <div v-else>
                      (daemon v{{ versionDaemon }}; UI v{{ versionUI }})
                    </div>
                  </div>
                </div>
                <div class="buttons">
                  <button class="slave btn" v-on:click="onCancel">Close</button>
                </div>
              </div>
              <!-- NEW UPDATE -->
              <div v-else>
                <div class="big_text">
                  <div v-if="versionLatestIsBeta">
                    New Beta version available
                  </div>
                  <div v-else>New IVPN version available</div>
                </div>

                <!-- info: new version -->
                <div class="small_text">
                  <span>
                    <span v-if="versionLatestGeneric">
                      v{{ versionLatestGeneric }}
                    </span>
                    <span v-else-if="versionLatestDaemon == versionLatestUI">
                      v{{ versionLatestDaemon }}
                    </span>
                    <span v-else>
                      Daemon v{{ versionLatestDaemon }}; UI v{{
                        versionLatestUI
                      }}
                    </span>
                  </span>
                  <!-- info: version you have -->
                  <span style="color: grey">
                    <span
                      v-if="versionDaemon == versionUI"
                      style="color: inherit"
                    >
                      (you have v{{ versionDaemon }})
                    </span>
                    <span v-else style="color: inherit">
                      (you have: Daemon v{{ versionDaemon }}; UI v{{
                        versionUI
                      }})
                    </span>
                  </span>
                  <!-- BETA info -->
                  <div
                    v-if="versionLatestIsBeta"
                    style="text-align: left; margin-top: 10px"
                  >
                    <b><span style="color: red">Use at your own risk!</span></b>
                    <div style="margin-top: 10px">
                      You are receiving this notification because you have
                      "Notify beta version updates" enabled in the General
                      settings.
                    </div>
                  </div>
                </div>

                <!--release notes-->
                <div
                  class="releaseNotes scrollableColumnContainer"
                  style="max-height: 350px"
                >
                  <!--release notes GENERIC-->
                  <div v-if="versionLatestGeneric">
                    <div class="relNotesPreText">Release notes:</div>
                    <releaseNotes
                      :releaseNotes="latestVersionInfo.generic.releaseNotes"
                    >
                    </releaseNotes>
                  </div>
                  <div v-else>
                    <!--release notes DAEMON-->
                    <div v-if="latestVersionInfo.daemon.releaseNotes">
                      <div class="relNotesPreText">Daemon release notes:</div>
                      <releaseNotes
                        :releaseNotes="latestVersionInfo.daemon.releaseNotes"
                      >
                      </releaseNotes>
                    </div>
                    <!--release notes UI-->
                    <div v-if="latestVersionInfo.uiClient.releaseNotes">
                      <div class="relNotesPreText">UI app release notes:</div>
                      <releaseNotes
                        :releaseNotes="latestVersionInfo.uiClient.releaseNotes"
                      >
                      </releaseNotes>
                    </div>
                  </div>
                </div>

                <!--buttons-->
                <div class="buttons flexRow">
                  <!-- Downloading || ready to install -->
                  <div
                    v-if="isShowProgressBar || isReadyToInstall"
                    class="flexRow flexRowRestSpace"
                  >
                    <UpdateProgress
                      v-if="!isReadyToInstall"
                      class="flexRow flexRowRestSpace"
                    />
                    <div v-else class="flexRow flexRowRestSpace" />

                    <button
                      v-if="isReadyToInstall"
                      class="master btn"
                      v-on:click="onInstall"
                    >
                      Install
                    </button>
                    <button class="slave btn" v-on:click="onCancel">
                      Cancel
                    </button>
                  </div>
                  <!-- installing -->
                  <div
                    v-else-if="isInstalling"
                    class="small_text flexRow flexRowRestSpace"
                    style="text-align: center"
                  >
                    Installing ...
                  </div>
                  <!-- update available || error -->
                  <div v-else class="flexRow flexRowRestSpace">
                    <div
                      v-if="isUpdateError"
                      class="flexRow flexRowRestSpace small_text"
                      style="text-align: right; color: orange"
                    >
                      Update failed {{ updateErrorText }}
                    </div>
                    <div v-else class="flexRow flexRowRestSpace">
                      <button
                        class="slave btn"
                        style="margin: 0px"
                        v-on:click="onCancel"
                      >
                        Remind Me Later
                      </button>
                      <button class="slave btn" v-on:click="onSkipThisVersion">
                        Skip This Version
                      </button>
                      <div class="flexRow flexRowRestSpace" />
                    </div>
                    <button class="master btn" v-on:click="onUpgrade">
                      <span v-if="!isUpdateError" style="color: inherit">
                        Update
                      </span>
                      <span v-else style="color: inherit"> Retry </span>
                    </button>
                    <button
                      v-if="isUpdateError"
                      class="slave btn"
                      v-on:click="onCancel"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flexColumn" style="flex-grow: 1" />
  </div>
</template>

<script>
const sender = window.ipcSender;

import { nextTick } from "vue";
import spinner from "@/components/controls/control-spinner.vue";
import releaseNotes from "@/components/controls/control-release-notes.vue";
import UpdateProgress from "@/components/UpdateProgress.vue";
import { AppUpdateStage } from "@/store/types";

import { IsNewVersion } from "@/app-updater/helper";

export default {
  components: { spinner, releaseNotes, UpdateProgress },

  data: function () {
    return {
      lastWindowHeight: 0,
    };
  },

  mounted() {},
  updated: async function () {
    await nextTick(); // DOM is now updated
    this.updateWindowSize();
  },

  computed: {
    latestVersionInfo: function () {
      //return null;
      return this.$store.state.latestVersionInfo;
    },
    updateState: function () {
      return this.$store.state.uiState?.appUpdateProgress?.state;
    },
    isCheckingUpdate: function () {
      return this.updateState == AppUpdateStage.CheckingForUpdates;
    },
    isHasUpgrade: function () {
      //if (this != "TEST") return false;
      if (this.versionLatestGeneric) {
        return (
          IsNewVersion(this.versionDaemon, this.versionLatestGeneric) ||
          IsNewVersion(this.versionUI, this.versionLatestGeneric)
        );
      }
      if (this.versionLatestUI && this.versionLatestDaemon) {
        return (
          IsNewVersion(this.versionDaemon, this.versionLatestDaemon) ||
          IsNewVersion(this.versionUI, this.versionLatestUI)
        );
      }
      return false;
    },
    isShowProgressBar: function () {
      let s = this.updateState;
      return (
        s == AppUpdateStage.Downloading || s == AppUpdateStage.CheckingSignature
      );
    },
    isDownloading: function () {
      return this.updateState == AppUpdateStage.Downloading;
    },
    isReadyToInstall: function () {
      return this.updateState == AppUpdateStage.ReadyToInstall;
    },
    isInstalling: function () {
      return this.updateState == AppUpdateStage.Installing;
    },
    isUpdateError: function () {
      return this.updateState == AppUpdateStage.Error;
    },
    updateErrorText: function () {
      if (!this.isUpdateError) return "";
      let err = this.$store.state.uiState?.appUpdateProgress?.error;
      if (!err) return null;
      return `(${err})`;
    },

    // ACTUAL VERSIONS
    versionSingle: function () {
      if (this.versionDaemon === this.versionUI) return this.versionDaemon;
      return null;
    },
    versionDaemon: function () {
      return this.$store.state.daemonVersion;
    },
    versionUI: function () {
      return sender.appGetVersion().Version;
    },
    // LATEST VERSIONS
    versionLatestGeneric: function () {
      return this.$store.state.latestVersionInfo?.generic?.version;
    },
    versionLatestUI: function () {
      return this.$store.state.latestVersionInfo?.uiClient?.version;
    },
    versionLatestDaemon: function () {
      return this.$store.state.latestVersionInfo?.daemon?.version;
    },
    versionLatestIsBeta: function () {
      return this.$store.state.latestVersionInfo?.beta === true ? true : false;
    },
  },

  methods: {
    onMainScrollChange() {
      this.updateWindowSize();
    },
    updateWindowSize() {
      let contentdiv = this.$refs.contentdiv;
      if (!contentdiv) return;
      let h = contentdiv.scrollHeight;
      if (!h || h < 150) return;
      if (this.lastWindowHeight == h) return;
      this.lastWindowHeight = h;
      sender.UpdateWindowResizeContent(0, h);
    },
    onCheckUpdates: async function () {
      await sender.AppUpdatesCheck();
    },
    onSkipThisVersion: function () {
      this.$store.dispatch("settings/skipAppUpdate", {
        genericVersion: this.versionLatestGeneric,
        daemonVersion: this.versionLatestDaemon,
        uiVersion: this.versionLatestUI,
      });

      this.onCancel();
    },
    onCancel: async function () {
      await sender.AppUpdatesCancelDownload();
      await sender.UpdateWindowClose();
    },
    onUpgrade: async function () {
      let isCanCloseWindow = await sender.AppUpdatesUpgrade();
      if (isCanCloseWindow === true) {
        await sender.UpdateWindowClose();
      }
    },
    onCancelDownload: async function () {
      await sender.AppUpdatesCancelDownload();
    },
    onInstall: async function () {
      await sender.AppUpdatesInstall();
    },
  },
  watch: {},
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";
.main {
  text-align: center;

  margin: 20px;
}

div.releaseNotes {
  @extend .settingsGrayDescriptionFont;
  margin-top: 20px;
  margin-bottom: 20px;
  text-align: left;
  font-size: 12px;
}

.btn {
  width: auto;
  height: 32px; //32px;

  padding-left: 20px;
  padding-right: 20px;
  margin-left: 10px;
}

.big_text {
  @extend .settingsBoldFont;
  margin: 20px;
  margin-bottom: 10px;
  font-size: 24px;
}
.small_text {
  @extend .settingsGrayDescriptionFont;
}

.fontBold {
  @extend .settingsBoldFont;
  margin: 0px;
}

.relNotesPreText {
  @extend .fontBold;
  margin-bottom: 10px;
}

div.buttons {
  margin-top: 20px;
  margin-bottom: 20px;
}
</style>
