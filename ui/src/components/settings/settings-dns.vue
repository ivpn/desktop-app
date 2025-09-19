<template>
  <div style="min-height: 100%; display: flex; flex-direction: column;">
    <div class="settingsTitle" tabindex="0">DNS SETTINGS</div>

    <div class="param" tabindex="0" style="margin-bottom: 4px;">
      <input type="checkbox" id="dnsIsCustom" v-model="_dnsIsCustom" @input="isDnsValueChanged = true"/>
      <label class="defColor" for="dnsIsCustom"
          >Use custom DNS server when connected to IVPN</label
      >
          <button
            class="noBordersBtn flexRow"
            title="Help"
            v-on:click="$refs.helpCustomDns.showModal()"
          >
            <img src="@/assets/question.svg" />
          </button>
          <ComponentDialog ref="helpCustomDns" header="Info">
            <div>
              <p>
                You can specify one or more custom DNS servers to be used when connected to IVPN.<br/>
                When multiple DNS servers are specified, there is no guarantee that they will be used in the order listed.
              </p>
              <p>
                <strong>DNS over HTTPS (DoH)</strong> can be enabled for each DNS server individually. 
                DoH is a protocol that performs Domain Name System (DNS) resolution via HTTPS, 
                designed to increase user privacy and security by preventing eavesdropping and manipulation of DNS data.
              </p>
              <p>
                <strong>Important:</strong> When enabling DoH for a DNS server, ensure that:
              </p>
              <ul>
                <li>The server supports DNS over HTTPS</li>
                <li>You provide a valid DoH template URI specific to your chosen DNS provider (check your provider's documentation for the correct endpoint)</li>
              </ul>

              <p v-if="isShowDnsproxyDescription" class="fwDescription">
                <strong>Implementation:</strong> DNS over HTTPS (DoH) is implemented using dnscrypt-proxy from
                the DNSCrypt project. Your DNS settings will be configured to
                send requests to dnscrypt-proxy listening on localhost (127.0.0.1).
              </p>
            </div>
          </ComponentDialog>
    </div>

    <div v-if="linuxIsShowResolvConfMgmtOption">
      <div class="param">
        <input
          type="checkbox"
          id="linuxDnsIsResolvConfMgmt"
          v-model="linuxDnsIsResolvConfMgmt"
        />
        <label class="defColor" for="linuxDnsIsResolvConfMgmt"
          >Force management of DNS using resolv.conf</label
        >
      </div>
      <div class="paramProps fwDescription" style="margin-bottom: 0px;">
        By default IVPN manages DNS resolvers using the 'systemd-resolved'
        daemon which is the correct method for systems based on Systemd. This
        option enables you to override this behavior and allow the IVPN app to
        directly modify the '/etc/resolv.conf' file.
      </div>
    </div>

    <div v-bind:class="{ disabled: dnsIsCustom === false }" style="overflow-y: auto; ">

      <div style="padding-left: 20px;">
      
      <!-- List of custom DNS servers -->
      <ul style="list-style: none; padding-left: 0px;">
        <li
          v-for="(svr, idx) in _dnsCustomCfg.Servers"
          style="margin-bottom: 10px;"
        >
          <div style="display: flex; ">
            <div style="font-weight: bold; flex: 1;"> 
              DNS Server {{ idx + 1 }} 
            </div>

            <button
              class="noBordersBtn flexRow remove-btn"
              v-on:click="removeServer(svr)"
            >
              <img src="@/assets/trash.svg" width="16" style="margin-right: 4px;"/>
              Remove
            </button>
          </div>

          <div>
            <div class="flexRow">
              <input
                  class="settingsTextInput"
                  style="width: 100%"
                  v-model.trim="svr.Address"
                  @input="isDnsValueChanged = true"
                  v-bind:class="{ badData: isIpAddressError(svr.Address) }"
                  :placeholder="'IP address'"              
                />              
            </div>

            <div class="flexRow" >
              <div 
                title="DNS encryption: DoH (DNS over HTTPS)"
                style="display: flex; flex: 1; align-items: center; min-height: 30px; "
                >
                  <input 
                    style="margin-left: 0px;"
                    type="checkbox"
                    :id="`doh-checkbox-${idx}`"
                    :checked="idDoH(svr.Encryption)"
                    @input="updateServerEncryption(svr, $event.target.checked)"
                  />
                  <label :for="`doh-checkbox-${idx}`">DNS over HTTPS</label>

                  <!-- Input with drop-down button of Predefined DoH configs-->
                  <div style="position: relative; display: flex; flex: 1;">                    
                    <input v-if="svr.Encryption !== 0"                      
                      class="settingsTextInput"
                      style="flex: 1; margin-left: 5px; padding-right: 22px;"                      
                      :placeholder="'DNS over HTTPS template URI'"
                      v-model.trim="svr.Template"
                      @input="isDnsValueChanged = true"
                      v-bind:class="{ badData: isDohTemplateURIError(svr.Encryption, svr.Template) }"
                    />
                                        
                    <!-- Predefined DoH/DoT configs -->
                    <div v-bind:class="{ HiddenDiv: svr.Encryption === 0 || isHasPredefinedDohConfigs !== true }"
                      style="margin-left: 5px;                      
                            position: absolute; 
                            top: 60%; 
                            right: 0px; 
                            transform: translateY(-50%); 
                            cursor: pointer;"
                    >                      
                      <div>
                        <!-- drop-down image -->
                        <img
                          style="position: fixed; width: 12px; margin-left: 5px; margin-top: 8px;                          "
                          src="@/assets/arrow-bottom.svg"
                        />
                        <!-- Popup -->
                        <select
                          title="Predefined DoH configurations"                          
                          style="cursor: pointer; width: 24px; height: 22px; opacity: 0"
                          @change="applyPredefinedConfig($event.target.value, svr)"
                        >
                          <option
                            v-for="m in predefinedDohConfigs"
                            v-bind:key="m.DohTemplate + m.DnsHost"                            
                            v-bind:value="JSON.stringify(m)"
                          >
                            {{ m.DnsHost }} ({{ m.DohTemplate }})
                          </option>
                        </select>
                      </div>
                    </div>
                  </div>
              </div>
            </div>
          </div>          
        </li>
      </ul>
      
      <button title="Add new DNS server"
        @click="addServer" 
        class="likeText"
        style="padding-left: 0px;"
      >
        + Add custom DNS server
      </button>

      </div>

    </div>

    <div class="paramProps"  style="margin-top: auto; margin-bottom: 20px;">
      <div class="fwDescription" tabindex="0">
        AntiTracker will override the custom DNS when enabled.
      </div>
    </div>
  </div>
</template>

<script>
import { DnsEncryption, VpnStateEnum } from "@/store/types";
import { Platform, PlatformEnum } from "@/platform/platform";
import ComponentDialog from "@/components/component-dialog.vue";

const sender = window.ipcSender;

function checkIsDnsIPError(dnsIpString) {
  if (!dnsIpString || dnsIpString.trim() === '') return true;
  const singleIPRegex = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)(\.(?!$)|$)){4}$/;
  return !singleIPRegex.test(dnsIpString.trim());
}

function processError(e) {
  if (!e) return;
  console.error(e);
  sender.showMessageBoxSync({
    type: "error",
    buttons: ["OK"],
    message: e,
  });
}

export default {
  components: {
    ComponentDialog,
  },
  props: { registerBeforeCloseHandler: Function },
  created() {
    // We have to call applyChanges() even when Settings window was closed by user
    // (the 'beforeUnmount()' is not called in this case)
    window.addEventListener("beforeunload", this.applyChanges);

    if (this.registerBeforeCloseHandler != null) {
      // Register handler which will be called before closing current view
      // Handler MUST be 'async' function and MUST return 'true' to allow to switch current view
      this.registerBeforeCloseHandler(this.applyChanges);
    }
  },

  async beforeUnmount() {
    window.removeEventListener("beforeunload", this.applyChanges);
    await this.applyChanges();
  },

  data: function () {
    return {
      isEditingFinished: false,
      isDnsValueChanged: false,
      
      _dnsIsCustom: false,
      _dnsCustomCfg: { Servers: [] },
      _linuxDnsIsResolvConfMgmt: false,
    };
  },

  mounted() {
    const storeDnsCfg = this.getDnsCustomCfg() || {};
    this._dnsCustomCfg = JSON.parse(JSON.stringify(storeDnsCfg));
    this._dnsIsCustom = this.getDnsIsCustom() || false;
    this._linuxDnsIsResolvConfMgmt =
      this.daemonSettings?.UserPrefs?.Linux?.IsDnsMgmtOldStyle || false;

    // Remove empty servers (if any)
    this._dnsCustomCfg.Servers = this._dnsCustomCfg.Servers.filter(svr => svr && svr.Address);

    this.requestPredefinedDohConfigs();

  },
  methods: {
    checkIsDisconnectedAndWarn: function () {
      if (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      )
        return true;

      sender.showMessageBoxSync({
        type: "info",
        buttons: ["OK"],
        message: "You are now connected to IVPN",
        detail: "You can change this settings only when IVPN is disconnected.",
      });

      return false;
    },

    // Validate and APPLY changes
    async applyChanges(e) {
      this.isEditingFinished = true;

      let ipError = false;
      let isTemplateError = false;
      for (const svr of this._dnsCustomCfg.Servers) {
        if (svr.Encryption === DnsEncryption.None) {
          svr.Template = "" // Template must be empty when encryption is 'None'
        }

        if (this.isIpAddressError(svr.Address)) {
          ipError = true;
        }
        if (this.isDohTemplateURIError(svr.Encryption, svr.Template)) {
          isTemplateError = true;
        }
      }

      if (this._dnsIsCustom && (isTemplateError || ipError)) {
        sender.showMessageBoxSync({
          type: "warning",
          buttons: ["OK"],
          message: "Error in DNS configuration.",
          detail: `Custom DNS will not be applied.`,
        });

        if (e && typeof e.preventDefault === "function") {
          // it is 'beforeunload' handler. Prevent closing window.
          e.preventDefault();
          e.returnValue = "";
        }
        return false;
      }

      if (this.isDnsValueChanged !== true) return true;
      try {
        this.$store.dispatch("settings/dnsCustomCfg", this._dnsCustomCfg);
        this.$store.dispatch("settings/dnsIsCustom", this._dnsIsCustom);

        // Apply changes
        await sender.SetDNS();
        this.isDnsValueChanged = false;
      } catch (err) {
        processError(err);
        // it is 'beforeunload' handler. Prevent closing window.
        if (e && typeof e.preventDefault === "function") {
          e.preventDefault();
          e.returnValue = "";
        }
      }
      return true;
    },

    requestPredefinedDohConfigs() {
      if (!this.CanUseDnsOverHttps) return;
      if (this.$store.state.dnsPredefinedConfigurations) return; // configurations already initialized - no sense to request them again
       // Request predefined DoH configurations from main process
      setTimeout(() => {
        sender.RequestDnsPredefinedConfigs();
      }, 0);
    },

    getDnsCustomCfg() {
      return this.$store.state.settings.dnsCustomCfg;
    },

    getDnsIsCustom() {
        return this.$store.state.settings.dnsIsCustom;
    },

    idDoH(encryption) {
      return encryption === DnsEncryption.DnsOverHttps;
    },

    updateServerEncryption(server, isChecked) {
      this.isDnsValueChanged = true;
      server.Encryption = isChecked ? DnsEncryption.DnsOverHttps : DnsEncryption.None;
    },

    removeServer(server) {
      this.isDnsValueChanged = true;
      let cfg = this._dnsCustomCfg;
      if (!cfg || !cfg.Servers) return;
      const index = cfg.Servers.indexOf(server);
      if (index > -1) {
        cfg.Servers.splice(index, 1);
      }
    },

    addServer() {
      this.isDnsValueChanged = true;
      this._dnsCustomCfg.Servers.push({
        Address: "",
        Encryption: 0,
        Template: ""
      });
    },

    applyPredefinedConfig(config, svr) {
      if (!config || !svr) return;
    
      try {     
        const cfg = JSON.parse(config);
        this.isDnsValueChanged = true;

        svr.Template = cfg.DohTemplate;
        svr.Address = cfg.DnsHost;
        svr.Encryption = DnsEncryption.DnsOverHttps;
      } catch (e) {
        console.error('Error parsing predefined config:', e);
      }
    },

    isIpAddressError(address) {
      if (!this.isEditingFinished) return false;
      return checkIsDnsIPError(address);
    },

    isDohTemplateURIError(dnsEncryption, template) {
      if (!this.isEditingFinished) return false;
      if (!dnsEncryption) return false; // no error when encryption is 'None'
      if (!template || template.trim() === "") return true; // error when empty
      // Basic validation of URI template
      const uriRegex = /^(https?:\/\/)?([\w.-]+)(:\d+)?(\/.*)?$/i;
      return !uriRegex.test(template);
    },
  },
  watch: {
    daemonSettings() {
      this._linuxDnsIsResolvConfMgmt =
        this.daemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle;
    },    
  },

  computed: {
    // needed for 'watch'
    daemonSettings() {
      return this.$store.state.settings.daemonSettings;
    },   

    isShowDnsproxyDescription() {
      return Platform() !== PlatformEnum.Windows;
    },
    CanUseDnsOverTls: {
      get() {
        return this.$store.state.dnsAbilities.CanUseDnsOverTls === true;
      },
    },
    CanUseDnsOverHttps: {
      get() {
        return this.$store.state.dnsAbilities.CanUseDnsOverHttps === true;
      },
    },

    dnsIsCustom: {
      get() {
        return this._dnsIsCustom;
      },
      async set(value) {
        this.isDnsValueChanged = true;
        this._dnsIsCustom = value;
      },
    },

    linuxIsShowResolvConfMgmtOption() {
      try {
        if (Platform() !== PlatformEnum.Linux) return false;

        const disabledFuncs = this.$store.state.disabledFunctions;
        if (
          disabledFuncs.Platform.Linux.DnsMgmtOldResolvconfError != "" ||
          disabledFuncs.Platform.Linux.DnsMgmtNewResolvectlError != ""
        )
          return false;

        const dSettings = this.$store.state.settings.daemonSettings;
        if (dSettings?.UserPrefs?.Linux?.IsDnsMgmtOldStyle == null)
          return false;

        return true;
      } catch (e) {
        console.error(e);
        return false;
      }
    },

    linuxDnsIsResolvConfMgmt: {
      get() {
        return this._linuxDnsIsResolvConfMgmt;
      },
      async set(value) {
        const clone = function (obj) {
          return JSON.parse(JSON.stringify(obj));
        };

        try {
          // We need to erase value in order to the check-box be updated correctly according to confirmation response from daemon
          // The value will be updated in "watch: daemonSettings()"
          this._linuxDnsIsResolvConfMgmt = null;

          if (!this.checkIsDisconnectedAndWarn()) {
            return;
          }

          let prefs = clone(
            this.$store.state.settings.daemonSettings.UserPrefs,
          );
          if (prefs.Linux.IsDnsMgmtOldStyle != value) {
            prefs.Linux.IsDnsMgmtOldStyle = value;
            await sender.SetUserPrefs(prefs);
          }
        } catch (e) {
          processError(e);
        } finally {
          setTimeout(() => {
            this._linuxDnsIsResolvConfMgmt =
              this.daemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle;
          }, 0);
        }
      },
    },

    isHasPredefinedDohConfigs: {
      get() {
        if (!this.CanUseDnsOverHttps && !this.CanUseDnsOverTls) return false;
        return this.predefinedDohConfigs && this.predefinedDohConfigs.length > 0;        
      },
    },

    predefinedDohConfigs: {
      get() {
        let cfgs = this.$store.state.dnsPredefinedConfigurations;
        if (!cfgs) return null;

        let filtered = cfgs.filter(
          (cfg) =>
            cfg.Encryption === DnsEncryption.DnsOverHttps &&
            cfg.DnsHost &&
            cfg.DohTemplate &&
            !checkIsDnsIPError(cfg.DnsHost),
        );
        return filtered;
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

div.paramProps {
  margin-top: 9px;
  margin-bottom: 17px;
  margin-left: 22px;
}
div.fwDescription {
  @extend .settingsGrayLongDescriptionFont;
  margin-top: 8px;
  max-width: 425px;
}

div.param {
  @extend .flexRow;
  margin-top: 3px;
}

div.paramName {
  min-width: 120px;
  max-width: 120px;
}

label {
  margin-left: 1px;
  font-weight: 500;
}

input.badData {
  border-color: red;
}

input:disabled {
  opacity: 0.5;
}

div.disabled {
  pointer-events: none;
  opacity: 0.5;
}

div.settingsRadioBtnEx {
  @extend .settingsRadioBtn;
  padding-right: 20px;
}

div.HiddenDiv {
  opacity: 0;
}
div.HiddenDiv > * {
  opacity: 0;
  pointer-events: none;
  cursor: default;
}

button.likeText {
  border: none; 
  background: none;
  color: #0066cc; 
  cursor: pointer; 
  font-size: inherit;
}

// Hide remove button by default
li .remove-btn {
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.2s ease, visibility 0.2s ease;
}

// Show remove button on li hover or focus
li:hover .remove-btn,
li:focus-within .remove-btn {
  opacity: 1;
  visibility: visible;
}
</style>
