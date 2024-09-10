import { getHwyGlobal } from "../../common/index.mjs";
import { dynamicNodePath } from "./url-polyfills.js";

const hwyGlobal = getHwyGlobal();

export const DEV_BUNDLED_CSS_QUERY_PARAM =
  "?NOTE_TO_DEV=this-will-be-hashed-and-cached-in-prod-just-like-your-client-entry-file";

export const DEV_BUNDLED_CSS_LINK =
  "/public/dist/standard-bundled.css" + DEV_BUNDLED_CSS_QUERY_PARAM;

export function getPublicURL(url: string): string {
  let hashedURL: string | undefined;

  if (url.startsWith("/")) url = url.slice(1);
  if (url.startsWith("./")) url = url.slice(2);

  const publicMap = hwyGlobal.get("publicMap");

  if (!dynamicNodePath) {
    throw new Error("dynamicNodePath is not defined");
  }

  hashedURL = publicMap?.[dynamicNodePath.join("public", url)];

  if (!hashedURL) {
    const noNeedToLogList = ["dist/standard-bundled.css", "favicon.ico"];
    if (!noNeedToLogList.includes(url)) {
      console.log("No hashed URL found for", url);
    }
    return "";
  }

  if (hwyGlobal.get("isDev")) {
    const normalizedURL = url.replace(/\\/g, "/");
    if (normalizedURL === "dist/standard-bundled.css") {
      return DEV_BUNDLED_CSS_LINK;
    }
  }

  return "/" + hashedURL;
}

export function getOrigPublicURL(hashedURL: string): string {
  if (!dynamicNodePath) {
    throw new Error("dynamicNodePath is not defined");
  }

  const slicedURL = dynamicNodePath.normalize(hashedURL.slice(1));

  if (hwyGlobal.get("isDev")) {
    const normalizedSlicedURL = slicedURL.replace(/\\/g, "/");

    if (normalizedSlicedURL.startsWith("public/dist/standard-bundled")) {
      return "./" + "public/dist/standard-bundled.css";
    }
  }

  if (slicedURL.includes("hwy_chunk__") || slicedURL.includes("hwy_entry__")) {
    return "./" + slicedURL;
  }

  const reversePublicMap = hwyGlobal.get("publicReverseMap");

  const origURL = reversePublicMap?.[slicedURL];

  if (!origURL) {
    throw new Error(`No original URL found for ${slicedURL}`);
  }

  return "./" + origURL;
}
