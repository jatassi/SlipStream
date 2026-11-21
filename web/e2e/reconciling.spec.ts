import { join } from "node:path";
import { DatabaseSync } from "node:sqlite";

import { expect, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 300_000 });

// Until a restore lands, the test writes the rows a restore onto a new
// machine writes into home's stopped library.db.
function beginRestore(dataDir: string): void {
  const db = new DatabaseSync(join(dataDir, "library.db"));
  const now = Date.now() * 1000;
  db.prepare(
    "INSERT INTO reconciling_state (id, restore_id, started_at, cause, backup_name, backup_taken_at, new_machine) VALUES (1, 'e2e', ?, 'restore', 'nightly', ?, 1)",
  ).run(now, now);
  db.exec("DELETE FROM reconciling_node");
  db.prepare(
    "INSERT INTO reconciling_node (node_id, cause, since, round) SELECT id, 'restore', ?, 1 FROM nodes_node",
  ).run(now);
  db.exec("INSERT INTO reconciling_root (root_id) SELECT id FROM roots_root");
  db.close();
}

test("after a restore the folders step comes first, then the banner names the next step until Continue without ends it", async ({
  adminPage: page,
  world,
}) => {
  await world.fw.node("stop", "seedbox");
  await world.fw.node("stop", "home");
  beginRestore(world.node.dataDir);
  await world.fw.node("start", "home");

  await page.goto("/library");
  await expect(
    page.getByRole("heading", { name: "Find the old Controller's folders", level: 1 }),
  ).toBeVisible();
  await expect(page.getByRole("list", { name: "Folders" })).toContainText("The folder is there.");
  await page.getByRole("button", { name: "Save all" }).click();
  await expect(page.getByRole("heading", { name: "Reconciling", level: 1 })).toBeVisible();
  await expect(page).toHaveURL(/\/reconciling$/u);

  const readd = await world.api("POST", "/api/v1/reconciling/torrents/readd", {
    reason: "not_held",
    found: "files",
  });
  expect(readd.status).toBe(204);

  await page.goto("/library");
  const banner = page.getByRole("complementary", { name: "Reconciling" });
  await expect(banner).toContainText(
    "SlipStream is matching the restored library with the torrents and files on your Nodes.",
  );
  await expect(banner).toContainText(/Pair seedbox again/u);
  if (isPhone(page)) {
    await banner.getByRole("link").click();
  } else {
    await banner.getByRole("button", { name: "Open Reconciling" }).click();
  }

  const nodes = page.getByRole("list", { name: "Nodes" });
  await expect(nodes).toContainText(
    "seedbox is still paired with the old Controller. Pair it again.",
  );
  await nodes.getByRole("button", { name: "Continue without seedbox" }).click();
  await expect(page.getByRole("button", { name: "Finish now" })).toHaveCount(0, {
    timeout: 60_000,
  });
  await page.goto("/library");
  await expect(banner).toHaveCount(0);
});
