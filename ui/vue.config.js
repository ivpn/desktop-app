module.exports = {
  configureWebpack: {
    devtool: "source-map", // possibility to debug from VS Code

    // required after moving from webpack < 5 (do not include polyfill)
    resolve: {
      fallback: {
        os: false,
        fs: false,
        path: false,
      },
    },
  },

  pluginOptions: {
    electronBuilder: {
      preload: "src/preload.js",
      //nodeIntegration: true,
      builderOptions: {
        // options placed here will be merged with default configuration and passed to electron-builder

        mac: {
          // do not build DMG. We do not need it
          target: "dir",

          extendInfo: {
            // this section contains extendend elements to be added to Info.plist
            LSUIElement: 1, // ability to hide app icon from macOS dock
            SUPublicDSAKeyFile: "dsa_pub.pem", // possibility to perform Sparkle automatic update from old version of IVPN Client
          },
        },

        win: {
          // do not build exe installer. We do not need it
          target: "dir",

          extraResources: [
            {
              from: "public/tray/windows",
              to: "tray/windows",
              filter: ["**/*"],
            },
          ],
        },

        linux: {
          //target: ["dir", "snap"],
          target: ["dir"],
          category: "Network",
        },

        snap: {
          confinement: "strict",
          autoStart: true, // ability to autostart (when file exists: '$SNAP_USER_DATA/.config/autostart/ivpn-ui.desktop')
          //stagePackages: ["default", "ivpn"],
          plugs: [
            "default",
            {
              port: {
                interface: "content",
                content: "file",
                target: "$SNAP_COMMON/opt/ivpn/mutable",
              },
            },
          ],
        },

        extraResources: [
          {
            from: "extraResources",
            filter: ["**/*"],
          },
        ],
      },
    },
  },
};
