"use client";
import { use, useEffect, useState } from "react";
import { loadClient, useSession } from "@/lib/socket";
import { sessionExists } from "@/lib/api";
import { ConnectionBanner } from "@/components/shared/ConnectionBanner";
import { JoinForm } from "@/components/client/JoinForm";
import { WaitingView } from "@/components/client/WaitingView";
import { AnsweringView } from "@/components/client/AnsweringView";
import { LockedResultsView } from "@/components/client/LockedResultsView";

export default function JoinPage({ params }: { params: Promise<{ sessionId: string }> }) {
  const { sessionId } = use(params);
  const { state, status, error, send, join } = useSession(sessionId);
  const [gone, setGone] = useState<string | null>(null);
  const [name, setName] = useState<string>();
  useEffect(() => { setName(loadClient(sessionId).name); }, [sessionId]);
  useEffect(() => {
    sessionExists(sessionId).then((r) => {
      if (r === null) return;
      if (!r.exists) setGone("This session doesn't exist or has expired.");
    });
  }, [sessionId]);

  const body = () => {
    if (gone) return <div className="card"><h1>Session not found</h1><p className="muted">{gone}</p></div>;
    if (!state) return <p className="muted">Connecting…</p>;
    if (!state.joined) {
      return <JoinForm onJoin={(n) => { setName(n); join(n); }} error={error && Date.now() - error.at < 5000 ? error.message : undefined} />;
    }
    if (state.phase === "complete") return <div className="card center"><h1>Quiz complete</h1><p className="muted">Thanks for playing!</p></div>;
    if (state.phase === "running" && state.question) return <AnsweringView key={state.question.id} s={state} send={send} />;
    if (state.phase === "results" && state.question) return <LockedResultsView s={state} />;
    return <WaitingView s={state} name={name} />;
  };

  return (
    <>
      <ConnectionBanner status={status} />
      <div className="page narrow">{body()}</div>
    </>
  );
}
