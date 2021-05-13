//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-ui2
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

export function IsNewVersion(oldVer, newVer) {
  if (!oldVer || !newVer) return false;
  oldVer = oldVer.trim();
  newVer = newVer.trim();
  if (!oldVer || !newVer) return false;

  const newVerStrings = newVer.split(".");
  const curVerStrings = oldVer.split(".");

  try {
    for (let i = 0; i < newVerStrings.length && i < curVerStrings.length; i++) {
      if (parseInt(newVerStrings[i], 10) > parseInt(curVerStrings[i], 10))
        return true;
      if (parseInt(newVerStrings[i], 10) < parseInt(curVerStrings[i], 10))
        return false;
    }

    if (newVerStrings.length > curVerStrings.length) {
      for (let i = curVerStrings.length; i < newVerStrings.length; i++) {
        if (parseInt(newVerStrings[i], 10) > 0) return true;
      }
    }
  } catch (e) {
    console.log(e);
  }
  return false;
}
