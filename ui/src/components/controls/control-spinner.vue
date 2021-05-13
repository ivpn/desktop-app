<template>
  <transition name="fade">
    <div id="preloader" v-if="loading">
      <div id="loader" v-bind:style="spinnerStyle"></div>
    </div>
  </transition>
</template>

<script>
export default {
  props: {
    loading: {
      type: Boolean,
      default: false
    },
    color: {
      type: String,
      default: "#3385ff"
    },
    radius: {
      type: Number,
      default: 50
    }
  },
  data() {
    return {
      spinnerStyle: {
        "border-top-color": this.color,
        width: `${this.radius * 2}px`,
        height: `${this.radius * 2}px`,
        margin: `-${this.radius}px 0 0 -${this.radius}px`
      }
    };
  },
  computed: {},

  methods: {}
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
#preloader {
  //opacity: 0;
  background-color: #bbbbbb77;
  //background-color: #3385ff33;
  z-index: 100;

  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

#loader {
  display: block;
  position: relative;
  left: 50%;
  top: 50%;
  width: 100px;
  height: 100px;
  margin: -50px 0 0 -50px;
  border-radius: 50%;
  border: 2px solid transparent;
  //border-top-color: #9370db;
  //border-top-color: #66a3ff;
  //border-top-color: #a0a0a0;

  border-top-color: #3385ff;
  opacity: 1;

  animation: spin 2s linear infinite;
}

#loader:before {
  content: "";
  position: absolute;
  top: 5px;
  left: 5px;
  right: 5px;
  bottom: 5px;
  border-radius: 50%;
  border: 2px solid transparent;
  //border-top-color: #ba55d3;
  //border-top-color: #4d94ff;
  //border-top-color: #808080;

  border-top-color: inherit;
  opacity: 0.8;

  animation: spin 3s linear infinite;
}

#loader:after {
  content: "";
  position: absolute;
  top: 15px;
  left: 15px;
  right: 15px;
  bottom: 15px;
  border-radius: 50%;
  border: 2px solid transparent;
  //border-top-color: #ff00ff;
  //border-top-color: #3385ff;
  //border-top-color: #606060;

  border-top-color: inherit;
  opacity: 0.6;

  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
