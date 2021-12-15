<template>
  <div class="flexRow">
    <binaryIconControl
      :binaryPath="app.AppBinaryPath"
      :preloadedBase64Icon="app.AppIcon"
      style="
        min-width: 32px;
        min-height: 32px;
        max-width: 32px;
        max-height: 32px;
        padding: 4px;
      "
    />

    <!--Note: The width value (the style in element bellow) does not set real width
    On fact it can have any value.
    But if it is not defined - thw with of the element can grow outside of the window
     -->
    <div class="flexRowRestSpace text" style="padding-left: 5px; width: 200px">
      <!-- Manually added application -->
      <div v-if="!app.AppName">
        <div class="text">
          {{ getFileName(app.AppBinaryPath) }}
        </div>
        <div class="settingsGrayLongDescriptionFont text">
          {{ getFileFolder(app.AppBinaryPath) }}
        </div>
      </div>
      <div v-else>
        <!-- Application from the installed apps list (AppName and AppGroup is known)-->
        <div class="text">
          {{ app.AppName }}
        </div>
        <div
          class="settingsGrayLongDescriptionFont text"
          v-if="app.AppName != app.AppGroup"
        >
          {{ app.AppGroup }}
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import binaryIconControl from "@/components/controls/control-app-binary-icon.vue";

export default {
  props: ["app"],
  components: {
    binaryIconControl,
  },
  methods: {
    getFileFolder(filePath) {
      let fname = this.getFileName(filePath);
      if (!fname) return filePath;
      return filePath.substring(0, filePath.length - fname.length);
    },

    getFileName(filePath) {
      if (!filePath) return null;
      return filePath.split("\\").pop().split("/").pop();
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
@import "@/components/scss/platform/base.scss";

.text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
