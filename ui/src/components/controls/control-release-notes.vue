<template>
  <div class="flexColumn">
    <div v-if="releaseNotes">
      <div v-for="note of releaseNotes" v-bind:key="note.description">
        <div class="flexRow rnItem">
          <div
            class="badge"
            v-bind:class="{
              'badge-grey':
                note.type && note.type.toLowerCase().startsWith('fix'),
              'badge-green':
                note.type && note.type.toLowerCase().startsWith('new'),
              'badge-blue':
                note.type && note.type.toLowerCase().startsWith('improve')
            }"
          >
            {{ note.type }}
          </div>
          <div>
            {{ note.description }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  props: ["releaseNotes"],
  data: function() {
    return {};
  },
  mounted() {},
  methods: {},
  computed: {}
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.rnItem {
  margin-top: 4px;
  margin-bottom: 4px;
  text-align: left;
}

.badge {
  border-radius: 4pt;
  height: 16pt;
  width: 60pt;
  font-size: 9pt;
  color: white;
  text-align: center;
  line-height: 17pt;
  margin-right: 15px;
  text-transform: capitalize;

  min-width: 80px;
}

.badge-grey {
  background-color: rgb(152, 165, 179);
}

.badge-green {
  background-color: rgb(33, 208, 116);
}

.badge-blue {
  background-color: rgb(57, 158, 230);
}

.boldFont {
  @extend .settingsBoldFont;
  margin-top: 20px;
}
</style>
