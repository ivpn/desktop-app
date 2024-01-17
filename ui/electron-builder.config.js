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
    extendInfo: {
      LSUIElement: 1,
      SUPublicDSAKeyFile: "dsa_pub.pem",
    },
  },
  win: {
    target: "dir",    
  },
  linux: {
    target: "dir",
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