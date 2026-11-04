import { useState } from "react";
import { api, ApiError } from "./api";

export function SignOutButton() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function signOut() {
    setBusy(true);
    setError("");
    try {
      await api.signOut();
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 401)) {
        setError("Could not sign out. Try again.");
        setBusy(false);
        return;
      }
    }
    // Unmount the project and its event stream without reloading dev-login.
    window.dispatchEvent(new Event("sitewise:signed-out"));
  }
  return <div className="sign-out-control">
    <button type="button" className="btn" disabled={busy} onClick={() => void signOut()}>{busy ? "Signing out…" : "Sign out"}</button>
    {error && <span role="alert" className="error-text">{error}</span>}
  </div>;
}
