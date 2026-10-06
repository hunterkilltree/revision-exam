// Mirrors the Go snapshot types in backend/internal/session/views.go.
export type Phase = "idle" | "running" | "results" | "complete";
export type QStatus = "not_started" | "running" | "finished";
export type Tally = Record<string, number>;

export interface Option { key: string; text: string }
export interface QuestionView { id: string; text: string; options: Option[] }
export interface ListItem {
  id: string; text: string; status: QStatus;
  options?: Option[]; defaultDurationSec?: number; correctKey?: string;
}
export interface Timer { durationSec: number; startedAt: number; status: "idle" | "running" | "finished" }

export interface View {
  sessionId: string;
  role: "admin" | "client";
  serverNow: number;
  phase: Phase;
  currentQuestionIndex: number;
  questionCount: number;
  isLast: boolean;
  question?: QuestionView;
  timer: Timer;
  answeredCount: number;
  clientCount: number;
  joined?: boolean;
  myAnswer?: string;
  tally?: Tally;
  correctKey?: string;
  questions?: ListItem[];
  pastTallies?: Record<string, Tally>;
}

/** A snapshot plus the local clock reading at receipt, for skew-corrected countdowns. */
export interface Snapshot extends View { receivedAt: number }

export type Outbound =
  | { type: "join"; name: string; clientId?: string }
  | { type: "answer"; questionId: string; key: string }
  | { type: "start"; durationSec: number }
  | { type: "finish" }
  | { type: "next" }
  | { type: "ping" };

export const DURATIONS = [15, 10, 30, 60] as const;
export const durationLabel = (s: number) => (s === 60 ? "1 min" : `${s}s`);

export interface FieldError { path: string; message: string }
