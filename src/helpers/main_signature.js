const fs = require("fs");
const os = require("os");
const path = require("path");

import { GetOpenSSLBinaryPath } from "./main_platform";

// SIGNING EXAMPLE
//    sign:
//        openssl dgst -sha256 -sign private.pem -out sign.sha256 IVPN-2.12.17.dmg
//    encode to base64:
//        openssl base64 -in sign.sha256 -out sign.sha256.base64
// SIGNATURE CHECK
//    decode base64:
//        openssl base64 -d -in sign.sha256.base64 -out sign2.sha256
//    check sign:
//        openssl dgst -sha256 -verify public.pem -signature sign2.sha256 IVPN-2.12.17.dmg

async function executeOpenSSlCommand(opensslArgs) {
  let opensslBin = await GetOpenSSLBinaryPath();

  return new Promise((resolve, reject) => {
    let spawn = require("child_process").spawn;

    let logStringPrefix = "[OpenSSL call]";

    try {
      let cmd = spawn(opensslBin, opensslArgs);

      cmd.on("error", err => {
        console.log(`[ERROR] ${logStringPrefix}: ${err}`);
        reject(err);
      });

      cmd.on("exit", code => {
        resolve(code);
      });
    } catch (e) {
      console.log(`Failed to run ${logStringPrefix}: ${e}`);
      reject(e);
    }
  });
}

export async function ValidateFileOpenSSLCertificate(
  fileToValidate,
  signSha256FileBase64
) {
  let tmpSignSha256File = signSha256FileBase64
    .split(".")
    .slice(0, -1)
    .join(".");

  // eslint-disable-next-line no-undef
  let pubKeyPathInternal = path.join(__static, "update", "public.pem");
  let pubKeyPath = path.join(os.tmpdir(), "tmp_ivpn_public.pem");
  try {
    fs.copyFileSync(pubKeyPathInternal, pubKeyPath);
    //    decode base64:
    //        openssl base64 -d -in sign.sha256.base64 -out sign2.sha256
    let retCode = await executeOpenSSlCommand([
      "base64",
      "-d",
      "-in",
      signSha256FileBase64,
      "-out",
      tmpSignSha256File
    ]);
    if (retCode != 0) return false;

    //    check sign:
    //        openssl dgst -sha256 -verify public.pem -signature sign2.sha256 IVPN-2.12.17.dmg
    retCode = await executeOpenSSlCommand([
      "dgst",
      "-sha256",
      "-verify",
      pubKeyPath,
      "-signature",
      tmpSignSha256File,
      fileToValidate
    ]);
    if (retCode != 0) return false;
  } catch (err) {
    console.error("Signature verification error: ", err);
    return false;
  } finally {
    try {
      fs.unlinkSync(pubKeyPath);
      fs.unlinkSync(tmpSignSha256File);
    } catch (e) {
      console.warn(e);
    }
  }

  return true;
}

export async function ValidateDataOpenSSLCertificate(
  dataToValidate,
  sha256SignatureInBase64,
  dataName
) {
  let tmpFileToValidate = path.join(os.tmpdir(), `tmp_${dataName}`);
  let tmpSignFileBase64 = tmpFileToValidate + ".sign.sha256.base64";
  try {
    fs.writeFileSync(tmpFileToValidate, dataToValidate);
    fs.writeFileSync(tmpSignFileBase64, sha256SignatureInBase64);

    let retValue = await ValidateFileOpenSSLCertificate(
      tmpFileToValidate,
      tmpSignFileBase64
    );

    return retValue;
  } catch (err) {
    console.error("Signature verification error: ", err);
    return false;
  } finally {
    try {
      fs.unlinkSync(tmpFileToValidate);
      fs.unlinkSync(tmpSignFileBase64);
    } catch (e) {
      console.warn(e);
    }
  }
}
