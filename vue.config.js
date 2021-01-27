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
        mac: {
          extendInfo: {
            LSUIElement: 1 // ability to hide app icon from macOS dock
          }
        }
      }
    }
  }
};
