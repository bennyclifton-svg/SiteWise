// The public front door: one headline, then the sample project being built
// in front of the visitor (Stage.tsx): the profile filling in as a real Update
// does, then the cost plan, the program and the tenders. Shown to a
// signed-out visitor at "/", and to anyone at "/welcome". It reads nothing
// from the server. The screens are a sample project (T05 from
// testing/synthetic).

import { useEffect } from "react";
import { GroundButton } from "../GroundButton";
import { Wordmark } from "../Wordmark";
import { Stage } from "./Stage";
import "./landing.css";

export default function Landing({ onStart, onSignIn }: { onStart: () => void; onSignIn: () => void }) {
  useEffect(() => {
    const title = document.title;
    document.title = "SiteWise: files your documents in about a second";
    return () => { document.title = title; };
  }, []);

  return (
    <div className="landing">
      <a className="sr-only skip-link" href="#main">Skip to content</a>
      <header className="landing-top landing-wrap">
        <Wordmark height={30} />
        <span className="sr-only">SiteWise</span>
        <nav aria-label="Main">
          <a className="landing-wide" href="#stage">What it builds</a>
          <a className="landing-wide" href="#pricing">Pricing</a>
          <button type="button" className="landing-link" onClick={onSignIn}>Sign in</button>
          <GroundButton primary small onClick={onStart}>Start free</GroundButton>
        </nav>
      </header>

      <main id="main" className="landing-wrap">
        <div className="landing-sheet">
          <section className="landing-open perspex" aria-labelledby="landing-title">
            <h1 id="landing-title">Files your documents in about a second.</h1>
            <div className="landing-open-side">
              <p className="landing-lede">Drop in the folder you already have. SiteWise files every drawing, report and quote, then builds the job from them.</p>
              <div className="landing-act">
                <GroundButton primary className="landing-cta" onClick={onStart}>Start a project free</GroundButton>
                <span>Your first project is free.</span>
              </div>
            </div>
          </section>

          <Stage />

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
            <GroundButton primary className="landing-cta landing-price-cta" onClick={onStart}>Start a project free</GroundButton>
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
