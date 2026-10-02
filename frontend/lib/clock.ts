/**
 * Seconds left on the question clock. The server's `serverNow` anchors the clock, and the time
 * elapsed locally since the snapshot arrived advances it, so client clock skew never matters.
 */
export function remainingMs(
  timer: { startedAt: number; durationSec: number },
  serverNow: number,
  receivedAt: number,
  localNow: number,
): number {
  const estimatedServerNow = serverNow + (localNow - receivedAt);
  return Math.max(0, timer.startedAt + timer.durationSec * 1000 - estimatedServerNow);
}
