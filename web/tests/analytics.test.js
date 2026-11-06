import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import { startAnalytics } from "../browser/analytics.js";

let options;
class MockOpenPanel { constructor(value) { options = value; } }

function browser(path = "/checkout/success?session_id=private#fragment", host = "trackbaseline.com") {
  const url = new URL(`https://${host}${path}`);
  globalThis.window = { location: url, localStorage: { getItem: () => null } };
  Object.defineProperty(globalThis, "navigator", { value: {}, configurable: true });
  options = undefined;
}

afterEach(() => { delete globalThis.window; delete globalThis.navigator; });

test("only a production public page initializes automatic screen views", () => {
  browser(); startAnalytics(MockOpenPanel);
  assert.equal(options.clientId, "f6d278c9-847b-4420-8867-a22ce7b7e3fd");
  assert.equal(options.trackScreenViews, true);
  assert.equal(options.trackOutgoingLinks, false);
  assert.equal(options.trackAttributes, false);
  const event = { type: "track", payload: { name: "screen_view", properties: { __path: window.location.href, __referrer: "https://trackbaseline.com/checkout?secret=1#x" } } };
  assert.equal(options.filter(event), true);
  assert.equal(event.payload.properties.__path, "https://trackbaseline.com/checkout/success");
  assert.equal(event.payload.properties.__referrer, "https://trackbaseline.com/checkout");
  assert.equal(options.filter({ type: "track", payload: { name: "click" } }), false);
  window.location = new URL("https://trackbaseline.com/admin");
  assert.equal(options.filter(event), false);
});

test("privacy signals, opted-out storage, previews and private pages fail closed", () => {
  for (const host of ["localhost", "preview.example.com"]) { browser("/", host); startAnalytics(MockOpenPanel); assert.equal(options, undefined); }
  for (const path of ["/dashboard", "/admin", "/auth/login", "/api/data"]) { browser(path); startAnalytics(MockOpenPanel); assert.equal(options, undefined); }
  browser("/"); navigator.globalPrivacyControl = true; startAnalytics(MockOpenPanel); assert.equal(options, undefined);
  browser("/"); navigator.doNotTrack = "1"; startAnalytics(MockOpenPanel); assert.equal(options, undefined);
  browser("/"); window.localStorage.getItem = () => "1"; startAnalytics(MockOpenPanel); assert.equal(options, undefined);
  browser("/"); window.localStorage.getItem = () => { throw new Error("denied"); }; startAnalytics(MockOpenPanel); assert.equal(options, undefined);
});

test("the built SDK sends sanitized public screen views on navigation", async () => {
  const listeners = new Map();
  const sent = [];
  const window = {
    location: new URL("https://trackbaseline.com/checkout/success?session_id=secret#fragment"),
    localStorage: { getItem: () => null },
    addEventListener(name, listener) { listeners.set(name, listener); },
    dispatchEvent(event) { listeners.get(event.type)?.(event); }
  };
  const history = {
    pushState(_state, _title, value) { window.location = new URL(value, window.location.origin); },
    replaceState(_state, _title, value) { window.location = new URL(value, window.location.origin); }
  };
  const context = {
    window, history, URL, Event, setTimeout, clearTimeout, navigator: {},
    document: { referrer: "https://trackbaseline.com/checkout?plan=private", title: "Baseline" },
    sessionStorage: { getItem: () => null },
    fetch: async (_url, request) => { sent.push(JSON.parse(request.body)); return { status: 200, text: async () => "{}" }; },
    console
  };
  vm.runInNewContext(readFileSync(new URL("../public/assets/openpanel.js", import.meta.url), "utf8"), context);
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(sent.length, 1);
  assert.equal(sent[0].payload.properties.__path, "https://trackbaseline.com/checkout/success");
  assert.equal(sent[0].payload.properties.__referrer, "https://trackbaseline.com/checkout");
  history.pushState({}, "", "/docs/mcp?secret=2#example");
  await new Promise((resolve) => setTimeout(resolve, 100));
  assert.equal(sent.length, 2);
  assert.equal(sent[1].payload.properties.__path, "https://trackbaseline.com/docs/mcp");
  history.pushState({}, "", "/admin?secret=3");
  await new Promise((resolve) => setTimeout(resolve, 100));
  assert.equal(sent.length, 2);
});
