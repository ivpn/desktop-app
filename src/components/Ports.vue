<template>
  <div id="main">
    <div class="flexRow">
      <button
        v-on:click="back"
        class="headerBtn"
        style="width: 35px; text-align: left;"
      >
        <img class="flag" src="@/assets/arrow-left-macos.svg" />
      </button>

      <button class="headerBtn headerBtnSelected">
        CONNECTION PORT
      </button>
    </div>

    <div id="list">
      <div key="list">
        <div v-for="port of ports" v-bind:key="portName(port)">
          <button class="selectBtn" v-on:click="onPortSelected(port)">
            <div align="left">
              <div>
                {{ portName(port) }}
              </div>
            </div>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { VpnTypeEnum, PortTypeEnum, Ports } from "@/store/types";
import { enumValueName } from "@/helpers/helpers";
export default {
  props: ["onBack", "isExitServer"],

  computed: {
    ports: function() {
      if (this.$store.state.settings.vpnType === VpnTypeEnum.OpenVPN)
        return Ports.OpenVPN;
      return Ports.WireGuard;
    }
  },

  watch: {},

  methods: {
    back() {
      this.onBack();
    },
    onPortSelected: function(port) {
      this.$store.dispatch("settings/setPort", port);
      this.back();
    },
    portName: function(port) {
      return `${enumValueName(PortTypeEnum, port.type)} ${port.port}`;
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
#main {
  display: flex;
  flex-flow: column;
  height: 100vh;
  color: #2a394b;
}

#list {
  overflow: auto;
}

.flexRow {
  display: flex;
  align-items: center;
}

.headerBtn {
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  width: 100%;
  height: 43px;

  font-style: normal;
  font-weight: 500;
  font-size: 11px;
  line-height: 13px;

  letter-spacing: 0.5px;
  text-transform: uppercase;

  color: #2a394b;

  border-bottom: 2px solid #d9e0e5;
}

.headerBtnSelected {
  border-bottom: 2px solid #449cf8;
}

.selectBtn {
  margin-left: 28px;
  border: none;
  background-color: inherit;
  outline-width: 0;
  cursor: pointer;

  height: 43px;
  width: 100%;
}
</style>
