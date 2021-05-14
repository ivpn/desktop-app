//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
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
