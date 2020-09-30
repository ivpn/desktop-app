const updateJsonUrl = "https://www.ivpn.net/updates/linux/update.json";
import fetch from "electron-fetch";

export async function CheckUpdates() {
  try {
    var options = { headers: { "Cache-Control": "no-cache" } };
    const response = await fetch(updateJsonUrl, options);
    return await response.json();
  } catch (e) {
    console.error(e);
    return null;
  }
}

export function Upgrade(latestVersionInfo) {
  if (!latestVersionInfo) {
    console.error("Upgrade skipped: no information about latest version");
    return null;
  }

  try {
    require("electron").shell.openExternal(latestVersionInfo.downloadPageLink);
  } catch (e) {
    console.error(e);
  }
}
