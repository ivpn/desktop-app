<template>
  <div id="main">
    <div v-if="isErrorState" style="color: grey">{{ errorMessage }}</div>
    <div v-else id="progressBar">
      <div id="progress" ref="progress"></div>
    </div>
  </div>
</template>

<script>
import { AppUpdateStage } from "@/store/types";

export default {
  data: function() {
    return {};
  },
  mounted() {
    this.updateProgressBarState();
  },
  methods: {
    updateProgressBarState: function() {
      if (!this.$refs.progress || !this.$refs.progress.style) return;
      this.$refs.progress.style.width =
        (Number(this.downloaded) / Number(this.contentLength)) * 100 + "%";
    }
  },
  computed: {
    updateProgress: function() {
      if (!this.$store.state.uiState) return null;
      return this.$store.state.uiState.appUpdateProgress;
    },
    state: function() {
      if (!this.updateProgress) return null;
      return this.updateProgress.state;
    },
    errorMessage: function() {
      if (!this.isErrorState || !this.updateProgress) return null;
      let msg = this.updateProgress.error;
      if (!msg) return "Update failed";
      return msg;
    },
    isErrorState: function() {
      if (this.state == AppUpdateStage.Error) return true;
      return false;
    },

    downloaded: function() {
      if (!this.state || !this.updateProgress) return 0;
      if (this.state == AppUpdateStage.ReadyToInstall) return 1; // for 'ReadyToInstall' downloaded and contentLength must be same
      if (!this.updateProgress.downloadStatus) return 0;
      return this.updateProgress.downloadStatus.downloaded;
    },
    contentLength: function() {
      if (!this.state || !this.updateProgress) return 0;
      if (this.state == AppUpdateStage.ReadyToInstall) return 1; // for 'ReadyToInstall' downloaded and contentLength must be same
      if (!this.updateProgress.downloadStatus) return 0;
      return this.updateProgress.downloadStatus.contentLength;
    }
  },
  watch: {
    downloaded() {
      this.updateProgressBarState();
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

#progressBar {
  width: 100%;
  background-color: #ddd;
  border-radius: 4px;
}

#progress {
  width: 0%;
  height: 8px;
  background-color: #398fe6;
  border-radius: 4px;
}
</style>
