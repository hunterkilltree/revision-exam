"use client";
import { useState } from "react";

export function LinkCopy({ label, href }: { label: string; href: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try { await navigator.clipboard.writeText(href); setCopied(true); setTimeout(() => setCopied(false), 1500); } catch { /* ignore */ }
  };
  return (
    <div>
      <label className="muted" htmlFor={label}>{label}</label>
      <div className="row">
        <input id={label} className="input mono" style={{ flex: 1 }} readOnly value={href} onFocus={(e) => e.currentTarget.select()} />
        <button className="btn" onClick={copy}>{copied ? "Copied" : "Copy"}</button>
      </div>
    </div>
  );
}
