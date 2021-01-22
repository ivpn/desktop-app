import { Menu, ipcMain } from "electron";

ipcMain.on("renderer-request-show-context-menu-copy", () => {
  InputMenuLabel.popup();
});

ipcMain.on("renderer-request-show-context-menu-edit", () => {
  InputMenuInput.popup();
});

// Default COPY/PASTE context menu for all input elements
const InputMenuInput = Menu.buildFromTemplate([
  {
    label: "Undo",
    role: "undo"
  },
  {
    label: "Redo",
    role: "redo"
  },
  {
    type: "separator"
  },
  {
    label: "Cut",
    role: "cut"
  },
  {
    label: "Copy",
    role: "copy"
  },
  {
    label: "Paste",
    role: "paste"
  },
  {
    type: "separator"
  },
  {
    label: "Select all",
    role: "selectall"
  }
]);
// Default COPY context menu for all label elements
const InputMenuLabel = Menu.buildFromTemplate([
  {
    label: "Copy",
    role: "copy"
  },
  {
    label: "Select all",
    role: "selectall"
  }
]);
