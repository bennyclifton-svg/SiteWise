// Account: plan, usage, upgrade, billing and team. Payment happens on
// Stripe's own pages; this page only links to them (design 2026-11-07-billing).

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, ApiError, type Account as AccountData, type Plan, type Team } from "./api";
import { LegalLinks } from "./LegalLinks";

const aud = (cents: number) =>
  (cents / 100).toLocaleString("en-AU", { style: "currency", currency: "AUD", maximumFractionDigits: 0 });

function size(bytes: number): string {
  const gb = 1 << 30;
  if (bytes >= gb) return `${(bytes / gb).toFixed(bytes % gb === 0 ? 0 : 1)} GB`;
  return `${Math.max(0, Math.round(bytes / (1 << 20)))} MB`;
}

const date = (iso: string) => new Date(iso).toLocaleDateString("en-AU", { day: "numeric", month: "long", year: "numeric" });

function message(err: unknown): string {
  if (err instanceof ApiError && err.message) return err.message;
  return "Something went wrong. Try again.";
}

export function Account({ onSignedOut }: { onSignedOut: () => void }) {
  const [account, setAccount] = useState<AccountData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [returned] = useState(() => new URLSearchParams(location.search).get("checkout") === "done");

  const load = useCallback(() => {
    setLoadError("");
    api.account().then(setAccount, (e: unknown) => {
      if (e instanceof ApiError && e.status === 401) onSignedOut();
      else setLoadError("Could not load your account. Check your connection and retry.");
    });
  }, [onSignedOut]);

  useEffect(load, [load]);

  // After Checkout, Stripe's webhook usually lands within seconds.
  useEffect(() => {
    if (!returned) return;
    history.replaceState(null, "", "/account");
    let tries = 0;
    const id = window.setInterval(() => {
      tries++;
      api.account().then(setAccount, () => undefined);
      if (tries >= 6) window.clearInterval(id);
    }, 2500);
    return () => window.clearInterval(id);
  }, [returned]);

  if (loadError) {
    return (
      <div className="account">
        <p className="error-text" role="alert">{loadError}</p>
        <button className="btn" onClick={load}>Retry</button>
      </div>
    );
  }
  if (!account) return <div className="account" aria-busy="true" />;

  const owner = account.role === "owner";
  return (
    <div className="account">
      <h1>Account</h1>
      <p className="muted">{account.email}{owner ? " · Account owner" : " · Team member"}</p>
      {returned && <p className="account-note" role="status">Thank you. Your plan updates here as soon as Stripe confirms the payment.</p>}
      {!account.email_verified && <VerifyBanner />}
      <PlanSummary account={account} />
      <Usage account={account} />
      {account.billing_enabled && owner && <Plans account={account} />}
      {!account.billing_enabled && <p className="muted">Billing is not set up on this server, so plan limits do not apply.</p>}
      <TeamSection account={account} />
      <footer className="account-footer">
        <LegalLinks />
      </footer>
    </div>
  );
}

function VerifyBanner() {
  const [state, setState] = useState<"idle" | "busy" | "sent">("idle");
  const [error, setError] = useState("");
  async function resend() {
    setState("busy");
    setError("");
    try {
      await api.resendVerify();
      setState("sent");
    } catch (e) {
      setError(message(e));
      setState("idle");
    }
  }
  return (
    <section className="account-banner" role="status">
      <p>Confirm your email address to upload documents. We sent a link when you signed up.</p>
      {state === "sent" ? (
        <p className="muted">A new link is on its way. It works once, within 24 hours.</p>
      ) : (
        <button className="btn" disabled={state === "busy"} onClick={() => void resend()}>
          {state === "busy" ? "Sending…" : "Send a new link"}
        </button>
      )}
      {error && <p className="error-text" role="alert">{error}</p>}
    </section>
  );
}

function PlanSummary({ account }: { account: AccountData }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const owner = account.role === "owner";
  async function manage() {
    setBusy(true);
    setError("");
    try {
      const { url } = await api.billingPortal();
      location.assign(url);
    } catch (e) {
      setError(message(e));
      setBusy(false);
    }
  }
  const lapsed = account.subscribed_plan !== "free" && account.plan.id === "free" && account.status !== "";
  let status = "";
  if (account.status === "past_due") status = "Your last payment did not go through. Update your card in Manage billing to keep your plan.";
  else if (account.cancel_at_period_end && account.paid_until) status = `Cancelled. Your plan stays active until ${date(account.paid_until)}, then moves to Free.`;
  else if (account.paid_until && account.plan.id !== "free") status = `Renews on ${date(account.paid_until)}, billed ${account.interval === "year" ? "yearly" : "monthly"}.`;
  else if (lapsed) status = "Your subscription has ended. Your projects and documents are all still here; choose a plan to add more.";
  return (
    <section className="account-section" aria-labelledby="plan-heading">
      <h2 id="plan-heading">Plan</h2>
      {/* With billing off nothing is enforced, so naming a plan would mislead. */}
      <p className="account-plan">{account.billing_enabled ? account.plan.name : "No limits"}</p>
      {status && <p className="muted">{status}</p>}
      {owner && account.billing_enabled && account.has_customer && (
        <button className="btn" disabled={busy} onClick={() => void manage()}>
          {busy ? "Opening…" : "Manage billing"}
        </button>
      )}
      {owner && account.has_customer && <p className="muted small">Change plan, update your card, download tax invoices or cancel on Stripe's secure page.</p>}
      {error && <p className="error-text" role="alert">{error}</p>}
    </section>
  );
}

function Meter({ label, used, cap, format }: { label: string; used: number; cap: number; format: (n: number) => string }) {
  const unlimited = cap === 0;
  const fraction = unlimited ? 0 : Math.min(1, used / cap);
  return (
    <div className="account-meter">
      <div className="account-meter-row">
        <span>{label}</span>
        <span className="mono">{format(used)}{unlimited ? "" : ` of ${format(cap)}`}</span>
      </div>
      {!unlimited && <meter min={0} max={1} value={fraction} high={0.8} optimum={0} aria-label={label} />}
    </div>
  );
}

function Usage({ account }: { account: AccountData }) {
  const { usage } = account;
  // A cap of 0 shows the count alone; with billing off no cap applies.
  const plan = account.billing_enabled ? account.plan : { documents_per_month: 0, storage_bytes: 0, projects: 0, users: 0 };
  const n = (v: number) => v.toLocaleString("en-AU");
  return (
    <section className="account-section" aria-labelledby="usage-heading">
      <h2 id="usage-heading">This month</h2>
      <Meter label="Documents read" used={usage.documents_this_month} cap={plan.documents_per_month} format={n} />
      <Meter label="Storage" used={usage.storage_bytes} cap={plan.storage_bytes} format={size} />
      <Meter label="Active projects" used={usage.projects} cap={plan.projects} format={n} />
      <Meter label="Users" used={usage.seats} cap={plan.users} format={n} />
      <p className="muted small">The document count resets on the first of each month (Sydney time). Dropping the same file again does not count.</p>
    </section>
  );
}

function Plans({ account }: { account: AccountData }) {
  const [period, setPeriod] = useState<"month" | "year">("month");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const subscribed = account.plan.id !== "free" && account.has_customer && account.status !== "";
  async function choose(plan: Plan) {
    if (plan.id === "free") return;
    setBusy(plan.id);
    setError("");
    try {
      const { url } = await api.checkout(plan.id, period);
      location.assign(url);
    } catch (e) {
      setError(message(e));
      setBusy("");
    }
  }
  return (
    <section className="account-section" aria-labelledby="plans-heading">
      <div className="account-plans-head">
        <h2 id="plans-heading">Plans</h2>
        <div className="account-toggle" role="group" aria-label="Billing period">
          <button className={period === "month" ? "btn btn-primary" : "btn"} aria-pressed={period === "month"} onClick={() => setPeriod("month")}>Monthly</button>
          <button className={period === "year" ? "btn btn-primary" : "btn"} aria-pressed={period === "year"} onClick={() => setPeriod("year")}>Yearly, 2 months free</button>
        </div>
      </div>
      <div className="account-plans">
        {account.plans.map((p) => {
          const current = p.id === account.plan.id;
          const price = period === "year" ? p.annual_cents : p.monthly_cents;
          return (
            <article key={p.id} className={current ? "account-plan-card current" : "account-plan-card"}>
              <h3>{p.name}</h3>
              <p className="account-price">
                {p.monthly_cents === 0 ? "Free" : <>{aud(price)}<span className="muted"> / {period === "year" ? "year" : "month"} + GST</span></>}
              </p>
              <ul>
                <li>{p.projects === 0 ? "Unlimited projects" : `${p.projects} active project${p.projects === 1 ? "" : "s"}`}</li>
                <li>{p.documents_per_month.toLocaleString("en-AU")} documents read a month</li>
                <li>{size(p.storage_bytes)} storage</li>
                <li>{p.users === 1 ? "1 user" : `Up to ${p.users} users`}</li>
              </ul>
              {current ? (
                <span className="muted">Current plan</span>
              ) : p.id === "free" ? (
                subscribed ? <span className="muted">Cancel in Manage billing</span> : null
              ) : subscribed ? (
                <span className="muted">Change in Manage billing</span>
              ) : (
                <button className="btn btn-primary" disabled={busy !== ""} onClick={() => void choose(p)}>
                  {busy === p.id ? "Opening Stripe…" : `Choose ${p.name}`}
                </button>
              )}
            </article>
          );
        })}
      </div>
      {error && <p className="error-text" role="alert">{error}</p>}
      <p className="muted small">
        Prices are in Australian dollars and exclude GST, which is added at checkout. Payment is handled by Stripe; SiteWise never sees your card number.
        See <a href="/legal/refunds.html">cancellations and refunds</a>.
      </p>
    </section>
  );
}

function TeamSection({ account }: { account: AccountData }) {
  const [team, setTeam] = useState<Team | null>(null);
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const owner = account.role === "owner";
  const load = useCallback(() => {
    api.team().then(setTeam, (e: unknown) => setError(message(e)));
  }, []);
  useEffect(load, [load]);
  if (!team) return null;
  if (team.seats <= 1 && team.members.length <= 1 && team.invites.length === 0) {
    return (
      <section className="account-section" aria-labelledby="team-heading">
        <h2 id="team-heading">Team</h2>
        <p className="muted">The Team plan lets up to five colleagues work in the same projects.</p>
      </section>
    );
  }
  async function invite(e: FormEvent) {
    e.preventDefault();
    if (!email.includes("@")) {
      setError("Enter your colleague's email address.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api.inviteMember(email.trim());
      setEmail("");
      load();
    } catch (err) {
      setError(message(err));
    }
    setBusy(false);
  }
  async function act(run: () => Promise<void>) {
    setError("");
    try {
      await run();
      load();
    } catch (err) {
      setError(message(err));
    }
  }
  return (
    <section className="account-section" aria-labelledby="team-heading">
      <h2 id="team-heading">Team</h2>
      {account.billing_enabled && <p className="muted">{team.members.length + team.invites.length} of {team.seats} seats used, counting open invites.</p>}
      <ul className="account-team">
        {team.members.map((m) => (
          <li key={m.user_id}>
            <span>{m.email}</span>
            <span className="muted">{m.role === "owner" ? "Owner" : "Member"}</span>
            {owner && m.role !== "owner" && (
              <button className="link-button" onClick={() => {
                if (confirm(`Remove ${m.email}? They will be signed out and lose access. Their work stays in your projects.`)) void act(() => api.removeMember(m.user_id));
              }}>Remove</button>
            )}
          </li>
        ))}
        {team.invites.map((i) => (
          <li key={i.id}>
            <span>{i.email}</span>
            <span className="muted">Invited, link expires {date(i.expires_at)}</span>
            {owner && <button className="link-button" onClick={() => void act(() => api.revokeInvite(i.id))}>Cancel invite</button>}
          </li>
        ))}
      </ul>
      {owner && (
        <form className="account-invite" onSubmit={invite} noValidate>
          <label htmlFor="invite-email">Invite a colleague</label>
          <input id="invite-email" className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} spellCheck={false} placeholder="name@company.com.au" />
          <button className="btn btn-primary" type="submit" disabled={busy}>{busy ? "Sending…" : "Send invite"}</button>
        </form>
      )}
      {error && <p className="error-text" role="alert">{error}</p>}
    </section>
  );
}
