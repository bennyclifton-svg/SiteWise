// Keep navigation in the URL so a reload restores it without a storage or API read.
export function readLocationChoice<T extends string>(key: string, choices: readonly T[], fallback: T): T {
  const value = new URLSearchParams(location.search).get(key);
  return choices.find(choice => choice === value) ?? fallback;
}

export function writeLocationChoice(key: string, value: string) {
  const url = new URL(location.href);
  url.searchParams.set(key, value);
  history.replaceState(history.state, "", url);
}
