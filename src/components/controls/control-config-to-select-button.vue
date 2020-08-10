<template>
  <div>
    <button id="selectBtn" v-on:click="click">
      <div align="left">
        <div class="small_text">{{ description }}</div>
        <div style="height: 5px" />
        <div class="large_text">{{ text }}</div>
      </div>
      <div class="selectArrow"></div>
    </button>
  </div>
</template>

<script>
export default {
  props: ["click", "text", "description"]
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
  color: $base-text-color-details;
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

.selectArrow {
  border: solid #8b9aab;
  border-width: 0 2px 2px 0;
  display: inline-block;
  padding: 4px;
  transform: rotate(-45deg);
  -webkit-transform: rotate(-45deg);
}
</style>
