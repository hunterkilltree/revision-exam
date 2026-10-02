import type { Snapshot } from "@/lib/types";
import { BarChart } from "../shared/BarChart";

export function LockedResultsView({ s }: { s: Snapshot }) {
  const q = s.question!;
  return (
    <div className="card stack">
      <p className="muted">Question {s.currentQuestionIndex + 1} of {s.questionCount} · answers locked</p>
      <h1>{q.text}</h1>
      <div className="opts" aria-hidden="true">
        {q.options.map((o) => (
          <button key={o.key} className="opt" disabled aria-pressed={s.myAnswer === o.key}><span className="k">{o.key}</span>{o.text}</button>
        ))}
      </div>
      <BarChart options={q.options} tally={s.tally ?? {}} mine={s.myAnswer} />
      {!s.myAnswer && <p className="muted">You didn&apos;t answer this one.</p>}
    </div>
  );
}
