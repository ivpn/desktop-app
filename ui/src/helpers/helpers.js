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

export function enumValueName(theEnum, value) {
  for (var k in theEnum) if (theEnum[k] == value) return k;
  return null;
}

export function IsRenderer() {
  if (typeof window !== "undefined") return true;
  return false;
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

const RegExpIpv4Addr = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)(\.(?!$)|$)){4}$/;
const RegExpIpv6Addr =
  /((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))/;

export function isValidIPv4(ipStr) {
  return RegExpIpv4Addr.test(ipStr);
}
export function isValidIPv6(ipStr) {
  return RegExpIpv6Addr.test(ipStr);
}
export function isValidIpOrMask(ipStr) {
  if (!ipStr) return false;

  const parts = ipStr.split("/");
  if (parts.length > 2 || parts.length == 0) return false;

  let ip = ipStr;
  if (parts.length == 2) ip = parts[0];
  const isIpv4 = RegExpIpv4Addr.test(ip);
  const isIpv6 = RegExpIpv6Addr.test(ip);

  if (!isIpv4 && !isIpv6) return false;
  if (parts.length == 2) {
    const parsed = parseInt(parts[1], 10);
    if (isNaN(parsed)) return false;
    if (isIpv4 && (parsed > 32 || parsed <= 0)) return false;
    if (isIpv6 && (parsed > 128 || parsed <= 0)) return false;
  }
  return true;
}

export function capitalizeFirstLetter(string) {
  return string.charAt(0).toUpperCase() + string.slice(1);
}

export function dateDefaultFormat(date) {
  return dateYyyyMonDd(date);
}

export function dateYyyyMonDd(date) {
  const monthNames = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];

  var mm = date.getMonth();
  var dd = date.getDate();

  return [date.getFullYear(), monthNames[mm], (dd > 9 ? "" : "0") + dd].join(
    "-"
  );
}

export function dateYyyyMmDd(date) {
  var mm = date.getMonth() + 1; // getMonth() is zero-based
  var dd = date.getDate();

  return [
    date.getFullYear(),
    (mm > 9 ? "" : "0") + mm,
    (dd > 9 ? "" : "0") + dd,
  ].join("-");
}

export function notLinear(k) {
  return easeOutQuart(k);
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

// MAP COORDINATES
// fit coordinates to map image
function toRadian(value) {
  return (value * Math.PI) / 180.0;
}
export function getCoordinatesBy(
  longitude,
  latitude,
  bitmapWidth,
  bitmapHeight
) {
  if (bitmapWidth == null) bitmapWidth = 11300;
  if (bitmapHeight == null) bitmapHeight = 8249;

  let x = toRadian(longitude) - 0.18;
  let y = toRadian(latitude);

  let yStrech = 0.542;
  let yOffset = 0.053;
  y = yStrech * Math.log(Math.tan(0.25 * Math.PI + 0.4 * y)) + yOffset;

  x = bitmapWidth / 2 + (bitmapWidth / (2 * Math.PI)) * x;
  y = bitmapHeight / 2 - (bitmapHeight / 2) * y;

  return { x, y };
}

function fromRadian(r) {
  return (r * 180.0) / Math.PI;
}

export function getPosFromCoordinates(x, y, bitmapWidth, bitmapHeight) {
  if (bitmapWidth == null) bitmapWidth = 11300;
  if (bitmapHeight == null) bitmapHeight = 8249;

  y = (y - bitmapHeight / 2) / (-bitmapHeight / 2);
  x = (x - bitmapWidth / 2) / (bitmapWidth / (2 * Math.PI));

  let yStrech = 0.542;
  let yOffset = 0.053;
  y =
    (Math.atan(Math.pow(Math.E, (y - yOffset) / yStrech)) - 0.25 * Math.PI) /
    0.4;

  let latitude = fromRadian(y);
  let longitude = fromRadian(x + 0.18);

  return { longitude, latitude };
}

export function getDistanceFromLatLonInKm(lat1, lon1, lat2, lon2) {
  var R = 6371; // Radius of the earth in km
  var dLat = deg2rad(lat2 - lat1); // deg2rad below
  var dLon = deg2rad(lon2 - lon1);
  var a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(deg2rad(lat1)) *
      Math.cos(deg2rad(lat2)) *
      Math.sin(dLon / 2) *
      Math.sin(dLon / 2);
  var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  var d = R * c; // Distance in km
  return d;
}

function deg2rad(deg) {
  return deg * (Math.PI / 180);
}
