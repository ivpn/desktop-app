export function IsOsDarkColorScheme() {
  //matchMedia method not supported
  if (!window.matchMedia) return false;
  //OS theme setting detected as dark
  if (window.matchMedia("(prefers-color-scheme: dark)").matches) return true;
  return false;
}
