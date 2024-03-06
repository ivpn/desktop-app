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

console.log("[ ] Preparing build...");

const os = require("os");

// PLATFORM SPECIFIC STYLE LOADER
console.log("\n[ ] Copying platform specific style loader...");
var styleLoaderFile = __dirname + "/src/main_style.js";
var styleLoaderFilePlatform =
  __dirname + "/src/main_style_" + os.platform() + ".js";

console.log("Using style loader script: " + styleLoaderFilePlatform);

var fs = require("fs");
if (fs.existsSync(styleLoaderFilePlatform)) {
  fs.copyFileSync(styleLoaderFilePlatform, styleLoaderFile);
} else {
  var mainStyleFileNotExists =
    " *** ERROR: FILE NOT EXISTS: *** '" +
    styleLoaderFilePlatform +
    "'. Is [" +
    os.platform() +
    "] platform supported? ";

  console.error("");
  console.error(mainStyleFileNotExists);
  console.error();

  throw mainStyleFileNotExists;
}

console.log("\n[ ] Build preparation finished.\n");
