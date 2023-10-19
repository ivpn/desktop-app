<template>
  <div class="defaultMainDiv">
    <!--
    <div class="settingsBoldFont" style="margin-top: 0px; margin-bottom: 12px">
      New port properties
    </div>
    -->

    <!-- Port number -->
    <div class="flexRow">
      <div class="paramName">Port:</div>
      <input
        ref="portField"
        type="number"
        style="flex-grow: 1; min-width: 286px"
        class="settingsTextInput"
        :placeholder="numberPlaceholder"
        :title="`Allowed port range: ${numberPlaceholder}`"
        v-model="portNumber"
        v-on:keyup.enter="onApply()"
      />
    </div>

    <!-- Port type -->
    <div
      class="flexRow"
      v-bind:class="{
        disabled: isSupportedTCP !== true || isSupportedUDP !== true,
      }"
      style="margin-top: 6px"
    >
      <div class="paramName">Protocol:</div>
      <div style="margin-right: 12px">
        <input
          type="radio"
          id="typeUDP"
          name="type"
          v-model="type"
          value="UDP"
        />
        <label class="defColor" for="typeUDP">UDP</label>
      </div>
      <div>
        <input
          type="radio"
          id="typeTCP"
          name="type"
          v-model="type"
          value="TCP"
        />
        <label class="defColor" for="typeTCP">TCP</label>
      </div>
    </div>

    <!-- Footer buttons -->
    <div class="flexRow" style="margin-top: 10px">
      <div style="flex-grow: 1"></div>
      <div class="flexRow">
        <button
          class="slave"
          style="height: 28px; min-width: 100px"
          v-on:click="onCancel()"
        >
          Cancel
        </button>

        <button
          class="master"
          style="height: 28px; min-width: 100px; margin-left: 12px"
          v-on:click="onApply()"
        >
          Add
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { PortTypeEnum, V2RayObfuscationEnum } from "@/store/types";
import { SetInputFilterNumbers } from "@/helpers/renderer";

const sender = window.ipcSender;

export default {
  props: {
    onClose: Function,
  },
  mounted() {
    this.initializeType();
    if (this.$refs.portField) this.$refs.portField.focus();
    SetInputFilterNumbers(this.$refs.portField);
  },
  data: function () {
    return {
      type: "UDP",
      portNumber: null,
    };
  },

  watch: {
    isSupportedTCP() {
      this.initializeType();
    },
    isSupportedUDP() {
      this.initializeType();
    },
  },

  computed: {
    ranges: function () {
      let ranges = this.$store.getters["vpnState/portRanges"];

      // The V2Ray service listens on it’s own IP address using the same ports as WireGuard.
      // It listens on both TCP and UDP ports.
      let updatePortType = null;
      const V2RayType = this.$store.getters["settings/getV2RayConfig"];
      if (V2RayType === V2RayObfuscationEnum.QUIC) {
        updatePortType = PortTypeEnum.UDP;
      } else if (V2RayType === V2RayObfuscationEnum.TCP) {
        updatePortType = PortTypeEnum.TCP;
      }
      if (updatePortType != null) {
        ranges = ranges.map((r) => {
          r.type = updatePortType;
          return r;
        });
      }

      return ranges;
    },

    portType: function () {
      return this.type == "TCP" ? PortTypeEnum.TCP : PortTypeEnum.UDP;
    },

    isSupportedTCP: function () {
      const pos = this.ranges.find((r) => r.type === PortTypeEnum.TCP);
      return pos != undefined;
    },
    isSupportedUDP: function () {
      const pos = this.ranges.find((r) => r.type === PortTypeEnum.UDP);
      return pos != undefined;
    },

    numberPlaceholder: function () {
      try {
        let retPlaceholder = "";

        // filter required ranges (by port type)
        let ranges = JSON.parse(JSON.stringify(this.ranges)); // clone array

        ranges = ranges.filter(
          (r) =>
            r.type === this.portType && r.range && r.range.min <= r.range.max
        );

        // Trying to union ranges (when it is possible)
        // E.g. : [40000-49999], [50000-59999] => [40000-59999]

        // sort
        ranges.sort((a, b) => {
          if (a.range.min < b.range.min) return -1;
          if (a.range.min > b.range.min) return 1;
          return 0;
        });

        let newRanges = [];
        if (ranges.length <= 1) {
          newRanges = ranges;
        } else {
          let r = Object.assign({}, ranges[0]);
          for (let i = 1; i < ranges.length; i++) {
            let rCur = Object.assign({}, ranges[i]);

            if (r.range.max >= rCur.range.min - 1) {
              if (rCur.range.max >= r.range.max) r.range.max = rCur.range.max;
            } else {
              newRanges.push(r);
              r = Object.assign({}, rCur);
            }
          }
          if (r) newRanges.push(r);
        }

        // make final string
        newRanges.forEach((r) => {
          if (r.type !== this.portType || !r.range) return;
          if (retPlaceholder.length > 0) retPlaceholder += ", ";
          retPlaceholder += `${r.range.min} - ${r.range.max}`;
        });
        return retPlaceholder;
      } catch (e) {
        console.error(e);
        return "";
      }
    },
  },

  methods: {
    initializeType() {
      if (this.isSupportedUDP !== true && this.isSupportedTCP === true)
        this.type = "TCP";
      else if (this.isSupportedUDP === true && this.isSupportedTCP !== true)
        this.type = "UDP";
    },
    async parseAndGetPortObj() {
      // check if port defined
      if (!this.portNumber) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Port number is not defined",
          detail: `Please, enter port number`,
        });
        return null;
      }

      const portNumVal = parseInt(this.portNumber, 10);
      if (!portNumVal) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Bad data",
          detail: `Please, enter port number`,
        });
        return null;
      }

      // check port range
      const rPos = this.ranges.find(
        (r) => portNumVal >= r.range.min && portNumVal <= r.range.max
      );
      if (!rPos) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Port number does not fit the acceptable range",
          detail:
            `Please, enter port number in the range: \n` +
            this.numberPlaceholder,
        });
        return null;
      }

      // check if port already exists
      const ports = this.$store.getters["vpnState/connectionPorts"];
      const pPos = ports.find(
        (p) => p.type === this.portType && p.port === portNumVal
      );
      if (pPos) {
        await sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Port already exists",
          detail: `Port '${this.type} ${portNumVal}' already exists`,
        });
        return null;
      }

      return {
        port: portNumVal,
        type: this.portType,
      };
    },

    close() {
      if (this.onClose) this.onClose();
    },
    onCancel() {
      this.close();
    },
    async onApply() {
      try {
        const newPort = await this.parseAndGetPortObj();
        if (newPort) {
          console.log("New port added: ", newPort);
          this.$store.dispatch("settings/addNewCustomPort", newPort);
          this.close();
        }
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to define new port`,
          detail: e,
        });
      }
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.paramName {
  min-width: 80px;
}

.description {
  @extend .settingsGrayLongDescriptionFont;
  margin-left: 80px;
}
</style>
