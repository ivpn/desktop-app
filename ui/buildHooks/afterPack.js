var fs = require ('fs');
const path = require('path');

exports.default = async function (context) {
  console.log (`AfterPack hook triggered ('${__filename}')`);

  // In order to reduce the size of the app, we remove all unused locales
  try {
    if (context?.packager?.platform?.buildConfigurationKey === 'mac') {
      let localeDir = context.appOutDir+'/IVPN.app/Contents/Frameworks/Electron Framework.framework/Resources/';    
      if (fs.existsSync(localeDir))
        removeLocalesMac(localeDir);
    } else {
      let localeDir = context.appOutDir+'/locales/';
      if (fs.existsSync(localeDir))
        removeLocales(localeDir);
    }
  } catch (e) {
    console.error("Error removing locales in afterPack hook:", e);
  }
}

function removeLocales(localesFolderPath) {
  console.log (` - removing all locales except en-US (from '${localesFolderPath}')`);
  let removedCnt = 0;
  let files = fs.readdirSync(localesFolderPath);
  if (files && files.length) {
      for  (var i = 0, len = files.length; i < len; i++) {
          var match = files[i].match(/en-US\.pak/);
          if (match === null) {
              fs.unlinkSync(path.join(localesFolderPath, files[i]));
              removedCnt+=1;
          }
      }
  }
  console.log (`   removed ${removedCnt} locales`);
}

function removeLocalesMac(resourcesFolderPath) {
  console.log (` - removing all locales except en/en-US (from '${resourcesFolderPath}')`);
  let removedCnt = 0;
  let files = fs.readdirSync(resourcesFolderPath);
  if (files && files.length) {
      for  (var i = 0, len = files.length; i < len; i++) {
          const lprojDir = files[i];
          if (lprojDir === 'en.lproj' || lprojDir === 'en-US.lproj' || lprojDir === 'en_US.lproj') 
              continue;
          const lprojDirPath = path.join(resourcesFolderPath, files[i]);
          const localePakPath = path.join(lprojDirPath,"locale.pak");
          if (fs.lstatSync(lprojDirPath).isDirectory() && path.extname(lprojDirPath) === '.lproj' && fs.existsSync(localePakPath)) {        
              fs.unlinkSync(localePakPath);
              fs.rmdirSync(lprojDirPath);
              removedCnt+=1;
          }
      }
  }
  console.log (`   removed ${removedCnt} locales`);
}