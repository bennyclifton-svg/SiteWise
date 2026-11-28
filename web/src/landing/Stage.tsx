// The landing page's one moving picture: the sample project being built, one
// view after another. Profile first, as a replay of a real Update (documents
// taken three at a time, values written the moment a passage settles them,
// in a different order each time), then the cost plan, the program and the
// tenders, then round again. Every chapter runs on one clock, so it pauses
// as a whole: off screen, in a hidden tab, or when the visitor presses Pause.
// With reduced motion nothing is covered and nothing advances by itself.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { GroundButton } from "../GroundButton";
import { layCovers, type Covers, type Shot } from "./shotMotion";
import { docsAt, findAt, makeReplay, type Replay } from "./profileReplay";
import { QUOTE_COUNT, procurementShot, tenderShot } from "./tenderShots";
import shots from "./shots.json";
import profilePng from "./profile.webp";
import costPng from "./cost.webp";
import programPng from "./program.webp";
import procurementPng from "./procurement.webp";
import tenderPng from "./tender.webp";

type Id = "profile" | "cost" | "program" | "tenders";
export const CHAPTERS: { id: Id; title: string; text: string; view: string }[] = [
  { id: "profile", view: "Project profile", title: "Builds your project profile.", text: "Press Update and it reads the whole set: the site and planning, the building, the client and the scope of works, system by system. Each value is written as soon as a document settles it." },
  { id: "cost", view: "Cost plan", title: "Builds your cost plan.", text: "Authority fees, each consultant, construction and contingency. Award a consultant and their fee drops in by stage; drop in an invoice and claimed to date follows." },
  { id: "program", view: "Program", title: "Builds your program.", text: "Stages and activities from approvals to completion, with every date worked out. Drag a bar and every date after it moves." },
  { id: "tenders", view: "Procurement", title: "Sets up and evaluates your tenders.", text: "It writes the RFP for each consultant and trade. Drop the quotes on the row and the prices line up stage by stage, ready to award." },
];

const HOLD = 2600; // the finished screen stays a moment before the next one
const PRESS = 450; // when Update is pressed in the profile replay
const TENDER_IN = 4200; // when the tender comparison comes up over the table

function stillMotion() {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
}
function shuffled(n: number) {
  return Array.from({ length: n }, (_, i) => i).sort(() => Math.random() - 0.5);
}

export function Stage() {
  const still = useMemo(stillMotion, []);
  const [active, setActive] = useState(0);
  const [round, setRound] = useState(0); // a new random run each time round
  const [t, setT] = useState(still ? Infinity : 0);
  const [paused, setPaused] = useState(false);
  const [shown, setShown] = useState(true);
  const [length, setLength] = useState(Infinity);
  const [replay, setReplay] = useState<Replay>(() => makeReplay(shots.profile as Shot));

  const stage = useRef<HTMLDivElement>(null);
  const frames = useRef<Record<string, HTMLDivElement | null>>({});
  const tenderCard = useRef<HTMLElement>(null);
  const covers = useRef<Record<string, Covers>>({});
  const clock = useRef({ start: 0, pausedAt: 0, anims: [] as Animation[], length: Infinity });
  const first = useRef(true); // the first run is the one already made

  // Covers are laid once; each chapter plays them again from empty.
  useEffect(() => {
    if (still || typeof HTMLElement.prototype.animate !== "function") return;
    const laid: Record<string, Covers> = {};
    const lay = (key: string, shot: Shot) => {
      const frame = frames.current[key];
      if (frame) laid[key] = layCovers(frame, shot);
    };
    lay("profile", replay.shot);
    lay("cost", shots.cost as Shot);
    lay("program", shots.program as Shot);
    lay("procurement", procurementShot(shuffled(QUOTE_COUNT)));
    lay("tender", tenderShot());
    covers.current = laid;
    return () => Object.values(laid).forEach(c => c.destroy());
    // The profile's covers are the same boxes every round; only their times change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [still]);

  // Starts the active chapter from its empty screen.
  useEffect(() => {
    if (still) return;
    const c = covers.current;
    const id = CHAPTERS[active].id;
    const anims: Animation[] = [];
    let end = 0;
    if (id === "profile") {
      const r = first.current ? replay : makeReplay(shots.profile as Shot);
      first.current = false;
      if (r !== replay) setReplay(r);
      if (c.profile) anims.push(...c.profile.play(r.times));
      end = r.end;
    } else if (id === "cost" || id === "program") {
      if (c[id]) { anims.push(...c[id].play()); end = c[id].end(); }
    } else {
      // The table fills as quotes land, a fresh order each time; then the
      // architect's comparison comes up over it and is read.
      if (c.procurement) anims.push(...c.procurement.play(procurementShot(shuffled(QUOTE_COUNT)).boxes.map(b => b[5])));
      if (c.tender) {
        const tender = tenderShot();
        anims.push(...c.tender.play(tender.boxes.map(b => b[5] + TENDER_IN)));
        end = c.tender.end(tender.boxes.map(b => b[5] + TENDER_IN));
      }
      if (tenderCard.current) {
        anims.push(tenderCard.current.animate(
          [{ opacity: 0, transform: "translateY(18px)" }, { opacity: 1, transform: "none" }],
          { delay: TENDER_IN - 350, duration: 520, easing: "cubic-bezier(.16, 1, .3, 1)", fill: "both" },
        ));
      }
    }
    clock.current = { start: performance.now(), pausedAt: 0, anims, length: Math.max(end, 1000) + HOLD };
    setLength(clock.current.length);
    setT(0);
    return () => anims.forEach(a => a.cancel());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, round, still]);

  // Off screen or in a hidden tab, the clock stops.
  useEffect(() => {
    if (still || !stage.current || !("IntersectionObserver" in window)) return;
    const seen = new IntersectionObserver(([e]) => setShown(e.isIntersecting), { threshold: 0.2 });
    seen.observe(stage.current);
    const onVis = () => setShown(s => (document.hidden ? false : s || isOnScreen(stage.current)));
    document.addEventListener("visibilitychange", onVis);
    return () => { seen.disconnect(); document.removeEventListener("visibilitychange", onVis); };
  }, [still]);

  const running = !still && !paused && shown;
  useEffect(() => {
    if (still) return;
    const k = clock.current;
    if (running) {
      if (k.pausedAt) {
        k.start += performance.now() - k.pausedAt;
        k.pausedAt = 0;
        k.anims.forEach(a => { if (a.playState === "paused") a.play(); });
      }
    } else if (!k.pausedAt) {
      k.pausedAt = performance.now();
      k.anims.forEach(a => { if (a.playState === "running") a.pause(); });
    }
  }, [running, still, active, round]);

  // The clock: about ten readings a second drive the strip and the progress line.
  useEffect(() => {
    if (!running) return;
    let frame = 0, last = -1;
    const tick = () => {
      const k = clock.current;
      const now = performance.now() - k.start;
      if (now >= k.length) {
        advance();
        return;
      }
      if (now - last >= 90) { last = now; setT(now); }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [running, active, round]);

  const advance = useCallback(() => {
    setActive(a => {
      if (a === CHAPTERS.length - 1) setRound(r => r + 1);
      return (a + 1) % CHAPTERS.length;
    });
  }, []);
  const choose = (i: number) => {
    if (i === active) setRound(r => r + 1); // pressing the open one plays it again
    else setActive(i);
  };

  return (
    <section className="landing-stage" id="stage" ref={stage} aria-label="What SiteWise builds from your folder">
      <div className="stage-side">
      <div className="stage-steps" role="tablist" aria-orientation="vertical">
        {CHAPTERS.map((c, i) => (
          <button
            key={c.id}
            type="button"
            role="tab"
            id={`stage-tab-${c.id}`}
            aria-selected={i === active}
            aria-controls="stage-screen"
            className="stage-step"
            onClick={() => choose(i)}
          >
            <span className="stage-no" aria-hidden="true">{i + 1}</span>
            <span className="stage-step-title">{c.title}</span>
            <span className="stage-step-text">{c.text}</span>
            {!still && <span className="stage-line" aria-hidden="true"><i style={{ transform: `scaleX(${i === active ? Math.min(1, t / length) : 0})` }} /></span>}
          </button>
        ))}
      </div>
      {/* On a narrow screen the steps are numbers; this says the open one. */}
      <p className="stage-caption" aria-hidden="true"><b>{CHAPTERS[active].title}</b> {CHAPTERS[active].text}</p>
      {!still && (
        <button type="button" className="stage-pause" onClick={() => setPaused(p => !p)} aria-pressed={paused}>
          {paused ? "Play" : "Pause"}
        </button>
      )}
      </div>

      <div className="stage-screen" id="stage-screen" role="tabpanel" aria-labelledby={`stage-tab-${CHAPTERS[active].id}`}>
        {CHAPTERS.map((c, i) => (
          <div key={c.id} className="stage-view" data-on={i === active || undefined} aria-hidden={i !== active}>
            <div className="stage-band">
              <span className="stage-band-view">{c.view}</span>
              <span className="stage-band-job">House alterations, 41 Tallowwood Avenue</span>
              {c.id === "profile" && (
                <GroundButton primary small icon="refresh" tabIndex={-1} aria-hidden="true" busy={!still && i === active && t >= PRESS && t < replay.end - 300}>
                  Update
                </GroundButton>
              )}
            </div>
            {c.id === "profile" && (
              <>
                <figure className="landing-shot">
                  <div className="landing-shot-frame" ref={el => { frames.current.profile = el; }}>
                    <img src={profilePng} width={2000} height={1200} alt="The sample project's profile filling in as its documents are read: a one-sentence summary of a house extension in Concord, site and planning, the building, delivery, and the scope of works by building system." />
                  </div>
                </figure>
                <Strip replay={replay} t={i === active ? t : Infinity} />
              </>
            )}
            {c.id === "cost" && <Screen frameKey="cost" frames={frames} src={costPng} shot={shots.cost as Shot} alt="Cost plan being set up: fees and charges and consultant lines, then the architect's four stage fees and the BASIX fees arriving in the approved contract column." />}
            {c.id === "program" && <Screen frameKey="program" frames={frames} src={programPng} shot={shots.program as Shot} alt="Program being drawn: activities with days, start, finish and predecessors, and each bar of the Gantt chart from November 2026 to mid 2027." />}
            {c.id === "tenders" && (
              <div className="landing-pair">
                <Screen frameKey="procurement" frames={frames} src={procurementPng} shot={{ width: 2000, height: 1000 } as Shot} alt="Procurement: one row per consultant, from architect to town planner, with quotes landing in the firm cells and the awarded firms ticked." />
                <Screen frameKey="tender" frames={frames} src={tenderPng} shot={{ width: 2000, height: 1039 } as Shot} figureRef={tenderCard} alt="Tender evaluation for the architect: the firm's price against each stage of the RFP, totalling $44,000 excluding GST, awarded." />
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

function isOnScreen(el: Element | null) {
  if (!el) return false;
  const r = el.getBoundingClientRect();
  return r.bottom > 0 && r.top < window.innerHeight;
}

function Screen({ frameKey, frames, src, shot, alt, figureRef }: {
  frameKey: string;
  frames: React.RefObject<Record<string, HTMLDivElement | null>>;
  src: string;
  shot: Shot;
  alt: string;
  figureRef?: React.Ref<HTMLElement>;
}) {
  return (
    <figure className="landing-shot" ref={figureRef}>
      <div className="landing-shot-frame" ref={el => { frames.current[frameKey] = el; }}>
        <img src={src} width={shot.width} height={shot.height} alt={alt} />
      </div>
    </figure>
  );
}

/** The workbench strip at the foot of the profile, as it reads during an Update. */
function Strip({ replay, t }: { replay: Replay; t: number }) {
  const total = replay.docs.length;
  if (t < PRESS) {
    return (
      <div className="lp-strip" data-mode="rest">
        <Ring done={0} total={total} />
        <p>Included for profile: {total} of 19 documents</p>
        <p className="lp-next">{total} documents ready to read. Select Update to start.</p>
      </div>
    );
  }
  const states = docsAt(replay, t);
  const done = states.filter(s => s.state === "done").length;
  if (done === total && t >= replay.end - 300) {
    const passages = replay.docs.reduce((n, d) => n + d.units, 0);
    const cells = replay.shot.boxes.filter(b => b[6] === "pop").length;
    return (
      <div className="lp-strip" data-mode="done">
        <Ring done={total} total={total} />
        <p><b>Update finished.</b> Read {total} documents, {passages} passages.</p>
        <p className="lp-chips"><span><b>{replay.filled}</b> filled</span><span><b>{cells}</b> systems added</span><span><b>1</b> of yours read again, kept</span></p>
      </div>
    );
  }
  const reading = states.filter(s => s.state === "reading").length;
  const queued = states.filter(s => s.state === "queued").length;
  // The document most recently taken is the one the line follows.
  const now = replay.passes
    .map((p, i) => ({ p, i, s: states[i] }))
    .filter(x => x.s.state === "reading")
    .sort((a, b) => b.p.start - a.p.start)[0];
  const find = findAt(replay, t);
  return (
    <div className="lp-strip" data-mode="run">
      <ul className="lp-docs" aria-hidden="true">
        {replay.docs.map((d, i) => {
          const s = states[i];
          const rise = (n: number) => `inset(${100 - Math.min(100, (n / d.units) * 100)}% 0 0 0)`;
          return (
            <li key={d.name} className="lp-doc" data-state={s.state}>
              <span className="lp-sheet">
                <span className="lp-hatch" style={{ clipPath: rise(s.labelled) }} />
                <span className="lp-ink" style={{ clipPath: rise(s.evidence) }} />
              </span>
              <span className="lp-doc-name">{d.short}</span>
              <span className="lp-tick">{s.state === "done" && <i />}</span>
            </li>
          );
        })}
      </ul>
      <div className="lp-live">
        <p className="lp-status"><Ring done={done} total={total} />Reading documents · {reading} processing · {queued > 0 ? `${queued} queued · ` : ""}{done} of {total} documents read</p>
        {now && (
          <p className="lp-now">
            <b>{replay.docs[now.i].name}</b> · {now.s.pass} pass · {now.s.pass === "label" ? now.s.labelled : now.s.evidence} of {replay.docs[now.i].units} passages
          </p>
        )}
        {find && <p className="lp-find" key={find.at}><b>{find.what}</b> · <q>{find.excerpt}</q> · <span>{find.where}</span></p>}
      </div>
    </div>
  );
}

function Ring({ done, total }: { done: number; total: number }) {
  const finished = total > 0 && done === total;
  return (
    <svg className="lp-ring" viewBox="0 0 24 24" aria-hidden="true">
      <circle className="lp-ring-track" cx="12" cy="12" r="9" />
      {finished
        ? <path className="lp-ring-check" d="m7.5 12 3 3 6-6" />
        : <circle className="lp-ring-fill" cx="12" cy="12" r="9" pathLength="100" strokeDasharray={`${(done / Math.max(1, total)) * 100} 100`} />}
    </svg>
  );
}
