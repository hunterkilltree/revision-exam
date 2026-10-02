"use client";
import { useState } from "react";
import type { Outbound, Snapshot } from "@/lib/types";
import { Countdown, useSecondsLeft } from "../shared/Countdown";

export function AnsweringView({ s, send }: { s: Snapshot; send: (m: Outbound) => void }) {
  const q = s.question!;
  const left = useSecondsLeft(s);
  // Optimistic pick, keyed by question so it never leaks into the next one; the server's myAnswer wins once it arrives.
  const [opt, setOpt] = useState<{ qid: string; key: string } | null>(null);
  const mine = opt?.qid === q.id ? opt.key : s.myAnswer;
  const locked = left <= 0; // lock locally at 0; wait for the server's results state to switch screens
  const pick = (key: string) => {
    if (locked) return;
    setOpt({ qid: q.id, key });
    send({ type: "answer", questionId: q.id, key });
  };
  return (
    <div className="card stack">
      <div className="row" style={{ justifyContent: "space-between" }}>
        <span className="muted">Question {s.currentQuestionIndex + 1} of {s.questionCount}</span>
        <Countdown snapshot={s} />
      </div>
      <h1>{q.text}</h1>
      <div className="opts">
        {q.options.map((o) => (
          <button key={o.key} className="opt" aria-pressed={mine === o.key} disabled={locked} onClick={() => pick(o.key)}>
            <span className="k">{o.key}</span>{o.text}
          </button>
        ))}
      </div>
      <p className="muted" aria-live="polite">{mine ? `Your answer: ${mine}. You can change it until time runs out.` : "Tap an option to answer."}</p>
    </div>
  );
}
