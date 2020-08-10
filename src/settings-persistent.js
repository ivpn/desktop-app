const fs = require("fs");
import merge from "deepmerge";
import path from "path";
import { app } from "electron";

import store from "@/store";

var saveSettingsTimeout = null;

const userDataFolder = app.getPath("userData");
const filename = path.join(userDataFolder, "ivpn-settings.json");

export function InitPersistentSettings() {
  if (fs.existsSync(filename)) {
    try {
      // merge data from a settings file
      const data = fs.readFileSync(filename);
      const settings = JSON.parse(data);

      const mergedState = merge(store.state.settings, settings, {
        arrayMerge: combineMerge
      });
      store.commit("settings/replaceState", mergedState);
    } catch (e) {
      console.error(e);
    }
  } else {
    console.log(
      "Settings file not exist (probably, the first application start)"
    );
  }

  // saves settings object each 5 seconds
  // (in case if mutations happened in 'settings' named module)
  store.subscribe(mutation => {
    if (mutation.type.startsWith("settings/")) {
      if (saveSettingsTimeout != null) clearTimeout(saveSettingsTimeout);
      saveSettingsTimeout = setTimeout(() => {
        SaveSettings();
      }, 5000);
    }
  });
}

export function SaveSettings() {
  if (saveSettingsTimeout == null) return;

  clearTimeout(saveSettingsTimeout);
  saveSettingsTimeout = null;

  try {
    let data = JSON.stringify(store.state.settings, null, 2);
    fs.writeFileSync(filename, data);
  } catch (e) {
    console.error("Failed to save settings:" + e);
  }
}

function combineMerge(target, source, options) {
  const emptyTarget = value => (Array.isArray(value) ? [] : {});
  const clone = (value, options) => merge(emptyTarget(value), value, options);
  const destination = target.slice();

  source.forEach(function(e, i) {
    if (typeof destination[i] === "undefined") {
      const cloneRequested = options.clone !== false;
      const shouldClone = cloneRequested && options.isMergeableObject(e);
      destination[i] = shouldClone ? clone(e, options) : e;
    } else if (options.isMergeableObject(e)) {
      destination[i] = merge(target[i], e, options);
    } else if (target.indexOf(e) === -1) {
      destination.push(e);
    }
  });

  return destination;
}
