<template>
  <div>
    <label class="switch" v-bind:class="{ load: isProgress }">
      <input
        type="checkbox"
        :checked="isConnected"
        v-on:click="DoSwitch($event)"
      />
      <div :style="style"></div>
    </label>
  </div>
</template>

<script>
export default {
  props: ["onChecked", "isChecked", "isProgress", "checkedColor"],
  computed: {
    isConnected: function() {
      if (this.isProgress) return false;
      return this.isChecked === true;
    },
    style: function() {
      if (this.checkedColor == null || this.isConnected === false) return "";
      return `background: ${this.checkedColor}`;
    }
  },

  methods: {
    DoSwitch(e) {
      if (!this.isConnected) e.preventDefault();
      if (this.onChecked) {
        if (this.isProgress) this.onChecked(false, e);
        else this.onChecked(!this.isConnected, e);
      }
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
$switchSize: 18px;
$wToHproportion: 2;
$switchBorder: transparent; //#d1d7e3;
$switchBorderActive: transparent; //#5d9bfb;
$switchAnimationPieBorder: #449cf8;

$switchBackground: #ff6258; //#d1d7e3;
// $switchBackground: #d1d7e3; //#d1d7e3;
$switchDot: var(--background-color); // #fff;
$switchActive: #449cf8;
// $switchActive: #449cf8;

.switch {
  margin: 0;
  cursor: pointer;
  & > span {
    line-height: $switchSize;
    margin: 0 0 0 4px;
    vertical-align: top;
  }
  input {
    display: none;
    & + div {
      width: $switchSize * 1.6; // !!!
      height: $switchSize;
      border: 1px solid $switchBorder;
      background: $switchBackground;
      border-radius: $switchSize / 2;
      vertical-align: top;
      position: relative;
      display: inline-block;
      user-select: none;
      transition: all 0.4s ease;
      &:before {
        content: "";
        float: left;
        width: $switchSize - 6;
        height: $switchSize - 6;
        background: $switchDot;
        pointer-events: none;
        margin-top: 2px;
        margin-left: 2px;
        border-radius: inherit;
        transition: all 0.4s ease 0s;
      }
      &:after {
        content: "";
        left: -1px;
        top: -1px;
        width: $switchSize;
        height: $switchSize;
        border: 3px solid transparent;
        border-top-color: $switchAnimationPieBorder; //$switchBorderActive;
        border-radius: 50%;
        position: absolute;
        opacity: 0;
      }
    }
    &:checked + div {
      background: $switchActive;
      border: 1px solid $switchBorderActive;
      &:before {
        transform: translate($switchSize/1.6, 0); // !!!
      }
    }
  }
  &.load {
    input {
      & + div {
        width: $switchSize;
        margin: 0 $switchSize / 4; // !!!
        &:after {
          opacity: 1;
          animation: rotate 0.9s infinite linear;
          animation-delay: 0.2s;
        }
      }
    }
  }
  &:hover {
    input {
      & + div {
        opacity: 0.7;
      }
    }
  }
}

@keyframes rotate {
  0%,
  15% {
    transform: rotate(0deg);
  }
  50% {
    transform: rotate(290deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

div {
  box-sizing: border-box;
}

* {
  box-sizing: inherit;
  &:before,
  &:after {
    box-sizing: inherit;
  }
}
</style>
