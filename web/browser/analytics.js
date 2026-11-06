const productionHosts = new Set(["trackbaseline.com", "www.trackbaseline.com"]);
const privatePath = /^\/(?:admin|dashboard|auth|login|sign-in|api)(?:\/|$)/i;

function permitted() {
  if (!productionHosts.has(window.location.hostname) || privatePath.test(window.location.pathname)) return false;
  if (navigator.globalPrivacyControl === true || [navigator.doNotTrack, window.doNotTrack, navigator.msDoNotTrack].some((value) => value === "1" || value === "yes")) return false;
  try {
    return window.localStorage.getItem("asc-analytics-off") !== "1";
  } catch {
    return false;
  }
}

function cleanUrl(value) {
  if (!value) return "";
  try {
    const url = new URL(value, window.location.origin);
    return url.origin + url.pathname;
  } catch {
    return "";
  }
}

export function startAnalytics(OpenPanel) {
  if (!permitted()) return;
  new OpenPanel({
    clientId: "f6d278c9-847b-4420-8867-a22ce7b7e3fd",
    trackScreenViews: true,
    trackOutgoingLinks: false,
    trackAttributes: false,
    filter: (event) => {
      if (!permitted() || event.type !== "track" || event.payload.name !== "screen_view") return false;
      const properties = event.payload.properties;
      if (!properties || typeof properties.__path !== "string") return false;
      const path = cleanUrl(properties.__path);
      if (!path || privatePath.test(new URL(path).pathname)) return false;
      properties.__path = path;
      if (typeof properties.__referrer === "string") properties.__referrer = cleanUrl(properties.__referrer);
      return true;
    }
  });
}
