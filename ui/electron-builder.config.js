module.exports = {
  files: [
    "!**/.vscode/*",
    "!src/*",
    "!**/buildHooks/*",
    "!**/public/*",
    "!**/References/*",
    "!**/extraResources/*",
    "!{index.html,pre_build.js}",
    "!electron-builder.config.{js,ts,mjs,cjs}",
    "!electron.vite.config.{js,ts,mjs,cjs}",
    "!{.eslintignore,.eslintrc.cjs,.prettierignore,.prettierrc.yaml,dev-app-update.yml,CHANGELOG.md,README.md}",
    "!{.env,.env.*,.npmrc,pnpm-lock.yaml}",
  ],
  
  afterPack: "buildHooks/afterPack.js",

  mac: {
    target: "dir",
    icon: "buildResources/appicons/icon.icns",
    extendInfo: {
      LSUIElement: 1,
      SUPublicDSAKeyFile: "dsa_pub.pem",
      NSLocationUsageDescription: "IVPN requires location access to correctly detect WIFI network info",
      NSLocationAlwaysAndWhenInUseUsageDescription: "IVPN requires location access to correctly detect WIFI network info",
    },
  },
  win: {
    target: "dir",
    icon: "buildResources/appicons/icon.ico",
  },
  linux: {
    target: "dir",
    icon: "buildResources/appicons/icons",
    category: "Network",
  },
  snap: {
    confinement: "strict",
    autoStart: true,
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
};