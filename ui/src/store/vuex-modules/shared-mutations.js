//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Modified by Stelnykovych Alexandr
//  Copyright (c) 2023 IVPN Limited.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//
// This sources was inspired by 'vuex-electron' module (https://github.com/vue-electron/vuex-electron)
// https://github.com/vue-electron/vuex-electron/blob/master/src/shared-mutations.js
// Sources was modified (see 'loadOptions()') in order to be able
// to use it with disabled nodeIntegration in Electron BrowserWindow.
// No 'import' required if ipcMain/ipcRenderer was passed in options.
// For example, ipcRenderer can be initialized in 'preload' script.
//
const IPC_EVENT_CONNECT = "vuex-mutations-connect";
const IPC_EVENT_NOTIFY_MAIN = "vuex-mutations-notify-main";
const IPC_EVENT_NOTIFY_RENDERERS = "vuex-mutations-notify-renderers";

class SharedMutations {
  constructor(options, store) {
    this.options = options;
    this.store = store;
  }

  loadOptions() {
    if (!this.options.type)
      this.options.type = process.type === "renderer" ? "renderer" : "main";

    if (this.options.type === "main" && !this.options.ipcMain) {
      this.options.ipcMain = require("electron").ipcMain;
    }
    if (this.options.type === "renderer" && !this.options.ipcRenderer) {
      this.options.ipcRenderer = require("electron").ipcRenderer;
    }
  }

  connect(payload) {
    this.options.ipcRenderer.send(IPC_EVENT_CONNECT, payload);
  }

  onConnect(handler) {
    this.options.ipcMain.on(IPC_EVENT_CONNECT, handler);
  }

  notifyMain(payload) {
    this.options.ipcRenderer.send(IPC_EVENT_NOTIFY_MAIN, payload);
  }

  onNotifyMain(handler) {
    this.options.ipcMain.on(IPC_EVENT_NOTIFY_MAIN, handler);
  }

  notifyRenderers(connections, payload) {
    payload = JSON.parse(JSON.stringify(payload)); // INFO: making payload serializable! // TODO: ensure this is a good approach.

    Object.keys(connections).forEach((processId) => {
      connections[processId].send(IPC_EVENT_NOTIFY_RENDERERS, payload);
    });
  }

  onNotifyRenderers(handler) {
    this.options.ipcRenderer.on(IPC_EVENT_NOTIFY_RENDERERS, handler);
  }

  rendererProcessLogic() {
    // Connect renderer to main process
    this.connect();

    // Save original Vuex methods
    this.store.originalCommit = this.store.commit;
    this.store.originalDispatch = this.store.dispatch;

    // Don't use commit in renderer outside of actions
    this.store.commit = () => {
      throw new Error(
        `[Vuex Electron] Please, don't use direct commit's, use dispatch instead of this.`
      );
    };

    // Forward dispatch to main process
    this.store.dispatch = (type, payload) => {
      payload = JSON.parse(JSON.stringify(payload)); // INFO: making payload serializable! // TODO: ensure this is a good approach.

      this.notifyMain({ type, payload });
    };

    // Subscribe on changes from main process and apply them
    this.onNotifyRenderers((event, { type, payload }) => {
      this.store.originalCommit(type, payload);
    });
  }

  mainProcessLogic() {
    const connections = {};

    // Save new connection
    this.onConnect((event) => {
      const win = event.sender;
      const winId = win.id;

      connections[winId] = win;

      // Remove connection when window is closed
      win.on("destroyed", () => {
        delete connections[winId];
      });
    });

    // Subscribe on changes from renderer processes
    this.onNotifyMain((event, { type, payload }) => {
      this.store.dispatch(type, payload);
    });

    // Subscribe on changes from Vuex store
    this.store.subscribe((mutation) => {
      const { type, payload } = mutation;

      // Forward changes to renderer processes
      this.notifyRenderers(connections, { type, payload });
    });
  }

  activatePlugin() {
    switch (this.options.type) {
      case "renderer":
        this.rendererProcessLogic();
        break;
      case "main":
        this.mainProcessLogic();
        break;
      default:
        throw new Error(`[Vuex Electron] Type should be "renderer" or "main".`);
    }
  }
}

export default (options = {}) =>
  (store) => {
    const sharedMutations = new SharedMutations(options, store);

    sharedMutations.loadOptions();
    sharedMutations.activatePlugin();
  };
