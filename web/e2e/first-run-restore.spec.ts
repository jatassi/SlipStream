import { join } from "node:path";

import { z } from "zod";

import { declinePasskeyPrompt, submitPin } from "../harness/sign-in.ts";
import { type World, expect, test } from "./fixtures.ts";

test.use({ scenario: "dead-controller" });
test.describe.configure({ timeout: 300_000 });

const Listed = z.object({
  backups: z.array(z.object({ name: z.string(), trigger: z.string(), state: z.string() })),
});
const Passphrase = z.object({ passphrase: z.string() });

// The API key never re-confirms, so it reads the passphrase without the PIN.
async function takeBackup(world: World): Promise<{ file: string; passphrase: string }> {
  expect((await world.api("POST", "/api/v1/backups")).status).toBe(202);
  let name = "";
  await expect
    .poll(
      async () => {
        const { body } = await world.api("GET", "/api/v1/backups");
        const manual = Listed.parse(body).backups.find(
          (backup) => backup.trigger === "manual" && backup.state === "verified",
        );
        name = manual?.name ?? "";
        return name;
      },
      { timeout: 60_000 },
    )
    .not.toBe("");
  const { body } = await world.api("GET", "/api/v1/backups/passphrase");
  return {
    file: join(world.node.dataDir, "backups", name),
    passphrase: Passphrase.parse(body).passphrase,
  };
}

test("an Unclaimed Node restores the dead Controller's Backup from its first-run page, then signs in", async ({
  world,
  newContext,
}) => {
  const { file, passphrase } = await takeBackup(world);
  await world.fw.node("kill", "home");

  const page = await (await newContext("spare")).newPage();
  await page.goto("/setup");
  await page.getByLabel("Setup code").pressSequentially(await world.setupCode("spare"));
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByRole("button", { name: "Restore a Backup" }).click();
  await page.getByLabel("Backup file").setInputFiles(file);
  await page.getByRole("textbox", { name: "Backup passphrase" }).fill(passphrase);
  await page.getByRole("button", { name: "Open the Backup" }).click();
  await expect(
    page.getByText(/^This Backup was taken .* It's a Backup of home and lists 2 Nodes\.$/u),
  ).toBeVisible();

  const restore = page.getByRole("button", { name: "Restore", exact: true });
  await expect(restore).toBeDisabled();
  await page
    .getByRole("checkbox", { name: "home is off, and I'll wipe it if it starts again" })
    .check();
  await restore.click();
  await expect(page).toHaveURL(/\/sign-in$/u, { timeout: 120_000 });

  const admin = world.admin();
  await declinePasskeyPrompt(page, await submitPin(page, admin.email, admin.pin));
  await expect(
    page.getByRole("heading", { name: "Find the old Controller's folders", level: 1 }),
  ).toBeVisible();
});
