import { readFile } from "node:fs/promises";
import { connect } from "node:net";

import type { Page } from "@playwright/test";

import { type World, expect, test } from "./fixtures.ts";
import { navigate } from "./shell.ts";

test.describe.configure({ timeout: 120_000 });

type Section = "Logs" | "Log settings";

async function openSection(page: Page, section: Section): Promise<void> {
  await navigate(page, "Nodes");
  await page.getByRole("link", { name: new RegExp(`^${section}`, "u") }).click();
  await expect(page).toHaveURL(section === "Logs" ? /\/logs$/u : /\/log-settings$/u);
}

// A connection that never finishes TLS fails the link's handshake, which the
// Node logs at Debug.
function failLinkHandshake(world: World): Promise<void> {
  const address = world.node.listeners["link"];
  if (address === undefined) {
    throw new Error(`${world.node.name} lists no link listener`);
  }
  const port = Number(address.slice(address.lastIndexOf(":") + 1));
  const host = address.slice(0, address.lastIndexOf(":"));
  return new Promise((resolve, reject) => {
    const socket = connect({ host, port }, () => {
      socket.end("not tls\r\n\r\n", () => {
        resolve();
      });
    });
    socket.on("error", reject);
  });
}

const central = Buffer.from([0x50, 0x4b, 0x01, 0x02]);

function zipEntries(zip: Buffer): string[] {
  const names: string[] = [];
  for (let at = zip.indexOf(central); at !== -1; at = zip.indexOf(central, at + central.length)) {
    const length = zip.readUInt16LE(at + 28);
    names.push(zip.toString("utf8", at + 46, at + 46 + length));
  }
  return names;
}

const handshakeRows = (page: Page) =>
  page
    .getByRole("list", { name: "Log" })
    .getByRole("button")
    .filter({ hasText: "link handshake failed" });

test("a Debug level set in Log settings shows Debug lines live, through pause and resume, and the logs download", async ({
  adminPage: page,
  world,
}) => {
  await openSection(page, "Log settings");
  await page.getByRole("combobox", { name: "Level" }).click();
  await page.getByRole("option", { name: "Debug" }).click();
  await expect(page.getByText("Log settings saved")).toBeVisible();

  await openSection(page, "Logs");
  await expect(page.getByRole("status").filter({ hasText: /lines?$/u })).toBeVisible();
  await failLinkHandshake(world);
  await expect(handshakeRows(page)).toHaveCount(1, { timeout: 15_000 });
  await expect(handshakeRows(page)).toContainText("Debug");

  await page.getByRole("button", { name: "Pause" }).click();
  await failLinkHandshake(world);
  await expect(page.getByText(/^Paused, [1-9]\d* new$/u)).toBeVisible({ timeout: 15_000 });
  await expect(handshakeRows(page)).toHaveCount(1);
  await page.getByRole("button", { name: "Resume" }).click();
  await expect(handshakeRows(page)).toHaveCount(2);

  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download logs" }).click();
  const path = await (await download).path();
  expect(zipEntries(await readFile(path))).toContain("slipstream.log");
});
