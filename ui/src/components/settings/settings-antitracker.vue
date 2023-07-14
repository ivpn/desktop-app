<template>
  <div>
    <div class="settingsTitle">ANTITRACKER SETTINGS</div>

    <div class="defColor" style="margin-bottom: 24px">
      When AntiTracker is enabled, IVPN blocks ads, malicious websites, and
      third-party trackers using our private DNS servers.
      <button class="link" v-on:click="onLearnMoreLink">Learn more</button>
      about how IVPN AntiTracker is implemented.
    </div>

    <div class="flexRow paramBlock" style="margin-bottom: 12px">
      <div class="defColor paramName">Block list:</div>
      <select v-model="AtPlusListNameSelected">
        <optgroup
          v-for="group in AtPlusLists"
          :key="group.name"
          :label="group.name"
        >
          <option
            v-for="item in group.lists"
            :key="item.Name"
            :value="item.Name"
          >
            {{ item.Description ? item.Description : item.Name }}
          </option>
        </optgroup>
      </select>
    </div>
    <div class="fwDescription">
      Block lists refer to DNS blocking lists used by our AntiTracker. The
      'Basic', 'Comprehensive', and 'Restrictive' options are combinations of
      individual lists, each offering a different level of protection. You also
      have the freedom to select from individual lists for a more tailored
      AntiTracker experience.
    </div>
    <div class="fwDescription">
      <button class="link" v-on:click="onAntitrackerBlockListLink">
        Lern more
      </button>
      about AntiTracker block lists.
    </div>

    <div class="param">
      <input
        type="checkbox"
        id="isAntitrackerHardcore"
        v-model="isAntitrackerHardcore"
      />
      <label class="defColor" for="isAntitrackerHardcore">Hardcore Mode</label>
    </div>

    <div class="fwDescription">
      Hardcore mode blocks the leading companies with business models relying on
      user surveillance (currently: Google and Facebook)
    </div>
    <div class="fwDescription">
      To better understand how this may impact your experience please refer to
      our
      <button class="link" v-on:click="onHardcodeLink">hardcore mode FAQ</button
      >.
    </div>
  </div>
</template>

<script>
const sender = window.ipcSender;

export default {
  data: function () {
    return {};
  },
  methods: {
    onAntitrackerBlockListLink: () => {
      sender.shellOpenExternal(
        `https://www.ivpn.net/knowledgebase/antitracker/blocklists/`
      );
    },
    onLearnMoreLink: () => {
      sender.shellOpenExternal(`https://www.ivpn.net/antitracker`);
    },
    onHardcodeLink: () => {
      sender.shellOpenExternal(`https://www.ivpn.net/antitracker/hardcore`);
    },
  },
  computed: {
    isAntitrackerHardcore: {
      get() {
        return this.$store.state.settings.antiTracker?.Hardcore;
      },
      async set(value) {
        let at = this.$store.state.settings.antiTracker;
        if (!at)
          at = {
            Enabled: false,
            Hardcore: value,
            AntiTrackerBlockListName: "",
          };
        else at = JSON.parse(JSON.stringify(at));
        at.Hardcore = value;

        this.$store.dispatch("settings/antiTracker", at);
        await sender.SetDNS();
      },
    },
    AtPlusLists: {
      //groups: [
      //  {
      //    name: "Pre-defined lists",
      //    lists: [{"Name":"Basic", "Normal":"", "Hardcore":""}, ...],
      //  },
      //  {
      //    name: "Individual lists",
      //    lists: [{"Name":"Oisdbig", "Normal":"10.0.254.2", "Hardcore":"10.0.254.3"}, ...],
      //  },
      //],
      get() {
        let atPlusSvrs =
          this.$store.state.vpnState.servers.config?.antitracker_plus
            ?.DnsServers;
        if (!atPlusSvrs) {
          return [];
        }

        let listBasic = null;
        let listComprehensive = null;
        let listRestrictive = null;
        let listOisdbig = null;

        let groupPredefined = { name: "Pre-defined lists", lists: [] };
        let groupIndividual = { name: "Individual lists", lists: [] };

        for (var s of atPlusSvrs) {
          switch (s.Name) {
            case "Basic":
              listBasic = s;
              break;
            case "Comprehensive":
              listComprehensive = s;
              break;
            case "Restrictive":
              listRestrictive = s;
              break;
            case "Oisdbig":
              listOisdbig = s;
              break;
            default:
              groupIndividual.lists.push(s);
              break;
          }
        }
        if (listBasic) groupPredefined.lists.push(listBasic);
        if (listComprehensive) groupPredefined.lists.push(listComprehensive);
        if (listRestrictive) groupPredefined.lists.push(listRestrictive);

        if (listOisdbig) groupIndividual.lists.unshift(listOisdbig); // add as a first element

        return [groupPredefined, groupIndividual];
      },
    },
    AtPlusListNameSelected: {
      get() {
        return this.$store.state.settings.antiTracker.AntiTrackerBlockListName;
      },
      set(value) {
        let at = this.$store.state.settings.antiTracker;
        if (!at)
          at = {
            Enabled: false,
            Hardcore: false,
            AntiTrackerBlockListName: value,
          };
        else at = JSON.parse(JSON.stringify(at));
        at.AntiTrackerBlockListName = value;

        this.$store.dispatch("settings/antiTracker", at);
        sender.SetDNS();
      },
    },
  },
};
</script>

<style scoped lang="scss">
@import "@/components/scss/constants";

.defColor {
  @extend .settingsDefaultTextColor;
}

div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
  max-width: 425px;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

button.link {
  @extend .noBordersTextBtn;
  @extend .settingsLinkText;
  font-size: inherit;
}
label {
  margin-left: 1px;
  font-weight: 500;
}

div.paramName {
  min-width: 100px;
  max-width: 100px;
}
</style>
