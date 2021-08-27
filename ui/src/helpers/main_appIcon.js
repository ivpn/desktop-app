import { extractIcon } from "@bitdisaster/exe-icon-extractor";
import { Platform, PlatformEnum } from "@/platform/platform";

export function extractBinaryIcon(binaryPath) {
  try {
    if (Platform() != PlatformEnum.Windows) return null;

    // get icon
    const buffer = extractIcon(binaryPath, "large");

    // convert icon to base64
    let b = Buffer.from(buffer);
    let imageEncoded = b.toString("base64");
    imageEncoded = "data:image/x-icon;base64," + imageEncoded;

    return imageEncoded;
  } catch (e) {
    console.warn(`Failed to obtain app icon for '${binaryPath}:' ` + e);
    return null;
  }
}
