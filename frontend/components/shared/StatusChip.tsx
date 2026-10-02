import type { QStatus } from "@/lib/types";

const LABEL: Record<QStatus, string> = { not_started: "Not started", running: "Running", finished: "Finished" };

export function StatusChip({ status }: { status: QStatus }) {
  return <span className={`chip ${status === "not_started" ? "" : status}`}>{LABEL[status]}</span>;
}
