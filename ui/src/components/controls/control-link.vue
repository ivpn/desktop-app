<template>
  <span>
    <button class="link" v-on:click="shellOpenExternal(url)" :title="url">
      {{ label }}
    </button>
  </span>
</template>

<script>
const sender = window.ipcSender;

export default {
  props: {
    label: String,
    url: String,
  },
  methods: {
    shellOpenExternal: (urlToOpen) => {
      sender.shellOpenExternal(urlToOpen);
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
@import "@/components/scss/constants";

button.link {
  @extend .noBordersTextBtn;
  @extend .settingsLinkText;
  font-size: inherit;
}
</style>
