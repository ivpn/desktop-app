const { contextBridge } = require("electron");
import sender from "@/ipc/renderer-sender";

contextBridge.exposeInMainWorld("ipcSender", sender);
