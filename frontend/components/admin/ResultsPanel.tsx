import type { ListItem, Tally } from "@/lib/types";
import { BarChart } from "../shared/BarChart";

export function ResultsPanel({ item, tally, isLast, canAdvance, onNext }: {
  item: ListItem; tally: Tally; isLast: boolean; canAdvance: boolean; onNext: () => void;
}) {
  return (
    <div className="stack">
      <h2>Results</h2>
      <BarChart options={item.options ?? []} tally={tally} correct={item.correctKey} />
      {canAdvance && (
        <button className="btn primary" onClick={onNext}>{isLast ? "Finish quiz" : "Next question"}</button>
      )}
    </div>
  );
}
