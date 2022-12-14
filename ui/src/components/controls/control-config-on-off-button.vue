<template>
  <div>
    <div class="switchButton">
      <div align="left">
        <button
          v-if="onTextClick"
          class="noBordersTextBtnEx"
          v-on:click="onTextClick"
          :title="textClickTooltip"
        >
          <div class="large_text">{{ text }}</div>
        </button>
        <div v-else class="large_text">{{ text }}</div>

        <div style="height: 5px" />
        <div class="small_text">
          {{ description }}
        </div>
      </div>

      <div>
        <SwitchProgress
          :style="switcherStyle"
          :onChecked="onChecked"
          :isChecked="isChecked"
          :isProgress="isProgress"
          :checkedColor="checkedColor"
        />
      </div>
    </div>
  </div>
</template>

<script>
import SwitchProgress from "@/components/controls/control-switch-small.vue";

export default {
  components: {
    SwitchProgress,
  },
  props: [
    "onChecked",
    "isChecked",
    "isProgress",
    "text",
    "description",
    "checkedColor",
    "switcherOpacity",
    "onTextClick",
    "textClickTooltip",
  ],
  computed: {
    switcherStyle: function () {
      if (!this.switcherOpacity) return "";
      return `opacity: ${this.switcherOpacity}`;
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.large_text {
  font-size: 14px;
  line-height: 17px;
}

.small_text {
  font-size: 11px;
  line-height: 13px;
  color: var(--text-color-details);
}

.switchButton {
  @extend .left_panel_block;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

button.noBordersTextBtnEx {
  @extend .noBordersTextBtn;

  display: flex;
  justify-content: space-between;
  align-items: center;

  width: 100%;
}
</style>
