import * as Sentry from "@sentry/electron";

import { DSN } from "./dsn";

function beforeSendFunc(event) {
  if (event._isAllowedToSend === true) {
    // breadcrumbs is not informative for diagnostic report
    event.breadcrumbs = null;
    // remove internal properties
    delete event._isAllowedToSend;

    return event;
  }
  return null;
}

export function SentryIsAbleToUse() {
  if (!DSN) return false;
  return true;
}

export function SentryInit() {
  if (!DSN) {
    console.error(
      "Sentry DSN is not defined. Sending diagnostic reports functionality will not work"
    );
    return;
  }

  try {
    Sentry.init({
      dsn: DSN,
      beforeSend: beforeSendFunc, // allow us to control when data can be sent on server
      enableJavaScript: false, // Enables crash reporting for JavaScript errors in this process.
      enableUnresponsive: false, // Enables event reporting for BrowserWindow 'unresponsive' events
      useSentryMinidumpUploader: false, // Enables the Sentry internal uploader for minidumps.
    });

    // Disable sentry by default.
    // It will be temporary enabled only when user wants to send diagnostic report.
    // Note! It is important to have sentry disabled when app is going to close.
    //    Otherwise, there could be situations when Sentry block app to quit (for example, when internet connectivity is blocked)
    Sentry.getCurrentHub().getClient().getOptions().enabled = false;
  } catch (e) {
    console.error(e);
  }
}

export function SentrySendDiagnosticReport(
  AccountID,
  comment,
  eventAdditionalDataObject,
  daemonVer,
  buildExtraInfo
) {
  if (!DSN || comment == "" || eventAdditionalDataObject == null) return;

  if (!daemonVer) daemonVer = "UNKNOWN";
  // Sentry can not accept very long fields (>16KB)
  // therefore, here we are dividing fields on smaller
  const maxFieldSize = 16 * 1024;
  let objectToSend = {};

  // function to divide string on chunks of concrete length (ignore new lines)
  function chunkString(str, length) {
    return str.match(new RegExp("[^]{1," + length + "}", "g"));
  }

  for (var propName in eventAdditionalDataObject) {
    // ignore empty fields
    if (
      eventAdditionalDataObject[propName] == "" ||
      eventAdditionalDataObject[propName] === null ||
      eventAdditionalDataObject[propName] === undefined
    ) {
      continue;
    }

    if (eventAdditionalDataObject[propName].length <= maxFieldSize)
      objectToSend[propName] = eventAdditionalDataObject[propName];
    else {
      // divide field data on multiple smaller portions
      let fields = chunkString(
        eventAdditionalDataObject[propName],
        maxFieldSize
      );
      for (let i = 0; i < fields.length; i++) {
        objectToSend[`${propName} ${i}`] = fields[i];
      }
    }
  }

  try {
    // buildExtraInfo
    let tags = {
      AccountID: AccountID,
      DaemonVersion: daemonVer,
    };
    if (buildExtraInfo) tags.BuildExInfo = buildExtraInfo;

    // Temporary enable Sentry to send Event
    Sentry.getCurrentHub().getClient().getOptions().enabled = true;

    return Sentry.captureEvent({
      _isAllowedToSend: true,
      message: `Diagnostic report`,
      extra: objectToSend,
      contexts: {
        ["comment"]: { "User comment": comment },
      },
      tags: tags,
    });
  } catch (e) {
    console.error(e);
  } finally {
    // disable Sentry after event sent
    Sentry.getCurrentHub().getClient().getOptions().enabled = false;
  }
  return null;
}
