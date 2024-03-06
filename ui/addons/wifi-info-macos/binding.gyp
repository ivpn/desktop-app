{
  "targets": [
    {
      "target_name": "wifi-info-macos-native",
      "sources": [ "src/addon.m" ],      
      "cflags!": ["-fno-exceptions"],
      "cflags_cc!": ["-fno-exceptions"],
      "libraries": [ "-framework CoreLocation", "-framework ServiceManagement" ],
      "xcode_settings": {
        "OTHER_CFLAGS": ["-fno-exceptions", "-ObjC"],       
        'MACOSX_DEPLOYMENT_TARGET': '14.0'     
      }
    }
  ]
}
