"use client";
import { useState } from "react";

export function JoinForm({ onJoin, error }: { onJoin: (name: string) => void; error?: string }) {
  const [name, setName] = useState("");
  return (
    <form className="card stack" onSubmit={(e) => { e.preventDefault(); if (name.trim()) onJoin(name.trim()); }}>
      <h1>Join the quiz</h1>
      <label htmlFor="name">Your display name</label>
      <input id="name" className="input" value={name} maxLength={40} autoFocus autoComplete="nickname" onChange={(e) => setName(e.target.value)} />
      {error && <div className="errors" role="alert">{error}</div>}
      <button className="btn primary" disabled={!name.trim()}>Join</button>
    </form>
  );
}
