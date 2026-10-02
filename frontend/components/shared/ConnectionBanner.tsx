import type { ConnStatus } from "@/lib/socket";

export function ConnectionBanner({ status }: { status: ConnStatus }) {
  if (status === "open") return null;
  return <div className="banner" role="status">{status === "connecting" ? "Connecting…" : "Reconnecting…"}</div>;
}
