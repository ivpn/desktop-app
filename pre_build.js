const os = require("os");

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
