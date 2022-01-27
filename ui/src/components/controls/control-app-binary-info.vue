<template>
  <div class="flexRow">
    <binaryIconControl
      :binary-path="app.AppBinaryPath"
      :preloaded-base64-icon="app.AppIcon"
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
      <div class="text">
        {{ app.AppName }}
      </div>

      <div
        v-if="app.AppName != app.AppGroup && !app.RunningApp"
        class="settingsGrayLongDescriptionFont text"
      >
        {{ app.AppGroup }}
      </div>
      <div
        v-else-if="app.RunningApp && app.RunningApp.Pid"
        class="settingsGrayLongDescriptionFont text"
      >
        [ PID: {{ app.RunningApp.Pid }} ] {{ app.AppGroup }}
      </div>
    </div>
  </div>
</template>

<script>
import binaryIconControl from "@/components/controls/control-app-binary-icon.vue";

export default {
  components: {
    binaryIconControl,
  },
  props: [
    // App:
    //    AppName       string
    //    AppGroup      string // optional
    //    AppIcon       string - base64 icon of the executable binary
    //    AppBinaryPath string - The unique parameter describing an application
    //                    Windows: absolute path to application binary
    //                    Linux: program to execute, possibly with arguments.
    "app",
  ],
  methods: {},
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
