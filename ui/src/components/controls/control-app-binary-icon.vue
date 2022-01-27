<template>
  <img :src="theBase64Icon" />
</template>

<script>
const sender = window.ipcSender;

export default {
  props: {
    binaryPath: String,
    preloadedBase64Icon: String,
  },
  data: () => ({
    base64Icon: "",
    defaultIcon:
      "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAAEaSURBVFhH7ZTbCoJAEIaFCCKCCKJnLTpQVBdB14HQ00T0CqUP4AN41puJAVe92F3HRZegHfgQFvH7/1nQMmPmZ+Z8uYJOCm01vJe64PF8cZ+Ftho89DxPC8IAeZ73QpZlJWmattsAfsBavsk0yRsD3Ox7ST3A4uTC/OjC7ODCdO/AZOfAeOvAaPOB4foDg1UVwLZtIUmSqG2AIq9vgNcc5coBKHIWgNec0RhAdAUUOSJrjsRxrLYBihxBMa85QzkARY7ImjOkAURXQJEjKOY1Z0RRpLYBihyRNUe5cgCKHEEprzmjMYDoCqjImiNhGKptgApvA3V57wFkzbUGEMmDIGgfAKH84ShypQBdyn3fFwfQSaE1Y+bvx7K+efsbU5+Ow3MAAAAASUVORK5CYII=",
  }),
  computed: {
    theBase64Icon: function () {
      if (this.preloadedBase64Icon) return this.preloadedBase64Icon;
      if (this.base64Icon) return this.base64Icon;
      return this.defaultIcon;
    },
  },
  mounted() {
    if (!this.preloadedBase64Icon) this.loadIcon();
  },
  methods: {
    async loadIcon() {
      if (this.binaryPath == null) return null;
      try {
        let ico = await sender.getAppIcon(this.binaryPath);
        this.base64Icon = ico;
      } catch (e) {
        console.error(`Error receiving appicon '${this.binaryPath}': `, e);
      }
      if (!this.base64Icon) this.base64Icon = this.defaultIcon;
    },
  },
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss"></style>
