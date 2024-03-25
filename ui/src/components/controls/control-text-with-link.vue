<template>
  <span>    
    <span>{{ textPart1 }}</span>    
      <linkCtrl v-if="textLinkText && textLink"
        :label="textLinkText"
        :url="textLink"
      />
      <span>{{ textPart2 }}</span>
  </span>
</template>

<script>
import linkCtrl from "@/components/controls/control-link.vue";

export default {
  components: {    
    linkCtrl,
  },
  props: {
    text: String,
    textRequired: String,
    textToUseAsLink: String,
    link: String,
  },
  methods: {
    splitTextForLink(warning) {      
      let retObject = {
        Part1: warning,
        Part2: "",
        LinkText: "",
      };

      if (this.textRequired && !warning?.includes(this.textRequired)) 
        return retObject;

      if (warning?.includes(this.textToUseAsLink)) {
        let parts = warning?.split(this.textToUseAsLink);
        if (parts?.length == 2) {
          retObject.Part1 = parts[0];
          retObject.Part2 = parts[1];
          retObject.LinkText = this.textToUseAsLink;
          retObject.Link = this.link;
        }
      }  
      return retObject;
    }
  },
  computed: {
    textPart1: function () {
      return this.splitTextForLink(this.text)?.Part1;
    },
    textPart2: function () {
      return this.splitTextForLink(this.text)?.Part2;
    },
    textLinkText: function () {
      return this.splitTextForLink(this.text)?.LinkText;
    },
    textLink: function () {
      return this.splitTextForLink(this.text)?.Link;
    },
  }
};

</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
span {
  color: inherit;
}
</style>
