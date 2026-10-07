// Windows taskkill can hang in restricted runners. Close the server through
// its test-only control endpoint before Playwright kills the command wrapper.
export default async function stopServer() {
  const response = await fetch("http://127.0.0.1:4173/__e2e/shutdown", {
    method: "POST",
    signal: AbortSignal.timeout(5_000),
  });
  if (!response.ok) throw new Error(`Test server shutdown failed: ${response.status}`);
}
