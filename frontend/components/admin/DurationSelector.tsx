import { DURATIONS, durationLabel } from "@/lib/types";

export function DurationSelector({ value, onChange, disabled }: { value: number; onChange: (s: number) => void; disabled: boolean }) {
  return (
    <div className="seg" role="group" aria-label="Duration">
      {DURATIONS.map((d) => (
        <button key={d} aria-pressed={value === d} disabled={disabled} onClick={() => onChange(d)}>{durationLabel(d)}</button>
      ))}
    </div>
  );
}
