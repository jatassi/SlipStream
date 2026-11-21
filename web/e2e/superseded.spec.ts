import { join } from "node:path";
import { DatabaseSync } from "node:sqlite";

import { expect, test } from "./fixtures.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 180_000 });

// An Admin's Retire writes this row; the test writes it into home's stopped
// library.db, since Retire has no screen in this build.
function retire(dataDir: string): void {
  const db = new DatabaseSync(join(dataDir, "library.db"));
  db.prepare(
    "INSERT INTO nodes_fence (id, state, by_key, by_node, addresses, fenced_at) VALUES (1, 'retired', NULL, NULL, '[]', ?)",
  ).run(Date.now() * 1000);
  db.close();
}

test("a retired Controller shows only the Superseded page until an Admin resumes it", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.node("stop", "home");
  retire(world.node.dataDir);
  await world.fw.node("start", "home");

  await page.goto("/library");
  await expect(page.getByRole("heading", { name: "This Controller was replaced" })).toBeVisible();
  await expect(page).toHaveURL(/\/superseded$/u);
  await expect(page.getByText(/an Admin stopped this Controller\./u)).toBeVisible();
  await expect(page.getByLabel("Join Code")).toBeVisible();
  expect((await world.api("GET", "/api/v1/nodes")).status).toBe(409);

  await page.getByRole("button", { name: "Details" }).click();
  await page.getByRole("button", { name: "Resume this Controller…" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Resume", exact: true }).click();
  await expect(page.getByRole("heading", { name: "This Controller was replaced" })).toHaveCount(0);
  await expect(page).not.toHaveURL(/\/superseded$/u);
  expect((await world.api("GET", "/api/v1/nodes")).status).toBe(200);
});
