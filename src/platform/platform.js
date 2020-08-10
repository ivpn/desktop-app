export const PlatformEnum = Object.freeze({
  unknown: 0,
  macOS: 1,
  Linux: 2,
  Windows: 3
});

export function Platform() {
  //return PlatformEnum.macOS;
  //return PlatformEnum.Linux;
  //return PlatformEnum.Windows;

  const os = require("os");
  switch (os.platform()) {
    case "win32":
      return PlatformEnum.Windows;
    case "linux":
      return PlatformEnum.Linux;
    case "darwin":
      return PlatformEnum.macOS;
    default:
      return PlatformEnum.unknown;
  }
}

export function IsWindowHasTitle() {
  return Platform() !== PlatformEnum.macOS;
}
