import { z } from "zod";

import { expect, test } from "./fixtures.ts";
import { nodeId } from "./grabs.ts";

test.use({ scenario: "conversion" });

const Converting = z.object({
  hours: z.object({ on: z.boolean() }),
  proofs: z.array(z.object({ path: z.string(), provenAt: z.string().nullable() })),
});
const Servers = z.object({
  servers: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      pausesConversions: z.boolean(),
    }),
  ),
});

test("a Node's Converting section saves its hours switch and lists its test runs", async ({
  adminPage: page,
  world,
}) => {
  const home = await nodeId(world, "home");
  const read = async () =>
    Converting.parse((await world.api("GET", `/api/v1/nodes/${home}/converting`)).body);
  await expect
    .poll(async () => (await read()).proofs.some((proof) => proof.provenAt !== null), {
      message: "home's test run on its processor passes",
      timeout: 60_000,
    })
    .toBe(true);

  await page.goto(`/nodes/${home}/converting`);
  await expect(page.getByText("home converts with its processor.")).toBeVisible();
  await page.getByRole("button", { name: "Test runs", exact: true }).click();
  await expect(page.getByRole("main").getByRole("listitem").first()).toBeVisible();

  await page.getByRole("switch", { name: "Only start Conversions in set hours" }).click();
  await expect(page.getByText("Start Conversions between")).toBeVisible();
  await expect.poll(async () => (await read()).hours.on).toBe(true);
});

test("a Media Server that runs on a Node saves its pause switch", async ({
  adminPage: page,
  world,
}) => {
  const servers = async () =>
    Servers.parse((await world.api("GET", "/api/v1/media-servers")).body).servers;
  const [plex] = await servers();
  expect(plex?.pausesConversions, "a server pauses Conversions by default").toBe(true);

  await page.goto(`/settings/media-servers/${plex?.id ?? ""}/settings`);
  const pause = page.getByRole("switch", {
    name: "Pause Conversions while this server transcodes",
  });
  await expect(pause).toBeChecked();
  await pause.click();
  await expect(pause).not.toBeChecked();
  await expect.poll(async () => (await servers())[0]?.pausesConversions).toBe(false);
});
