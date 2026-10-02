import { describe, expect, it } from "vitest";
import { remainingMs } from "./clock";

describe("remainingMs", () => {
  const timer = { startedAt: 10_000, durationSec: 30 };
  it("counts down from the server clock", () => {
    expect(remainingMs(timer, 12_000, 500, 500)).toBe(28_000);
  });
  it("advances with local elapsed time since receipt", () => {
    expect(remainingMs(timer, 12_000, 500, 5_500)).toBe(23_000);
  });
  it("is independent of the absolute local clock (skew)", () => {
    expect(remainingMs(timer, 12_000, 9_999_000, 10_004_000)).toBe(23_000);
  });
  it("clamps at zero", () => {
    expect(remainingMs(timer, 99_000, 0, 0)).toBe(0);
  });
});
