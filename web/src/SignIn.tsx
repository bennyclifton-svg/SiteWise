// Sign in, create an account, reset a forgotten password, or accept a team
// invite. One panel; reset and invite links land here with their token in
// the fragment, which is read once and removed from the address bar.

import { useState, type FormEvent } from "react";
import { Pending } from "./Activity";
import { api, ApiError } from "./api";
import { LegalLinks } from "./LegalLinks";
import { Wordmark } from "./Wordmark";

type Mode = "signin" | "signup" | "forgot" | "sent" | "reset" | "invite";

function takeToken(key: "reset" | "invite"): string {
  const m = location.hash.match(new RegExp(`${key}=([0-9a-fA-F]+)`));
  if (!m) return "";
  history.replaceState(null, "", location.pathname + location.search);
  return m[1];
}

const titles: Record<Mode, string> = {
  signin: "Sign in",
  signup: "Create your account",
  forgot: "Reset your password",
  sent: "Check your email",
  reset: "Choose a new password",
  invite: "Join your team",
};

export function SignIn({ initialError, onSignedIn }: { initialError: string; onSignedIn: () => void }) {
  const [resetToken] = useState(() => takeToken("reset"));
  const [inviteToken] = useState(() => takeToken("invite"));
  const [mode, setMode] = useState<Mode>(resetToken ? "reset" : inviteToken ? "invite" : "signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [terms, setTerms] = useState(false);
  const [error, setError] = useState(resetToken || inviteToken ? "" : initialError);
  const [busy, setBusy] = useState(false);
  const needsTerms = mode === "signup" || mode === "invite";
  const needsEmail = mode !== "reset" && mode !== "invite";

  function go(next: Mode) {
    setMode(next);
    setError("");
    setPassword("");
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    const address = email.trim();
    if (needsEmail && !address.includes("@")) {
      setError("Enter your email address.");
      return;
    }
    if ((mode === "signup" || mode === "reset" || mode === "invite") && password.length < 8) {
      setError("Choose a password of at least 8 characters.");
      return;
    }
    if (needsTerms && !terms) {
      setError("Tick the box to accept the Terms of Service and Privacy Policy.");
      return;
    }
    if (mode === "signin" && !password) {
      setError("Enter your password.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (mode === "signin") await api.signInWithPassword(address, password);
      if (mode === "signup") await api.signUp(address, password, terms);
      if (mode === "reset") await api.resetPassword(resetToken, password);
      if (mode === "invite") await api.acceptInvite(inviteToken, password, terms);
      if (mode === "forgot") {
        await api.forgotPassword(address);
        setBusy(false);
        setMode("sent");
        return;
      }
      onSignedIn();
    } catch (err) {
      setBusy(false);
      setError(err instanceof ApiError && err.status === 429
        ? "Too many attempts. Wait 15 minutes and try again."
        : err instanceof Error && err.message ? err.message : "Something went wrong. Try again.");
    }
  }

  const label = { signin: "Sign in", signup: "Create account", forgot: "Send reset link", reset: "Save password", sent: "", invite: "Join team" }[mode];
  const pending = { signin: "Signing in…", signup: "Creating account…", forgot: "Sending…", reset: "Saving…", sent: "", invite: "Joining…" }[mode];

  return (
    <main className="entry">
      <div className="entry-panel">
        <Wordmark height={32} />
        <span className="sr-only">SiteWise</span>
        <h1>{titles[mode]}</h1>
        {mode === "signup" && <p>Start on the Free plan. Your projects are private to your account. We'll email you a link to confirm your address.</p>}
        {mode === "invite" && <p>Choose a password to join the team that invited you. You'll work in their projects.</p>}
        {mode === "forgot" && <p>Enter the email you signed up with. We'll send a link to choose a new password.</p>}
        {mode === "sent" ? (
          <>
            <p>If an account uses {email.trim()}, a reset link is on its way. It works once, within one hour.</p>
            <button type="button" className="btn" onClick={() => go("signin")}>Back to sign in</button>
          </>
        ) : (
          <form className="stack-form" onSubmit={submit} noValidate>
            {needsEmail && (
              <>
                <label htmlFor="email">Email</label>
                <input
                  id="email"
                  className="input"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete={mode === "signup" ? "email" : "username"}
                  spellCheck={false}
                  autoFocus
                  aria-invalid={error ? "true" : undefined}
                  aria-describedby={error ? "auth-error" : undefined}
                />
              </>
            )}
            {mode !== "forgot" && (
              <>
                <label htmlFor="password">{mode === "signin" ? "Password" : "New password"}</label>
                <input
                  id="password"
                  className="input"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete={mode === "signin" ? "current-password" : "new-password"}
                  autoFocus={mode === "reset" || mode === "invite"}
                  maxLength={72}
                  aria-invalid={error ? "true" : undefined}
                  aria-describedby={mode === "signin" ? (error ? "auth-error" : undefined) : "password-hint"}
                />
                {mode !== "signin" && <span id="password-hint" className="muted">At least 8 characters.</span>}
              </>
            )}
            {needsTerms && (
              <label className="entry-terms">
                <input type="checkbox" checked={terms} onChange={(e) => setTerms(e.target.checked)} />
                <span>
                  I agree to the <a href="/legal/terms.html" target="_blank" rel="noopener">Terms of Service</a> and{" "}
                  <a href="/legal/privacy.html" target="_blank" rel="noopener">Privacy Policy</a>.
                </span>
              </label>
            )}
            {error && (
              <span id="auth-error" className="error-text" role="alert">
                {error}
              </span>
            )}
            <button type="submit" className="btn btn-primary" disabled={busy}>
              {busy ? <Pending delayed>{pending}</Pending> : label}
            </button>
          </form>
        )}
        <div className="entry-switch">
          {mode === "signin" && (
            <>
              <button type="button" className="link-button" onClick={() => go("signup")}>Create an account</button>
              <button type="button" className="link-button" onClick={() => go("forgot")}>Forgot password?</button>
            </>
          )}
          {mode === "signup" && (
            <span>Already have an account? <button type="button" className="link-button" onClick={() => go("signin")}>Sign in</button></span>
          )}
          {(mode === "forgot" || mode === "reset" || mode === "invite") && (
            <button type="button" className="link-button" onClick={() => go("signin")}>Back to sign in</button>
          )}
        </div>
        <div className="entry-legal">
          <LegalLinks />
        </div>
      </div>
    </main>
  );
}
