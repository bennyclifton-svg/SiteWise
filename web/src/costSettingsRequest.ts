// The account menu asks for Cost settings; the Cost Plan opens them. The flag
// covers the first open, when the Cost Plan mounts after the request is made.

const EVENT = "sitewise:cost-settings";
let pending = false;

export function requestCostSettings() {
  pending = true;
  window.dispatchEvent(new Event(EVENT));
}

/** True once per request. */
export function takeCostSettingsRequest(): boolean {
  const was = pending;
  pending = false;
  return was;
}

export function onCostSettingsRequest(listener: () => void): () => void {
  window.addEventListener(EVENT, listener);
  return () => window.removeEventListener(EVENT, listener);
}
