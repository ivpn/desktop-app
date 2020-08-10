export function isStrNullOrEmpty(str) {
  return !str || str.length === 0;
}

export function enumValueName(theEnum, value) {
  for (var k in theEnum) if (theEnum[k] == value) return k;
  return null;
}

export function isValidURL(str, isIgnoreProtocol) {
  var pattern = new RegExp(
    "^(https?:\\/\\/)" +
    (isIgnoreProtocol === true ? "?" : "") + // protocol
    "((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|" + // domain name
    "((\\d{1,3}\\.){3}\\d{1,3}))" + // OR ip (v4) address
    "(\\:\\d+)?(\\/[-a-z\\d%_.~+]*)*" + // port and path
    "(\\?[;&a-z\\d%_.~+=-]*)?" + // query string
      "(\\#[-a-z\\d_]*)?$",
    "i"
  ); // fragment locator
  return !!pattern.test(str);
}

export function notLinear(k) {
  return easeOutQuart(k);
  //return easeInOutQuart(k);
  //return simpleSin(k);
  //return easeOutBounce(k);
}

export function simpleSin(k) {
  if (k < 0) k = 0;
  if (k > 1) k = 1;

  const k1 = k * Math.PI - Math.PI / 2;
  const y = 0.5 + Math.sin(k1) / 2;
  return y;
}

// https://easings.net/
export function easeInOutQuart(x) {
  return x < 0.5 ? 8 * x * x * x * x : 1 - Math.pow(-2 * x + 2, 4) / 2;
}
// https://easings.net/
export function easeOutQuart(x) {
  return 1 - Math.pow(1 - x, 4);
}
