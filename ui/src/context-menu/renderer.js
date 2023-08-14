//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

const sender = window.ipcSender;

const keyCodes = {
  V: 86,
  C: 67,
  X: 88,
  A: 65,
};

export function InitDefaultCopyMenus() {
  document.body.addEventListener("contextmenu", (e) => {
    e.preventDefault();
    e.stopPropagation();

    let node = e.target;

    while (node) {
      if (
        node.nodeName.match(/^(input|textarea)$/i) ||
        node.isContentEditable
      ) {
        sender.ShowContextMenuEdit();
        break;
      } else if (node.nodeName.match(/^(label)$/i)) {
        if (getSelection().toString()) {
          sender.ShowContextMenuCopy();
        }
        break;
      }
      node = node.parentNode;
    }
  });

  // Ability to get working Copy\Paste to 'input' elements
  // without modification application menu (which is required for macOS)
  document.onkeydown = function (event) {
    if (event.ctrlKey || event.metaKey) {
      // detect ctrl or cmd
      const field = document.activeElement;
      switch (event.which) {
        case keyCodes.A:
          document.execCommand("selectall");
          return false;
        case keyCodes.V:
          if (field != null) document.execCommand("paste");
          return false;
        case keyCodes.C:
          document.execCommand("copy");
          return false;
        case keyCodes.X:
          if (field != null) document.execCommand("cut");
          return false;
      }
    }
  };
}
