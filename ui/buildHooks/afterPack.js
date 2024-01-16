exports.default = async function (context) {
   console.log (`afterPack hook triggered (${__filename})`);
   console.log (" - removing all locales except en-US");
   var fs = require ('fs');
   var localeDir = context.appOutDir+'/locales/';
 
   fs.readdir (localeDir, function (err, files) {
     if (! (files && files.length)) return;
     for  (var i = 0, len = files.length; i < len; i++) {
       var match = files[i].match(/en-US\.pak/);
       if (match === null) {
         fs.unlinkSync (localeDir+files [i]);
       }
     }
   });
 }
 