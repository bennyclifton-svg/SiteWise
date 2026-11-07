import { useState } from "react";
import { api, ApiError } from "./api";

/** Ends the session. Returns an error message, or "" once the app has been told. */
export async function endSession(): Promise<string> {
  try {
    await api.signOut();
  } catch (err) {
    if (!(err instanceof ApiError && err.status === 401)) return "Could not sign out. Try again.";
  }
  // Unmount the project and its event stream without reloading dev-login.
  window.dispatchEvent(new Event("sitewise:signed-out"));
  return "";
}

export function SignOutButton() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function signOut() {
    setBusy(true);
    setError("");
    const failed = await endSession();
    if (failed) {
      setError(failed);
      setBusy(false);
    }
  }
  return <div className="sign-out-control">
    <a
      className="btn account-link"
      href="/account"
      onClick={(e) => {
        e.preventDefault();
        window.dispatchEvent(new CustomEvent("sitewise:navigate", { detail: "/account" }));
      }}
    >
      Account
    </a>
    <button type="button" className="btn" disabled={busy} onClick={() => void signOut()}>{busy ? "Signing out…" : "Sign out"}</button>
    {error && <span role="alert" className="error-text">{error}</span>}
  </div>;
}
