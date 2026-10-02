"use client";
export function UploadDropzone({ onFile, busy }: { onFile: (f: File) => void; busy: boolean }) {
  return (
    <label className="card stack" style={{ borderStyle: "dashed", textAlign: "center", cursor: "pointer" }}>
      <strong>{busy ? "Checking…" : "Choose a data.json file"}</strong>
      <span className="muted">Max 1 MB · up to 200 questions</span>
      <input type="file" accept="application/json,.json" disabled={busy} className="sr"
        onChange={(e) => { const f = e.target.files?.[0]; if (f) onFile(f); e.target.value = ""; }} />
    </label>
  );
}
