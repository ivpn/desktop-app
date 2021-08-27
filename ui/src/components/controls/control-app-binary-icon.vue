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
    base64Icon: "",
    defaultIcon:
      "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAAQXSURBVFhH7VfLbiNFFL3ddjKeeCIWEYNAvKQZwQoQs0RssmPDL7DgT9jxC0gs+AE+AUWseKwQMCQIhWRC4tiQZJzY3e52P4pz7q1qmyQ2sJjMAu7odD26+t5zT90qZ+Q/b9HW1tarvv80rB9tb2+7TqfjxzdnRVHIYDDYjH7d23Nra2t+GpL49kmY8y0tyzI57vU2o739fbd+Z/3JRr7GkiSR3tHRZuxAS5nxcdOARfuPDly325VPviuks4J5fWlv/ZqZOFGk/SiKrY/2wV0nrz9T4ZOw+qodZ6vyzQnqDEvWVyp576WppFSgBwXokZ+uxrXcbjmglk7bARyjD5RVIReTXE7HmfwxzuUsySXNC4lcqWT5/TLUrpao5vpCXF028zTdAlpVlX9BUZbSG6by/eFQfu6P5LezRAbnqZxcpDIADjDeGYwlLytzsMTqupYfexfyw9G59OFj3qAl2ICEBa4UCbJ9eDSUg9ORTPIcZKZSehRlgRaAKiVI0nkUtyReAq7JskQxnWYar1GADw4qOCSJi0kmP/UeyxhtCFoWbH1gncNaAt9QXu5/vQRVXclkMlbk+aQJTourCg7QKRE8RbY7x48lm+YIiswRmChLU8GUmJEg6hpBkOEyMLFJCjUnIxBINV6jgA7w4KLd388lAwmTmkEMSgKETIl5sKiogPlYBG5riuxTkKACjM55JcAH+2ejFEgsoA/aKBDaucBBCd0C72MRat0CKjBGgqZAMCiAf5g5Ho7MuQ8Y+krCBzYyBcaeAFQjAS221mJUUIkFSAJahAgcSDTH0KSncx+cAbH3oT+bm6IA2bdC1H2GE4daWASuofTc/wL1xZgNAT44oDMLYhma5DbX9HlSdM739RRYAGa5CHxfoIaYPRMIwWnRzi5+Dbvr8sGnX0gNSfmBc5V+RKZ6DePajWMCkira0oK07daKfPjua/LOvWfV2SIb51PZ7Z/Cn5M7t2/J/ec2JEsTOenzKvZy8BRwT5mVZW4Z2oUT1JjN68WFa1Xlp48l6K6uypsvPy9vvfKC3Lu7YfP+F8YIAOqQJDRI6Pu91r7NBbCyiXAJXZZ9HnB/BcH0HqDVyGYWIGSK/nzGJOEDK3C+AwG2CwH/14FmFxGg1ewd229CCAiwzxbvHN7ZOlY/6wQEcJsug54GBLkMGk6Bf6kEAjyRRhUEDvMIqvBrLT96WY5FBgIoBv2bAMvAxGlWIYhlGuZ0lbLlp+bann9vYV2z3mpQooe/7LkOjuFHn3+p2WpQbgGOIrPWRXGsx6+F48ebjW2rBbTb8v7b9+WNFzd03b+xPEtl2D/cbAg0lG7IpiQwOOQfpTgqesRC8V0FFQl1cS3+yZpL0K2ERV99/e1nK7f8f0xuUASejiJLPvbD/+1pmcifg3tWxjSNYQAAAAAASUVORK5CYII="
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
        console.error(`Error receiving appicon '${this.binaryPath}': `, e);
      }
      if (!this.base64Icon) this.base64Icon = this.defaultIcon;
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss"></style>
