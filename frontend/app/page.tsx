"use client";
import { useEffect, useState } from "react";
import { createSession, previewFile, sessionExists, type Created, type Preview } from "@/lib/api";
import type { FieldError } from "@/lib/types";
import { UploadDropzone } from "@/components/admin/UploadDropzone";
import { LinkCopy } from "@/components/admin/LinkCopy";

const ACTIVE = "quiz:activeHost";
type Active = { sessionId: string; hostLink: string; shareLink: string };

export default function SetupPage() {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [errors, setErrors] = useState<FieldError[]>([]);
  const [busy, setBusy] = useState(false);
  const [created, setCreated] = useState<Created | null>(null);
  const [active, setActive] = useState<Active | null>(null);

  // One session at a time per browser: offer to resume a still-live one.
  useEffect(() => {
    try {
      const a: Active | null = JSON.parse(localStorage.getItem(ACTIVE) ?? "null");
      if (a) sessionExists(a.sessionId).then((r) => (r?.exists && !r.ended ? setActive(a) : localStorage.removeItem(ACTIVE)));
    } catch { /* ignore */ }
  }, []);

  const onFile = async (f: File) => {
    setBusy(true); setErrors([]); setPreview(null); setFile(f);
    const r = await previewFile(f);
    setBusy(false);
    if (r.ok) setPreview(r.data); else { setErrors(r.errors); setFile(null); }
  };

  const create = async () => {
    if (!file) return;
    setBusy(true);
    const r = await createSession(file);
    setBusy(false);
    if (!r.ok) return setErrors(r.errors);
    setCreated(r.data);
    try { localStorage.setItem(ACTIVE, JSON.stringify({ sessionId: r.data.sessionId, hostLink: r.data.hostLink, shareLink: r.data.shareLink })); } catch { /* ignore */ }
  };

  if (created) {
    return (
      <div className="page narrow stack">
        <h1>Session ready</h1>
        <p className="muted">{created.questionCount} questions imported.</p>
        <LinkCopy label="Host link (keep private)" href={created.hostLink} />
        <LinkCopy label="Share link (give to participants)" href={created.shareLink} />
        <a className="btn primary" href={created.hostLink} style={{ display: "inline-block", textDecoration: "none" }}>Open admin console</a>
      </div>
    );
  }

  return (
    <div className="page narrow stack">
      <h1>Live Quiz Sessions</h1>
      {active && (
        <div className="card stack">
          <strong>You have a session in progress.</strong>
          <div className="row">
            <a className="btn primary" href={active.hostLink} style={{ textDecoration: "none" }}>Resume</a>
            <button className="btn" onClick={() => { localStorage.removeItem(ACTIVE); setActive(null); }}>Start a new one</button>
          </div>
        </div>
      )}
      {!active && <UploadDropzone onFile={onFile} busy={busy} />}
      {errors.length > 0 && (
        <div className="errors" role="alert">
          <strong>Couldn&apos;t import that file:</strong>
          <ul>{errors.map((e, i) => <li key={i}><code>{e.path}</code> — {e.message}</li>)}</ul>
        </div>
      )}
      {preview && (
        <div className="card stack">
          <h2>{preview.questionCount} questions</h2>
          <ol>{preview.questions.map((q) => <li key={q.id}>{q.text}</li>)}</ol>
          <button className="btn primary" disabled={busy} onClick={create}>Create session</button>
        </div>
      )}
    </div>
  );
}
