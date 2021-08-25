<template>
  <img :src="base64Icon" />
</template>

<script>
const sender = window.ipcSender;

export default {
  props: {
    binaryPath: String
  },
  data: () => ({
    base64Icon: ""
  }),
  mounted() {
    this.loadIcon();
  },
  methods: {
    async loadIcon() {
      if (this.binaryPath == null) return null;
      try {
        this.base64Icon = await sender.getAppIcon(this.binaryPath);
      } catch (e) {
        console.error(
          `Error receiving application icon '${this.binaryPath}': `,
          e
        );
      }
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss"></style>
