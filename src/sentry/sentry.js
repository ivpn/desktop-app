import * as Sentry from "@sentry/electron";

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

export function SentryInit() {
  Sentry.init({
    dsn: "https://bdf33f76e775412c8a57388556a8338e@crashes.ivpn.net/6",

    beforeSend: beforeSendFunc, // allow us to control when data can be sent on server

    enableJavaScript: false, // Enables crash reporting for JavaScript errors in this process.
    enableUnresponsive: false, // Enables event reporting for BrowserWindow 'unresponsive' events
    useSentryMinidumpUploader: false // Enables the Sentry internal uploader for minidumps.
  });
}

export function SentrySendDiagnosticReport(
  AccountID,
  comment,
  eventAdditionalDataObject
) {
  if (comment == "" || eventAdditionalDataObject == null) return;

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
    return Sentry.captureEvent({
      _isAllowedToSend: true,
      message: `Diagnostic report`,
      extra: objectToSend,
      contexts: {
        ["comment"]: { "User comment": comment }
      },
      tags: {
        AccountID: AccountID
      }
    });
  } catch (e) {
    console.error(e);
  }
  return null;
}
