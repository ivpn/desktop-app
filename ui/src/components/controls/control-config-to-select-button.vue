<template>
  <div>
    <button id="selectBtn" class="flexRow" v-on:click="click">
      <div align="left" class="flexRowRestSpace">
        <div class="small_text">{{ description }}</div>
        <div style="height: 5px" />
        <div class="large_text">{{ text }}</div>
      </div>

      <div v-if="isShowMarker" style="margin-right: 12px">
        <div :style="markerStyle">
          {{ this.markerText }}
        </div>
      </div>
      <div class="arrowRightSimple"></div>
    </button>
  </div>
</template>

<script>
export default {
  props: [
    "click",
    "text",
    "description",
    "markerText",
    "markerColor",
    "markerTextColor",
  ],
  computed: {
    isShowMarker: function () {
      return this.markerText != null && this.markerColor != null;
    },
    markerStyle: function () {
      let ret = `background: ${this.markerColor};  border-radius: 4px; padding: 4px; font-size: 10px; line-height: 10px; letter-spacing: 1px; `;
      if (this.markerTextColor) ret += `color: ${this.markerTextColor}`;
      return ret;
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

#selectBtn {
  padding: 0px;

  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  display: flex;
  justify-content: space-between;
  align-items: center;

  width: 100%;
}
</style>
