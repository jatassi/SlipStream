// A verify or browser-tier tab opens past the startup sequence, as a tab that
// has played it does. The key is prefKey("startupPlayed") in
// web/src/shell/prefs.ts, which e2e/startup.spec.ts proves.
if (location.protocol === "http:") {
  sessionStorage.setItem("slipstream.startupPlayed", "true");
}
