import type { Snapshot } from "@/lib/types";

export function WaitingView({ s, name }: { s: Snapshot; name?: string }) {
  return (
    <div className="card center">
      <p className="muted">Question {s.currentQuestionIndex + 1} of {s.questionCount}</p>
      <h1>Waiting for the next question…</h1>
      {name && <p className="muted">You&apos;re in as <strong>{name}</strong>.</p>}
    </div>
  );
}
