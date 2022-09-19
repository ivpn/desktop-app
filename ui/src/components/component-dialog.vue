<template>
  <!--
  <div
    style="position: absolute; left: 0px; top: 0px; width: 100%; height: 100%"
  >
  -->
  <dialog
    class="dialogDefaults selectable"
    v-bind:class="{ center: center == true }"
    ref="dlgEl"
  >
    <!-- Header -->
    <div v-if="this.header && this.header.length > 0">
      <H3>{{ header }}</H3>
      <div class="horizontalLine" />
    </div>
    <div>
      <slot>
        <!-- here will be shown content of component -->
      </slot>
    </div>
    <!-- Footer -->
    <div>
      <form method="dialog" v-if="noCloseButtons !== true">
        <div class="flexRow" style="margin-top: 10px">
          <div style="flex-grow: 1"></div>
          <div class="flexRow">
            <button
              class="master"
              style="height: 28px; min-width: 100px; margin-left: 12px"
            >
              Close
            </button>
          </div>
        </div>
      </form>
    </div>
  </dialog>
  <!--
  </div>
  -->
</template>

<script>
export default {
  props: {
    header: String,
    center: Boolean,
    noCloseButtons: Boolean,
    onShow: Function,
  },
  data: function () {
    return {};
  },
  created() {},
  mounted() {
    let theThis = this;
    window.onkeydown = function (event) {
      if (event.keyCode == 27) {
        theThis.$refs.dlgEl.close();
      }
    };

    // close dialog when click outside (applicable only for 'showModal()' )
    window.addEventListener("click", (event) => {
      try {
        if (event.target === theThis.$refs.dlgEl) {
          theThis.$refs.dlgEl.close();
        }
      } catch (e) {
        console.error(e);
      }
    });
  },

  computed: {
    isShowHeader: function () {
      return this.header && this.header.length > 0;
    },
  },

  watch: {},

  methods: {
    /*show() {
      try {
        this.$refs.dlgEl.show();
      } catch (e) {
        console.error(e);
      }
    },*/
    showModal() {
      try {
        this.$refs.dlgEl.showModal();
      } catch (e) {
        console.error(e);
      }
    },
    close() {
      try {
        this.$refs.dlgEl.close();
      } catch (e) {
        console.error(e);
      }
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";
dialog.dialogDefaults {
  background: var(--background-color);
  border: 1px solid;

  max-height: 85%;
  max-width: 85%;
  overflow-y: auto;
}

.center {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
}
</style>
