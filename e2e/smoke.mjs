// Run: NODE_PATH=$(npm root -g) node e2e/smoke.mjs  (backend on :8080, frontend on :3000)
import { createRequire } from "module";
const { chromium } = createRequire(import.meta.url)("playwright");
import assert from "assert";

const b = await chromium.launch({ executablePath: "/opt/pw-browsers/chromium" });
const ctx = () => b.newContext();
const admin = await (await ctx()).newPage();
await admin.goto("http://localhost:3000/");
await admin.setInputFiles('input[type=file]', new URL("./data.json", import.meta.url).pathname);
await admin.getByText("2 questions").waitFor();
await admin.getByRole("button", { name: "Create session" }).click();
await admin.getByText("Session ready").waitFor();
const share = await admin.inputValue('input[id^="Share"]');
await admin.getByRole("link", { name: "Open admin console" }).click();
await admin.getByText("Admin console").waitFor();

const clients = [];
for (const [name, pick] of [["Ann", "A"], ["Bob", "B"]]) {
  const p = await (await ctx()).newPage();
  await p.goto(share);
  await p.getByLabel("Your display name").fill(name);
  await p.getByRole("button", { name: "Join" }).click();
  await p.getByText("Waiting for the next question").waitFor();
  clients.push([p, pick]);
}
await admin.getByText("2 joined").waitFor();
await admin.getByRole("button", { name: "Start", exact: true }).click();
for (const [p, pick] of clients) {
  await p.getByText("Capital of France?").waitFor();
  await p.getByRole("button", { name: new RegExp("^" + pick) }).click();
}
await admin.getByText("2 of 2").waitFor();
// a client reload mid-question keeps its answer
await clients[0][0].reload();
await clients[0][0].getByRole("button", { name: /^A/ }).waitFor();
assert.equal(await clients[0][0].getByRole("button", { name: /^A/ }).getAttribute("aria-pressed"), "true");
await admin.getByRole("button", { name: "Finish early" }).click();
await admin.getByText("Results").first().waitFor();
await clients[1][0].getByText("answers locked").waitFor();
await clients[1][0].getByText(/your pick/).waitFor();
await admin.getByRole("button", { name: "Next question" }).click();
await admin.getByRole("button", { name: "Start", exact: true }).click();
await admin.getByRole("button", { name: "Finish early" }).click();
await admin.getByRole("button", { name: "Finish quiz" }).click();
await admin.getByText("Quiz complete").waitFor();
await clients[0][0].getByText("Quiz complete").waitFor();
await admin.screenshot({ path: "/tmp/admin.png" });
console.log("E2E OK");
await b.close();
