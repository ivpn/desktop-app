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

export function isStrNullOrEmpty(str) {
  return !str || str.length === 0;
}

export function enumValueName(theEnum, value) {
  for (var k in theEnum) if (theEnum[k] == value) return k;
  return null;
}

export function isValidURL(str, isIgnoreProtocol) {
  var pattern = new RegExp(
    "^(https?:\\/\\/)" +
    (isIgnoreProtocol === true ? "?" : "") + // protocol
    "((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|" + // domain name
    "((\\d{1,3}\\.){3}\\d{1,3}))" + // OR ip (v4) address
    "(\\:\\d+)?(\\/[-a-z\\d%_.~+]*)*" + // port and path
    "(\\?[;&a-z\\d%_.~+=-]*)?" + // query string
      "(\\#[-a-z\\d_]*)?$",
    "i"
  ); // fragment locator
  return !!pattern.test(str);
}

export function notLinear(k) {
  return easeOutQuart(k);
  //return easeInOutQuart(k);
  //return simpleSin(k);
  //return easeOutBounce(k);
}

export function simpleSin(k) {
  if (k < 0) k = 0;
  if (k > 1) k = 1;

  const k1 = k * Math.PI - Math.PI / 2;
  const y = 0.5 + Math.sin(k1) / 2;
  return y;
}

// https://easings.net/
export function easeInOutQuart(x) {
  return x < 0.5 ? 8 * x * x * x * x : 1 - Math.pow(-2 * x + 2, 4) / 2;
}
// https://easings.net/
export function easeOutQuart(x) {
  return 1 - Math.pow(1 - x, 4);
}
