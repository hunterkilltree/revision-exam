import { describe, expect, it } from "vitest";
import { percentages } from "./BarChart";

describe("percentages", () => {
  it("handles nobody answering", () => {
    const { total, pct } = percentages({ A: 0, B: 0 }, ["A", "B"]);
    expect(total).toBe(0);
    expect(pct("A")).toBe(0);
  });
  it("rounds shares", () => {
    const { pct } = percentages({ A: 1, B: 2 }, ["A", "B"]);
    expect([pct("A"), pct("B")]).toEqual([33, 67]);
  });
});
