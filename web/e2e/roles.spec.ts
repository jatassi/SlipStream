import type { Page } from "@playwright/test";
import { z } from "zod";

import { node } from "../harness/manifest.ts";
import { type World, expect, test } from "./fixtures.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 180_000 });

const Board = z.object({
  basis: z.string(),
  nodes: z.array(z.object({ id: z.string(), name: z.string(), assigned: z.array(z.string()) })),
});
const Added = z.object({ work: z.object({ id: z.string() }) });
const Grabs = z.object({ grabs: z.array(z.object({ stage: z.string() })) });

function idOf(world: World, name: string): string {
  const id = node(world.manifest, name).nodeId;
  if (id === undefined) {
    throw new Error(`the manifest has no id for ${name}`);
  }
  return id;
}

async function boardOf(world: World) {
  return Board.parse((await world.api("GET", "/api/v1/nodes/roles")).body);
}

async function assignedOf(world: World, name: string): Promise<string[]> {
  return (await boardOf(world)).nodes.find((seat) => seat.name === name)?.assigned ?? [];
}

const role = (page: Page, name: string) =>
  page.getByRole("main").getByRole("checkbox", { name, exact: true });
const review = (page: Page) => page.getByRole("dialog", { name: "Review the change" });

test("unticking Downloading on seedbox reviews the handback to home, then applies it", async ({
  adminPage: page,
  world,
}) => {
  await page.goto(`/nodes/${idOf(world, "seedbox")}/roles`);
  await expect(role(page, "Downloading")).toBeChecked();
  await expect(page.getByRole("switch", { name: "Acquire" })).toBeChecked();
  await role(page, "Downloading").click();
  await page.getByRole("button", { name: "Review", exact: true }).click();

  const changes = review(page).getByRole("list", { name: "What changes" });
  await expect(changes).toContainText("seedbox stops Downloading.");
  await expect(changes).toContainText(
    "home takes over Downloading, since SlipStream always needs a Node that downloads.",
  );
  await review(page).getByRole("button", { name: "Apply" }).click();
  await expect(review(page)).toHaveCount(0);
  await expect(role(page, "Downloading")).not.toBeChecked();
  await expect(page.getByRole("button", { name: "Review", exact: true })).toBeDisabled();
  expect(await assignedOf(world, "seedbox")).toEqual(["searching"]);
  expect(await assignedOf(world, "home")).toContain("downloading");
});

test("a change someone made since the Review opened comes back as a fresh Review", async ({
  adminPage: page,
  world,
}) => {
  const seedbox = idOf(world, "seedbox");
  await page.goto(`/nodes/${seedbox}/roles`);
  await role(page, "Subtitles").click();
  await page.getByRole("button", { name: "Review", exact: true }).click();
  await expect(review(page).getByRole("list", { name: "What changes" })).toHaveText(
    "seedbox starts Subtitles.",
  );

  const { basis } = await boardOf(world);
  const meanwhile = await world.api("PUT", "/api/v1/nodes/roles", {
    basis,
    set: [{ nodeId: seedbox, roles: ["searching", "downloading", "converting"] }],
  });
  expect(meanwhile.status, "another tab gives seedbox Converting").toBe(200);

  await review(page).getByRole("button", { name: "Apply" }).click();
  await expect(
    review(page).getByText(
      "Roles changed since you opened this. Here is the change as it stands now.",
    ),
  ).toBeVisible();
  await expect(review(page).getByRole("list", { name: "What changes" })).toContainText(
    "seedbox stops Converting.",
  );
  await review(page).getByRole("button", { name: "Apply" }).click();
  await expect(review(page)).toHaveCount(0);
  expect(await assignedOf(world, "seedbox")).toEqual(["searching", "downloading", "subtitles"]);
});

test("Write off shows what seedbox holds first, then gives it up", async ({
  adminPage: page,
  world,
}) => {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    monitor: true,
  });
  const workId = Added.parse(added.body).work.id;
  expect((await world.api("POST", `/api/v1/works/${workId}/search`)).status).toBe(202);
  await expect
    .poll(async () => Grabs.parse((await world.api("GET", "/api/v1/grabs")).body).grabs[0]?.stage, {
      message: "seedbox imports The Quiet Orchard",
      timeout: 90_000,
    })
    .toBe("imported");

  await page.goto(`/nodes/${idOf(world, "seedbox")}/roles`);
  await expect(page.getByText(/^Holding 1 torrent \(/u)).toBeVisible();
  await page.getByRole("button", { name: "Write off what seedbox holds" }).click();
  const dialog = page.getByRole("dialog", { name: "Write off what seedbox holds" });
  await expect(dialog).toContainText(
    /seedbox holds 1 torrent \([\d.]+ [KMGT]?B\) for Downloading\./u,
  );
  await expect(dialog).toContainText("This also unticks Downloading on seedbox.");
  await dialog.getByRole("button", { name: "Write off" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(role(page, "Downloading")).not.toBeChecked();
  await expect(page.getByRole("button", { name: "Write off what seedbox holds" })).toHaveCount(0, {
    timeout: 60_000,
  });
});
