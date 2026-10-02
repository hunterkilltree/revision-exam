"use client";
import { Suspense, use, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useSession } from "@/lib/socket";
import { sessionExists } from "@/lib/api";
import { ConnectionBanner } from "@/components/shared/ConnectionBanner";
import { QuestionList } from "@/components/admin/QuestionList";
import { QuestionPanel } from "@/components/admin/QuestionPanel";
import { LinkCopy } from "@/components/admin/LinkCopy";

function Console({ sessionId }: { sessionId: string }) {
  const key = useSearchParams().get("key") ?? "";
  const { state, status, error, send } = useSession(sessionId, key);
  const [sel, setSel] = useState<{ cur: number; pick: number } | null>(null);
  const [missing, setMissing] = useState(false);
  useEffect(() => { sessionExists(sessionId).then((r) => r && !r.exists && setMissing(true)); }, [sessionId]);

  if (missing) return <div className="page narrow"><h1>Session not found</h1><p className="muted">It may have expired. <a href="/">Start a new one.</a></p></div>;
  if (!key) return <div className="page narrow"><h1>Host link required</h1><p className="muted">Open the full host link, including <code>?key=…</code>.</p></div>;

  // The list follows the live question unless the host clicked a different one; moving on resets it.
  const cur = state?.currentQuestionIndex ?? 0;
  const selected = sel && sel.cur === cur ? sel.pick : cur;
  const share = typeof window === "undefined" ? "" : `${window.location.origin}/join/${sessionId}`;

  return (
    <>
      <ConnectionBanner status={status} />
      <div className="page">
        <div className="row" style={{ justifyContent: "space-between", marginBottom: 16 }}>
          <h1>Admin console</h1>
          <div style={{ minWidth: 320, flex: 1, maxWidth: 480 }}><LinkCopy label="Share link" href={share} /></div>
        </div>
        {error && Date.now() - error.at < 4000 && <div className="errors" role="alert">{error.message}</div>}
        {!state ? <p className="muted">Loading…</p> : (
          <div className="console">
            <QuestionList items={state.questions ?? []} current={cur} selected={selected} onSelect={(i) => setSel({ cur, pick: i })} />
            <QuestionPanel s={state} selected={selected} send={send} />
          </div>
        )}
      </div>
    </>
  );
}

export default function AdminPage({ params }: { params: Promise<{ sessionId: string }> }) {
  const { sessionId } = use(params);
  return <Suspense><Console sessionId={sessionId} /></Suspense>;
}
