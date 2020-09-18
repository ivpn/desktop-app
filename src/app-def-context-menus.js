//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui-beta
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

const electron = require("electron");
const remote = electron.remote;
const Menu = remote.Menu;

const { clipboard } = require("electron");
const keyCodes = {
  V: 86,
  C: 67,
  X: 88,
  A: 65
};

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

export function InitDefaultCopyMenus() {
  document.body.addEventListener("contextmenu", e => {
    e.preventDefault();
    e.stopPropagation();

    let node = e.target;

    while (node) {
      if (
        node.nodeName.match(/^(input|textarea)$/i) ||
        node.isContentEditable
      ) {
        InputMenuInput.popup(remote.getCurrentWindow());
        break;
      }
      else if (
        node.nodeName.match(/^(label)$/i)
      ) {
        if (getSelection().toString()) {
          InputMenuLabel.popup(remote.getCurrentWindow());
        }
        break;
      }
      node = node.parentNode;
    } 
  });

  // Ability to get working Copy\Paste to 'input' elements
  // without modification application menu (which is required for macOS)
  document.onkeydown = function(event) {
    let toReturn = true;
    if (event.ctrlKey || event.metaKey) {
      // detect ctrl or cmd
      if (event.which == keyCodes.A) {
        const field = document.activeElement;
        if (field != null) field.select();
        toReturn = false;
      } else if (event.which == keyCodes.V) {
        const field = document.activeElement;
        if (field != null) {
          const startPos = field.selectionStart;
          const endPos = field.selectionEnd;

          const text = clipboard.readText();

          field.value =
            field.value.substring(0, startPos) +
            text +
            field.value.substring(endPos, field.value.length);

          field.focus();
          field.setSelectionRange(
            startPos + text.length,
            startPos + text.length
          );

          toReturn = false;
        }
      } else if (event.which == keyCodes.C) {
        clipboard.writeText(getSelection().toString());
        toReturn = false;
      } else if (event.which == keyCodes.X) {
        const field = document.activeElement;
        if (field != null) {
          let selection = getSelection();
          clipboard.writeText(selection.toString());

          const startPos = field.selectionStart;
          const endPos = field.selectionEnd;

          field.value =
            field.value.slice(0, startPos) + field.value.slice(endPos);

          field.focus();
          field.setSelectionRange(startPos, startPos);

          toReturn = false;
        }
      }
    }
    return toReturn;
  };
}