import client from "@/daemon-client";
import config from "@/config";

export async function CheckUpdates() {
  try {
    return await client.GetAppUpdateInfo();
  } catch (e) {
    if (e instanceof SyntaxError)
      console.error("[updater] parsing update file info error: ", e.message);
    else console.error("[updater] error: ", e);

    return null;
  }
}

export function Upgrade(latestVersionInfo) {
  if (!latestVersionInfo) {
    console.error("Upgrade skipped: no information about latest version");
    return null;
  }

  try {
    require("electron").shell.openExternal(config.URLApps);
  } catch (e) {
    console.error(e);
  }
}
