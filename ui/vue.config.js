module.exports = {
  configureWebpack: config => {
    /* possibility to debug from VS code */
    config.devtool = "source-map";
  },

  pluginOptions: {
    electronBuilder: {
      preload: "src/preload.js",
      //nodeIntegration: true,
      builderOptions: {
        // options placed here will be merged with default configuration and passed to electron-builder

        mac: {
          // do not build DMG. We do not need it
          "target": "dir",

          extendInfo: {
            // this section contains extendend elements to be added to Info.plist
            LSUIElement: 1, // ability to hide app icon from macOS dock
            SUPublicDSAKeyFile: "dsa_pub.pem" // possibility to perform Sparkle automatic update from old version of IVPN Client
          }
        },

        win: {
          extraResources: [
            {
              from: "public/tray/windows",
              to: "tray/windows",
              filter: ["**/*"]
            }
          ]
        },

        extraResources: [
          {
            from: "extraResources",
            filter: ["**/*"]
          }
        ]
      }
    }
  }
};
