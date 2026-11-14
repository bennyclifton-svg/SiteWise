// The public front door: what SiteWise builds from a project's folder, in
// five plain notes, each beside the real screen. Shown to a signed-out
// visitor at "/", and to anyone at "/welcome". It reads nothing from the
// server. The screens are a sample project (T05 from testing/synthetic).

import { useEffect, useRef, type ReactNode } from "react";
import { GroundButton } from "../GroundButton";
import { Wordmark } from "../Wordmark";
import { mountShot, type Shot } from "./shotMotion";
import shots from "./shots.json";
import registerPng from "./register.webp";
import profilePng from "./profile.webp";
import costPng from "./cost.webp";
import programPng from "./program.webp";
import procurementPng from "./procurement.webp";
import tenderPng from "./tender.webp";
import "./landing.css";

const built: { id: string; name: string; from: string }[] = [
  { id: "files", name: "Document register", from: "About a second a file" },
  { id: "profile", name: "Project profile", from: "Read from the set" },
  { id: "cost", name: "Cost plan", from: "Lines set up for you" },
  { id: "program", name: "Program", from: "Every date worked out" },
  { id: "tenders", name: "Tender evaluation", from: "Quotes side by side" },
];

export default function Landing({ onStart, onSignIn }: { onStart: () => void; onSignIn: () => void }) {
  useEffect(() => {
    const title = document.title;
    document.title = "SiteWise: bring a project";
    return () => { document.title = title; };
  }, []);

  return (
    <div className="landing">
      <a className="sr-only skip-link" href="#main">Skip to content</a>
      <header className="landing-top landing-wrap">
        <Wordmark height={30} />
        <span className="sr-only">SiteWise</span>
        <nav aria-label="Main">
          <a className="landing-wide" href="#files">What it does</a>
          <a className="landing-wide" href="#pricing">Pricing</a>
          <button type="button" className="landing-link" onClick={onSignIn}>Sign in</button>
          <GroundButton primary small onClick={onStart}>Start free</GroundButton>
        </nav>
      </header>

      <main id="main" className="landing-wrap">
        <div className="landing-sheet">
          <section className="landing-open" aria-labelledby="landing-title">
            <div>
              <h1 id="landing-title">Bring a project.</h1>
              <p className="landing-lede">Make one, drop in the folder you already have, and SiteWise builds the job from it: your documents filed, the project profile, the cost plan, the program and the tenders.</p>
              <div className="landing-act">
                <GroundButton primary className="landing-cta" onClick={onStart}>Start a project free</GroundButton>
                <span>Your first project is free.</span>
              </div>
            </div>
            <nav className="landing-built" aria-label="What SiteWise builds">
              <div className="landing-built-head" aria-hidden="true"><span>No.</span><span>SiteWise builds</span><span>From your folder</span></div>
              {built.map((row, i) => (
                <a key={row.id} href={`#${row.id}`}>
                  <span className="landing-built-no">{i + 1}</span>
                  <b>{row.name}</b>
                  <span className="landing-built-from">{row.from}</span>
                </a>
              ))}
            </nav>
          </section>

          <div className="landing-notes">
            <Note id="files" n={1} lead title="Files your documents in about a second.">
              <p>Drag in the drawings, reports, quotes and invoices. SiteWise reads the number, revision, title and discipline off each one and puts it in the register. Drop a new revision and the old one is marked superseded.</p>
              <figure className="landing-shot landing-shot-narrow">
                <img src={registerPng} width={1024} height={1264} alt="The SiteWise document register: 18 documents listed with drawing number, title, revision, date and discipline, the superseded revision of A-201 greyed." />
              </figure>
            </Note>

            <Note id="profile" n={2} flip title="Builds your project profile.">
              <p>It reads the whole set and writes the job up on one page: what's being built, the site and planning, the client, the budget, and the scope of works, system by system.</p>
              <Animated shot={shots.profile as Shot} src={profilePng} alt="Project Profile being filled in from the documents: a one-sentence summary of a house extension in Concord, then site and planning, the building and delivery details, and the scope of works by building system." />
            </Note>

            <Note id="cost" n={3} title="Builds your cost plan.">
              <p>It works out the lines your job needs: authority fees, each consultant, construction and contingency. Award a consultant and their fee drops in by stage. Drop in an invoice and claimed to date updates.</p>
              <Animated shot={shots.cost as Shot} src={costPng} alt="Cost Plan being set up: fees and charges and consultant lines, then the architect's four stage fees and the BASIX fees arriving in the approved contract column." />
            </Note>

            <Note id="program" n={4} flip title="Builds your program.">
              <p>Stages and activities from approvals through to completion, laid out as a Gantt chart. Drag a bar or change a duration and every date after it moves. Export to Excel or print it A3.</p>
              <Animated shot={shots.program as Shot} src={programPng} alt="Program being drawn: a grid of activities with days, start, finish and predecessors, and each bar of the Gantt chart from November 2026 to mid 2027." />
            </Note>

            <Note id="tenders" n={5} title="Sets up and evaluates your tenders.">
              <p>It works out which consultants and trades the job needs and writes the RFP for each. When the quotes come back, drop them on the row. SiteWise lines the prices up stage by stage so you can compare and award.</p>
              <div className="landing-pair">
                <figure className="landing-shot">
                  <img src={procurementPng} width={2000} height={1000} loading="lazy" alt="Procurement: one row per consultant, from architect to town planner, with the firms that quoted, their fees, and the awarded firms ticked." />
                </figure>
                <figure className="landing-shot">
                  <img src={tenderPng} width={2000} height={1039} loading="lazy" alt="Tender evaluation for the architect: the firm's price against each stage of the RFP, totalling $44,000 excluding GST." />
                </figure>
              </div>
            </Note>
          </div>

          <section className="landing-price" id="pricing" aria-labelledby="landing-price-title">
            <h2 id="landing-price-title">Your first project is free.</h2>
            <div className="landing-price-table">
              <table>
                <thead>
                  <tr><th>Plan</th><th className="num">Projects</th><th className="num">Documents a month</th><th className="num landing-wide">Users</th><th className="num">Price, ex GST</th></tr>
                </thead>
                <tbody>
                  <tr><th>Free</th><td className="num">1</td><td className="num">50</td><td className="num landing-wide">1</td><td className="num amt">$0</td></tr>
                  <tr><th>Starter</th><td className="num">5</td><td className="num">500</td><td className="num landing-wide">1</td><td className="num amt">$50 a month</td></tr>
                  <tr><th>Team</th><td className="num">Unlimited</td><td className="num">3,000</td><td className="num landing-wide">Up to 5</td><td className="num amt">$200 a month</td></tr>
                </tbody>
              </table>
            </div>
            <p className="landing-fine">Australian dollars. Pay yearly and get two months free. <a href="/legal/pricing.html">Full pricing</a></p>
          </section>

          <section className="landing-close" aria-labelledby="landing-close-title">
            <h2 id="landing-close-title">Got a folder? Start there.</h2>
            <p>Make a project, drop it in, and have the register, profile, cost plan and program in front of you today.</p>
            <GroundButton primary className="landing-cta" onClick={onStart}>Start a project free</GroundButton>
          </section>
        </div>
      </main>

      <footer className="landing-foot landing-wrap">
        <span>SiteWise. For Australian project managers, designers and builders. Screens show a sample project.</span>
        <nav aria-label="Legal">
          <a href="/legal/pricing.html">Pricing</a>
          <a href="/legal/terms.html">Terms</a>
          <a href="/legal/privacy.html">Privacy</a>
        </nav>
      </footer>
    </div>
  );
}

function Note({ id, n, title, lead, flip, children }: { id: string; n: number; title: string; lead?: boolean; flip?: boolean; children: [ReactNode, ReactNode] }) {
  const [copy, picture] = children;
  return (
    <section className={`landing-note${lead ? " is-lead" : ""}${flip ? " is-flipped" : ""}`} id={id} aria-labelledby={`landing-${id}`}>
      <div className="landing-note-copy">
        <span className="landing-no" aria-hidden="true">{n}</span>
        <h2 id={`landing-${id}`}>{title}</h2>
        {copy}
      </div>
      {picture}
    </section>
  );
}

/** A screen that is filled in when it scrolls into view, with a way to see it again. */
function Animated({ shot, src, alt }: { shot: Shot; src: string; alt: string }) {
  const frame = useRef<HTMLDivElement>(null);
  const again = useRef<HTMLButtonElement>(null);
  const motion = useRef<ReturnType<typeof mountShot>>(null);
  useEffect(() => {
    if (!frame.current) return;
    const mounted = mountShot(frame.current, shot);
    motion.current = mounted;
    // The button is only offered where there is something to play.
    if (again.current) again.current.hidden = !mounted;
    return () => mounted?.destroy();
  }, [shot]);
  return (
    <figure className="landing-shot landing-shot-live">
      <div className="landing-shot-frame" ref={frame}>
        <img src={src} width={shot.width} height={shot.height} loading="lazy" alt={alt} />
      </div>
      <button type="button" className="landing-again" ref={again} hidden onClick={() => motion.current?.replay()}>Play again</button>
    </figure>
  );
}
