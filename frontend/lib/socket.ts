"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { WS_URL } from "./api";
import type { Outbound, Snapshot } from "./types";

export type ConnStatus = "connecting" | "open" | "reconnecting";

interface Stored { clientId?: string; name?: string }
const storeKey = (id: string) => `quiz:client:${id}`;

export function loadClient(sessionId: string): Stored {
  try { return JSON.parse(localStorage.getItem(storeKey(sessionId)) ?? "{}"); } catch { return {}; }
}
function saveClient(sessionId: string, v: Stored) {
  try { localStorage.setItem(storeKey(sessionId), JSON.stringify(v)); } catch { /* storage unavailable */ }
}

/**
 * Connects to the session socket, auto-reconnects with backoff (0.5s → 5s) and exposes the last
 * snapshot. Holds no other session state: a refresh just reconnects and receives the snapshot.
 */
export function useSession(sessionId: string, hostKey?: string) {
  const [state, setState] = useState<Snapshot | null>(null);
  const [status, setStatus] = useState<ConnStatus>("connecting");
  const [error, setError] = useState<{ code: string; message: string; at: number } | null>(null);
  const ws = useRef<WebSocket | null>(null);

  const send = useCallback((m: Outbound) => {
    if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(JSON.stringify(m));
  }, []);

  const join = useCallback((name: string) => {
    const prev = loadClient(sessionId);
    saveClient(sessionId, { ...prev, name });
    send({ type: "join", name, clientId: prev.clientId });
  }, [sessionId, send]);

  useEffect(() => {
    let closed = false, attempt = 0, timer: ReturnType<typeof setTimeout>, ping: ReturnType<typeof setInterval>;

    const connect = () => {
      const q = hostKey ? `?key=${encodeURIComponent(hostKey)}` : "";
      const sock = new WebSocket(`${WS_URL}/ws/${encodeURIComponent(sessionId)}${q}`);
      ws.current = sock;
      sock.onopen = () => {
        attempt = 0;
        setStatus("open");
        if (!hostKey) { // resume a previous identity after (re)connect
          const s = loadClient(sessionId);
          if (s.name) sock.send(JSON.stringify({ type: "join", name: s.name, clientId: s.clientId }));
        }
        clearInterval(ping);
        ping = setInterval(() => sock.readyState === WebSocket.OPEN && sock.send('{"type":"ping"}'), 15000);
      };
      sock.onmessage = (ev) => {
        const m = JSON.parse(ev.data);
        if (m.type === "state") setState({ ...m, receivedAt: Date.now() });
        else if (m.type === "joined") saveClient(sessionId, { ...loadClient(sessionId), clientId: m.clientId });
        else if (m.type === "error") setError({ code: m.code, message: m.message, at: Date.now() });
      };
      sock.onclose = () => {
        clearInterval(ping);
        if (closed) return;
        setStatus("reconnecting");
        timer = setTimeout(connect, Math.min(5000, 500 * 2 ** attempt++));
      };
    };
    connect();
    return () => { closed = true; clearTimeout(timer); clearInterval(ping); ws.current?.close(); };
  }, [sessionId, hostKey]);

  return { state, status, error, send, join };
}

/** Re-renders ~10x/s while `active`, returning Date.now(). */
export function useNow(active: boolean) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const t = setInterval(() => setNow(Date.now()), 100);
    return () => clearInterval(t);
  }, [active]);
  return now;
}
