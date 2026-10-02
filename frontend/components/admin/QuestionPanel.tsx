"use client";
import { useState } from "react";
import type { Outbound, Snapshot } from "@/lib/types";
import { Countdown } from "../shared/Countdown";
import { DurationSelector } from "./DurationSelector";
import { ResultsPanel } from "./ResultsPanel";

export function QuestionPanel({ s, selected, send }: { s: Snapshot; selected: number; send: (m: Outbound) => void }) {
  const items = s.questions ?? [];
  const item = items[selected];
  const isCurrent = selected === s.currentQuestionIndex;
  const [picked, setPicked] = useState<{ index: number; sec: number } | null>(null);
  const sec = picked?.index === s.currentQuestionIndex ? picked.sec : s.timer.durationSec;

  if (!item) return null;
  const header = (
    <>
      <p className="muted">Question {selected + 1} of {s.questionCount}</p>
      <h2>{item.text}</h2>
    </>
  );

  if (s.phase === "complete" && isCurrent) {
    return <div className="card stack"><h2>Quiz complete</h2><p className="muted">All {s.questionCount} questions have run.</p>
      <ResultsPanel item={item} tally={s.tally ?? {}} isLast canAdvance={false} onNext={() => {}} /></div>;
  }

  // Viewing a question that isn't the live one: read-only status / results.
  if (!isCurrent) {
    const past = s.pastTallies?.[item.id];
    return (
      <div className="card stack">
        {header}
        {past ? <ResultsPanel item={item} tally={past} isLast={false} canAdvance={false} onNext={() => {}} />
          : <p className="muted">This question has not been run yet.</p>}
      </div>
    );
  }

  if (s.phase === "results") {
    return <div className="card stack">{header}
      <ResultsPanel item={item} tally={s.tally ?? {}} isLast={s.isLast} canAdvance onNext={() => send({ type: "next" })} /></div>;
  }

  const running = s.phase === "running";
  return (
    <div className="card stack">
      {header}
      <ol type="A" className="stack" style={{ paddingLeft: 24 }}>
        {(item.options ?? []).map((o) => <li key={o.key}>{o.text}</li>)}
      </ol>
      <div className="row"><span className="muted">Duration</span>
        <DurationSelector value={sec} disabled={running} onChange={(v) => setPicked({ index: s.currentQuestionIndex, sec: v })} />
      </div>
      {running ? (
        <div className="row" style={{ justifyContent: "space-between" }}>
          <Countdown snapshot={s} />
          <div><strong className="mono">{s.answeredCount} of {s.clientCount}</strong> answered</div>
          <button className="btn danger" onClick={() => send({ type: "finish" })}>Finish early</button>
        </div>
      ) : (
        <div className="row">
          <button className="btn primary" onClick={() => send({ type: "start", durationSec: sec })}>Start</button>
          <span className="muted">{s.clientCount} joined</span>
        </div>
      )}
    </div>
  );
}
