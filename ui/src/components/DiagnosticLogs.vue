<template>
  <body id="main">
    <div id="innerColumn">
      <div style="text-align: center" id="title">Diagnostic logs</div>
      <div style="text-align: center">
        The following information will be submitted to IVPN for further analysis
      </div>

      <!-- TAB-view header (diagnostic) -->
      <div class="flexRow" style="margin-bottom: 10px">
        <button
          v-on:click="onTabSelected('daemonLogs')"
          class="selectableButtonOff"
          v-bind:class="{ selectableButtonOn: activeTabName == 'daemonLogs' }"
        >
          Logs
        </button>
        <button
          v-on:click="onTabSelected('settings')"
          class="selectableButtonOff"
          v-bind:class="{ selectableButtonOn: activeTabName == 'settings' }"
        >
          Settings
        </button>
        <button
          v-on:click="onTabSelected('other')"
          class="selectableButtonOff"
          v-bind:class="{ selectableButtonOn: activeTabName == 'other' }"
        >
          Other
        </button>
        <button
          v-on:click="onTabSelected('userComment')"
          class="selectableButtonOff"
          v-bind:class="{ selectableButtonOn: activeTabName == 'userComment' }"
        >
          User comment
        </button>
        <button
          style="cursor: auto; flex-grow: 1"
          class="selectableButtonSeparator"
        ></button>
      </div>

      <div class="flexColumn">
        <div v-if="activeTabName == 'daemonLogs'" class="flexColumn">
          <textarea readonly id="logsBlock" v-model="viewTextLogs" />
        </div>
        <div v-if="activeTabName == 'settings'" class="flexColumn">
          <textarea readonly id="logsBlock" v-model="viewTextSettings" />
        </div>
        <div v-if="activeTabName == 'other'" class="flexColumn">
          <textarea readonly id="logsBlock" v-model="viewTextOther" />
        </div>
        <div v-if="activeTabName == 'userComment'" class="flexColumn">
          <div style="margin-bottom: 5px">
            Please write a description of the problem you are experiencing:
          </div>
          <textarea
            id="commentBlock"
            v-model="userComment"
            v-bind:class="{
              redBorder:
                showRedBorderWhenEmptyComment === true &&
                userComment.length == 0,
            }"
          />
        </div>
      </div>
      <div
        v-if="!isLoggingEnabled"
        style="margin-top: 5px; text-align: right; color: orange"
      >
        Logging is disabled.<br />
        Please, enable "Allow logging" in the General settings to provide
        extended information to the developers.
      </div>
      <div class="flexRow" style="margin-top: 20px">
        <div style="flex-grow: 1" />
        <button class="slave btn" v-on:click="onCancel">Cancel</button>
        <div style="width: 10px" />
        <button class="master btn" v-on:click="onSendLogs">Send logs</button>
      </div>
    </div>
  </body>
</template>

<script>
const sender = window.ipcSender;

const LogProperties = Object.freeze({
  Settings: " Settings",
  Log0: "Log0_Old",
  Log1: "Log1_Active",
});

export default {
  props: {
    onClose: Function,
  },

  data() {
    return {
      showRedBorderWhenEmptyComment: false,
      activeTabName: "daemonLogs", // [daemonLogs, settings, other, userComment]
      diagnosticDataObj: null,
      userComment: "",
    };
  },
  mounted() {
    setTimeout(async () => {
      this.getDiagnosticData();
    }, 0);
  },
  computed: {
    isLoggingEnabled: function () {
      return this.$store.state.settings.logging;
    },
    viewTextLogs: function () {
      if (this.diagnosticDataObj == null) return null;
      let ret = [];
      if (this.diagnosticDataObj[LogProperties.Log0]) {
        ret.push("Service log (old session):\n");
        ret.push(this.diagnosticDataObj[LogProperties.Log0]);
        ret.push("\n");
      }
      if (this.diagnosticDataObj[LogProperties.Log1]) {
        ret.push("Service log (active session):\n");
        ret.push(this.diagnosticDataObj[LogProperties.Log1]);
      }
      return ret.join("\n");
    },
    viewTextSettings: function () {
      if (this.diagnosticDataObj == null) return null;
      let ret = [];
      if (this.diagnosticDataObj[LogProperties.Settings]) {
        ret.push(this.diagnosticDataObj[LogProperties.Settings]);
        ret.push("\n");
      }
      return ret.join("\n");
    },
    viewTextOther: function () {
      if (this.diagnosticDataObj == null) return null;

      const props = Object.keys(this.diagnosticDataObj);
      props.sort();

      let text = [];
      for (const pName of props) {
        if (
          pName == LogProperties.Settings ||
          pName == LogProperties.Log0 ||
          pName == LogProperties.Log1
        )
          continue;

        let val = `${this.diagnosticDataObj[pName]}`;
        val = val.trim();
        if (!val) continue;
        val = val.replace(/\\n/g, "\n");

        text.push(pName.trim() + ": ");
        text.push(val + "\n\n");
      }
      return text.join("");
    },
  },
  methods: {
    async getDiagnosticData() {
      this.diagnosticDataObj = await sender.GetDiagnosticLogs();
    },
    onTabSelected(tabName) {
      this.activeTabName = tabName;
    },
    onCancel() {
      if (this.onClose != null) this.onClose();
    },
    async onSendLogs() {
      this.showRedBorderWhenEmptyComment = true;
      if (this.diagnosticDataObj != null) {
        let comment = this.userComment.trim();
        if (comment.length <= 0) {
          this.activeTabName = "userComment";
          setTimeout(() => {
            sender.showMessageBoxSync({
              type: "warning",
              buttons: ["OK"],
              message: "User comment is empty",
              detail:
                "Please write a description of the problem you are experiencing",
            });
          }, 0);
          return;
        }

        let id = await sender.SubmitDiagnosticLogs(
          this.userComment,
          this.diagnosticDataObj
        );

        sender.showMessageBoxSync({
          type: "info",
          buttons: ["OK"],
          message: "Report sent to IVPN",
          detail: `Report ID: ${id}`,
        });
      }

      if (this.onClose != null) this.onClose();
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

#main {
  @extend .flexColumn;
  height: 100%;
  margin: 0px;
}

#title {
  margin-bottom: 10px;
  font-size: 16px;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: var(--text-color-settings-menu);
}

#innerColumn {
  @extend .flexColumn;
  margin: 20px;
}
#logsBlock {
  flex-grow: 1;
  font-family: monospace;
  resize: none;
}
#commentBlock {
  flex-grow: 1;
  resize: none;
  padding: 5px;
}
.redBorder {
  border-color: red;
}
button.btn {
  height: 30px;
  width: 150px;
}
button.btnDisabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
