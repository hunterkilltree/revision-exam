"use client";
import { useEffect, useRef, useState } from "react";
import { remainingMs } from "@/lib/clock";
import { useNow } from "@/lib/socket";
import type { Snapshot } from "@/lib/types";

/** Seconds left (ceil) for a running snapshot; 0 otherwise. Exposed so views can lock inputs locally at 0. */
export function useSecondsLeft(s: Snapshot): number {
  const running = s.timer.status === "running";
  const now = useNow(running);
  return running ? Math.ceil(remainingMs(s.timer, s.serverNow, s.receivedAt, now) / 1000) : 0;
}

export function Countdown({ snapshot }: { snapshot: Snapshot }) {
  const left = useSecondsLeft(snapshot);
  const [announce, setAnnounce] = useState("");
  const last = useRef(-1);
  useEffect(() => {
    if ((left === 10 || left === 5) && last.current !== left) setAnnounce(`${left} seconds left`);
    last.current = left;
  }, [left]);
  const mm = String(Math.floor(left / 60)).padStart(2, "0"), ss = String(left % 60).padStart(2, "0");
  return (
    <div>
      <div className={`timer ${left <= 5 ? "low" : ""}`} aria-hidden="true">{mm}:{ss}</div>
      <div className="sr" aria-live="polite">{announce}</div>
    </div>
  );
}
