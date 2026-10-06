{
  "targets": [
    {
      "target_name": "split-tunnel-macos-native",
      "sources": [ "src/addon.m" ],
      "cflags!": ["-fno-exceptions"],
      "cflags_cc!": ["-fno-exceptions"],
      "libraries": [ "-framework Foundation", "-framework NetworkExtension", "-framework SystemExtensions" ],
      "xcode_settings": {
        "OTHER_CFLAGS": ["-fno-exceptions", "-ObjC", "-fobjc-arc"],
        "MACOSX_DEPLOYMENT_TARGET": "12.0"
      }
    }
  ]
}
