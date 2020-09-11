<template>
  <div class="flexRow">
    <img src="@/assets/wifi.svg" />
    <div class="text">
      {{ wifiInfo.ssid }}
    </div>

    <div style="flex-grow: 1"></div>

    <select
      class="trustedConfigBase"
      v-bind:class="{
        trustedConfigUntrusted: isTrusted == false,
        trustedConfigTrusted: isTrusted == true
      }"
      v-model="isTrusted"
    >
      <option value="false">Untrusted</option>
      <option value="true">Trusted</option>
      <option value="null">No status</option>
    </select>
  </div>
</template>

<script>
export default {
  props: ["wifiInfo", "onChange"],
  computed: {
    isTrusted: {
      get() {
        return this.wifiInfo.isTrusted;
      },
      set(value) {
        let v = null;
        switch (value) {
          case "false":
            v = false;
            break;
          case "true":
            v = true;
            break;
        }
        if (this.onChange != null) this.onChange(this.wifiInfo.ssid, v);
      }
    }
  },
  methods: {}
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

.flagBigger {
  width: 26px;
}

.text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;

  font-size: 13px;
  line-height: 16px;

  letter-spacing: -0.078px;

  padding: 8px;
}

select.trustedConfigBase {
  min-width: 90px;
  border-width: 0px;
}

select.trustedConfigUntrusted {
  @extend .trustedConfigBase;
  color: red;
}
select.trustedConfigTrusted {
  @extend .trustedConfigBase;
  color: #3b99fc;
}
</style>
