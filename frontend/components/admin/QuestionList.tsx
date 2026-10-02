import type { ListItem } from "@/lib/types";
import { StatusChip } from "../shared/StatusChip";

export function QuestionList({ items, current, selected, onSelect }: {
  items: ListItem[]; current: number; selected: number; onSelect: (i: number) => void;
}) {
  return (
    <nav aria-label="Questions" className="card">
      <ul className="qlist">
        {items.map((q, i) => (
          <li key={q.id}>
            <button aria-current={i === selected} onClick={() => onSelect(i)}>
              <span className="qt">{i + 1}. {q.text}</span>
              <StatusChip status={q.status} />
            </button>
          </li>
        ))}
      </ul>
    </nav>
  );
}
