import type { Option, Tally } from "@/lib/types";

export function percentages(tally: Tally, keys: string[]) {
  const total = keys.reduce((n, k) => n + (tally[k] ?? 0), 0);
  return { total, pct: (k: string) => (total ? Math.round(((tally[k] ?? 0) / total) * 100) : 0) };
}

/** Pure CSS bars, with a text table fallback for screen readers. */
export function BarChart({ options, tally, mine, correct }: { options: Option[]; tally: Tally; mine?: string; correct?: string }) {
  const { total, pct } = percentages(tally, options.map((o) => o.key));
  return (
    <div>
      {total === 0 && <p className="muted"><strong>No answers</strong> — nobody picked an option.</p>}
      <div className="bars" aria-hidden="true">
        {options.map((o) => (
          <div key={o.key} className={`bar ${mine === o.key ? "mine" : ""} ${correct === o.key ? "correct" : ""} ${correct && mine === o.key && mine !== correct ? "wrong" : ""}`}>
            <div className="lbl">
              <span><strong>{o.key}</strong> {o.text}{mine === o.key && (correct && mine !== correct ? " ✗ your pick" : " ✓ your pick")}{correct === o.key && " ★ correct answer"}</span>
              <span className="mono">{tally[o.key] ?? 0} · {pct(o.key)}%</span>
            </div>
            <div className="track"><div className="fill" style={{ width: `${pct(o.key)}%` }} /></div>
          </div>
        ))}
      </div>
      {correct && (
        <p className={`correct-note ${mine && mine !== correct ? "wrong" : ""}`} role="status">
          Correct answer: <strong>{correct}. {options.find((o) => o.key === correct)?.text}</strong>
          {mine !== undefined && mine !== "" && <> — you were {mine === correct ? "right ✓" : "wrong ✗"}</>}
        </p>
      )}
      <table className="sr">
        <caption>Results</caption>
        <tbody>
          {options.map((o) => (
            <tr key={o.key}><th>{o.key}. {o.text}{correct === o.key && " (correct answer)"}</th><td>{tally[o.key] ?? 0} answers, {pct(o.key)}%</td></tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
