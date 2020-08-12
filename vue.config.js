module.exports = {
  configureWebpack: config => {
    /* This is important! Preventing errors in renderer thread like : 'Uncaught TypeError: fs.existsSync is not a function' */
    config.target = "electron-renderer";

    /* possibility to debug from VS code */
    config.devtool = "source-map";
  },

  pluginOptions: {
    electronBuilder: {
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
