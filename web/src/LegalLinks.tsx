// Links to the public legal pages, which are plain HTML in web/public/legal
// so they load without signing in or running scripts.
export function LegalLinks() {
  return (
    <nav className="legal-links" aria-label="Legal">
      <a href="/legal/pricing.html">Pricing</a>
      <a href="/legal/terms.html">Terms of Service</a>
      <a href="/legal/privacy.html">Privacy Policy</a>
      <a href="/legal/refunds.html">Cancellations and refunds</a>
    </nav>
  );
}
