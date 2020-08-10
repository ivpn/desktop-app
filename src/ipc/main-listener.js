import client from "../daemon-client";
const { ipcMain } = require("electron");
import store from "@/store";

ipcMain.handle("renderer-request-connect-to-daemon", async () => {
  return await client.ConnectToDaemon();
});
ipcMain.handle("renderer-request-refresh-storage", async () => {
  // function using to re-apply all mutations
  // This is required to send to renderer processes current storage state
  store.commit("replaceState", store.state);
});

ipcMain.handle("renderer-request-login", async (event, accountID, force) => {
  return await client.Login(accountID, force);
});

ipcMain.handle("renderer-request-logout", async () => {
  return await client.Logout();
});

ipcMain.handle("renderer-request-account-status", async () => {
  return await client.AccountStatus();
});

ipcMain.handle("renderer-request-ping-servers", async () => {
  return client.PingServers();
});

ipcMain.handle(
  "renderer-request-connect",
  async (event, entryServer, exitServer) => {
    return await client.Connect(entryServer, exitServer);
  }
);
ipcMain.handle("renderer-request-disconnect", async () => {
  return await client.Disconnect();
});

ipcMain.handle("renderer-request-pause-connection", async () => {
  return await client.PauseConnection();
});
ipcMain.handle("renderer-request-resume-connection", async () => {
  return await client.ResumeConnection();
});

ipcMain.handle("renderer-request-firewall", async (event, enable) => {
  return await client.EnableFirewall(enable);
});
ipcMain.handle(
  "renderer-request-KillSwitchSetAllowLANMulticast",
  async (event, enable) => {
    return await client.KillSwitchSetAllowLANMulticast(enable);
  }
);
ipcMain.handle(
  "renderer-request-KillSwitchSetAllowLAN",
  async (event, enable) => {
    return await client.KillSwitchSetAllowLAN(enable);
  }
);
ipcMain.handle(
  "renderer-request-KillSwitchSetIsPersistent",
  async (event, enable) => {
    return await client.KillSwitchSetIsPersistent(enable);
  }
);

ipcMain.handle("renderer-request-set-logging", async () => {
  return await client.SetLogging();
});
ipcMain.handle("renderer-request-set-obfsproxy", async () => {
  return await client.SetObfsproxy();
});

ipcMain.handle(
  "renderer-request-set-dns",
  async (event, antitrackerIsEnabled) => {
    return await client.SetDNS(antitrackerIsEnabled);
  }
);

ipcMain.handle("renderer-request-geolookup", async () => {
  const api = require("@/api");
  return await api.default.GeoLookup();
});

ipcMain.handle("renderer-request-wg-regenerate-keys", async () => {
  return await client.WgRegenerateKeys();
});

ipcMain.handle(
  "renderer-request-wg-set-keys-rotation-interval",
  async (event, intervalSec) => {
    return await client.WgSetKeysRotationInterval(intervalSec);
  }
);

// renderer-request-wg-set-keys-rotation-interval
