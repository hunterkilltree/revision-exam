import type { FieldError } from "./types";

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
export const WS_URL = API_URL.replace(/^http/, "ws");

export interface Preview { questionCount: number; questions: { id: string; text: string }[] }
export interface Created extends Preview { sessionId: string; hostKey: string; hostLink: string; shareLink: string }
export type Result<T> = { ok: true; data: T } | { ok: false; errors: FieldError[] };

async function upload<T>(path: string, file: File): Promise<Result<T>> {
  const body = new FormData();
  body.append("file", file);
  try {
    const res = await fetch(API_URL + path, { method: "POST", body });
    const json = await res.json().catch(() => ({}));
    if (res.ok) return { ok: true, data: json as T };
    return { ok: false, errors: json.errors ?? [{ path: "$", message: `Server error (${res.status})` }] };
  } catch {
    return { ok: false, errors: [{ path: "$", message: "Could not reach the quiz server." }] };
  }
}

export const previewFile = (f: File) => upload<Preview>("/sessions/preview", f);
export const createSession = (f: File) => upload<Created>("/sessions", f);

export async function sessionExists(id: string): Promise<{ exists: boolean; ended: boolean } | null> {
  try {
    const res = await fetch(`${API_URL}/sessions/${encodeURIComponent(id)}`);
    return res.ok ? await res.json() : null;
  } catch {
    return null;
  }
}
