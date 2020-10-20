<template>
  <body id="main">
    <div id="innerColumn">
      <h2 style="text-align: center;">Diagnostic logs</h2>
      <div style="margin-bottom: 5px;">
        The following information will be submited to IVPN for further analysis:
      </div>

      <textarea readonly id="logsBlock" v-model="diagDataText" />

      <div style="margin-top:20px; margin-bottom: 5px;">
        Please write a description of the problem you are experincing:
      </div>
      <textarea id="commentBlock" v-model="userComment"></textarea>
      <div class="flexRow" style="margin-top:20px;">
        <button class="slave btn" v-on:click="onCancel">
          Cancel
        </button>

        <div style="width:50px" />

        <button
          class="master btn"
          v-bind:class="{ btnDisabled: userComment.length <= 0 }"
          v-on:click="onSendLogs"
        >
          Send logs
        </button>
      </div>
    </div>
  </body>
</template>

<script>
import sender from "@/ipc/renderer-sender";
const { dialog, getCurrentWindow } = require("electron").remote;

export default {
  props: {
    onClose: Function
  },

  data() {
    return {
      diagnosticDataObj: null,
      userComment: ""
    };
  },
  mounted() {
    setTimeout(async () => {
      this.getDiagnosticData();
    }, 0);
  },
  computed: {
    diagDataText: function() {
      if (this.diagnosticDataObj == null) return null;
      let text = JSON.stringify(this.diagnosticDataObj, null, 2);
      return text.replace(/\\n/g, "\n");
    }
  },
  methods: {
    async getDiagnosticData() {
      this.diagnosticDataObj = await sender.GetDiagnosticLogs();
    },

    onCancel() {
      if (this.onClose != null) this.onClose();
    },
    async onSendLogs() {
      if (this.diagnosticDataObj != null) {
        let id = await sender.SubmitDiagnosticLogs(
          this.userComment,
          this.diagnosticDataObj
        );

        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "info",
          buttons: ["OK"],
          message: "Report sent to IVPN",
          detail: `Report ID: ${id}`
        });
      }

      if (this.onClose != null) this.onClose();
    }
  }
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

#main {
  @extend .flexColumn;
  height: 100%;
  margin: 0px;
}
#innerColumn {
  @extend .flexColumn;
  margin: 20px;
}
#logsBlock {
  flex-grow: 1;
  font-family: monospace;

  height: 0px;
  overflow: auto;
  resize: none;
  background: lightgrey;
  color: grey;
}
#commentBlock {
  height: 70px;
  padding: 10px;
  resize: none;
}
button.btn {
  height: 30px;
}
button.btnDisabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
