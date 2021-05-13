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

export function IsOsDarkColorScheme() {
  //matchMedia method not supported
  if (!window.matchMedia) return false;
  //OS theme setting detected as dark
  if (window.matchMedia("(prefers-color-scheme: dark)").matches) return true;
  return false;
}

export function GetTimeLeftText(endTime /*Date()*/) {
  if (endTime == null) return "";

  let secondsLeft = (endTime - new Date()) / 1000;
  if (secondsLeft <= 0) return "";

  function two(i) {
    if (i < 10) i = "0" + i;
    return i;
  }

  const h = Math.floor(secondsLeft / (60 * 60));
  const m = Math.floor((secondsLeft - h * 60 * 60) / 60);
  const s = Math.floor(secondsLeft - h * 60 * 60 - m * 60);
  return `${two(h)} : ${two(m)} : ${two(s)}`;
}
