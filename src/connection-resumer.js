import store from "@/store";
import daemonClient from "./daemon-client";

let resumerTimer = null;
export function InitConnectionResumer() {
  store.subscribe(mutation => {
    try {
      if (mutation.type === "uiState/pauseConnectionTill") {
        if (resumerTimer != null) clearTimeout(resumerTimer);
        resumerTimer = null;

        const pauseTill = store.state.uiState.pauseConnectionTill;
        if (pauseTill != null) {
          const timeDiff = pauseTill - new Date();
          if (timeDiff > 0) {
            resumerTimer = setTimeout(() => {
              resumeConnection();
            }, timeDiff);
          }
        }
      }
    } catch (e) {
      console.error("Error in InitConnectionResumer handler", e);
    }
  });
}

function resumeConnection() {
  try {
    console.log("Resuming connection");
    daemonClient.ResumeConnection();
  } catch (e) {
    console.error("Failed to resume connection:", e);
  }
}
