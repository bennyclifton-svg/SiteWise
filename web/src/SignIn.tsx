// Sign in, create an account, or reset a forgotten password.
// One panel, four modes; the reset link lands here with its token in the
// fragment, which is read once and removed from the address bar.

import { useState, type FormEvent } from "react";
import { Pending } from "./Activity";
import { api, ApiError } from "./api";
import { Wordmark } from "./Wordmark";

type Mode = "signin" | "signup" | "forgot" | "sent" | "reset";

function takeResetToken(): string {
  const m = location.hash.match(/reset=([0-9a-fA-F]+)/);
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
};

export function SignIn({ initialError, onSignedIn }: { initialError: string; onSignedIn: () => void }) {
  const [resetToken] = useState(takeResetToken);
  const [mode, setMode] = useState<Mode>(resetToken ? "reset" : "signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(resetToken ? "" : initialError);
  const [busy, setBusy] = useState(false);

  function go(next: Mode) {
    setMode(next);
    setError("");
    setPassword("");
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    const address = email.trim();
    if (mode !== "reset" && !address.includes("@")) {
      setError("Enter your email address.");
      return;
    }
    if ((mode === "signup" || mode === "reset") && password.length < 8) {
      setError("Choose a password of at least 8 characters.");
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
      if (mode === "signup") await api.signUp(address, password);
      if (mode === "reset") await api.resetPassword(resetToken, password);
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

  const label = { signin: "Sign in", signup: "Create account", forgot: "Send reset link", reset: "Save password", sent: "" }[mode];
  const pending = { signin: "Signing in…", signup: "Creating account…", forgot: "Sending…", reset: "Saving…", sent: "" }[mode];

  return (
    <main className="entry">
      <div className="entry-panel">
        <Wordmark height={32} />
        <span className="sr-only">SiteWise</span>
        <h1>{titles[mode]}</h1>
        {mode === "signup" && <p>Free to use. Your projects are private to your account.</p>}
        {mode === "forgot" && <p>Enter the email you signed up with. We'll send a link to choose a new password.</p>}
        {mode === "sent" ? (
          <>
            <p>If an account uses {email.trim()}, a reset link is on its way. It works once, within one hour.</p>
            <button type="button" className="btn" onClick={() => go("signin")}>Back to sign in</button>
          </>
        ) : (
          <form className="stack-form" onSubmit={submit} noValidate>
            {mode !== "reset" && (
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
                  autoFocus={mode === "reset"}
                  maxLength={72}
                  aria-invalid={error ? "true" : undefined}
                  aria-describedby={mode === "signin" ? (error ? "auth-error" : undefined) : "password-hint"}
                />
                {mode !== "signin" && <span id="password-hint" className="muted">At least 8 characters.</span>}
              </>
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
          {(mode === "forgot" || mode === "reset") && (
            <button type="button" className="link-button" onClick={() => go("signin")}>Back to sign in</button>
          )}
        </div>
      </div>
    </main>
  );
}
