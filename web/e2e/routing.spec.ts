import { z } from "zod";
import { node } from "../harness/manifest.ts";
import { type World, expect, test } from "./fixtures.ts";
import { add, toasts } from "./library.ts";
import { isPhone } from "./shell.ts";
import { clickWhenStill } from "./still.ts";

test.use({ scenario: "seeding-routing" });
test.describe.configure({ timeout: 300_000 });

const Trackers = z.object({
  trackers: z.array(z.object({ id: z.string(), name: z.string(), privacy: z.string() })),
});
const Grabs = z.object({
  grabs: z.array(z.object({ workId: z.string(), stage: z.string(), nodeId: z.string() })),
});

async function heron(world: World) {
  const { trackers } = Trackers.parse((await world.api("GET", "/api/v1/trackers")).body);
  return trackers.find((tracker) => tracker.name === "Heron");
}

async function searched(world: World, key: string) {
  const workId = await add(world, key, "all");
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, `movie ${key}'s search starts`).toBe(202);
  return workId;
}

async function grabOf(world: World, workId: string) {
  const { grabs } = Grabs.parse((await world.api("GET", "/api/v1/grabs")).body);
  return grabs.find((grab) => grab.workId === workId);
}

test("a Grab routed to an offline Node waits in Needs you until the Admin sends it to another client", async ({
  adminPage: page,
  world,
}) => {
  const [seedbox, den, nas] = ["seedbox", "den", "nas"].map(
    (name) => node(world.manifest, name).nodeId,
  );
  await searched(world, "9000001");
  await expect
    .poll(async () => (await heron(world))?.privacy, {
      message: "Heron reads private once its first torrent is fetched",
      timeout: 90_000,
    })
    .toBe("private");
  const tracker = await heron(world);
  const routed = await world.api("PUT", `/api/v1/trackers/${tracker?.id}/routes/${seedbox}`, {
    builtInOn: nas,
  });
  expect(routed.status, "Heron's Grabs from seedbox go to nas").toBe(200);
  await world.fw.node("stop", "nas");

  const harbor = await searched(world, "9000002");
  await page.goto(`/works/${harbor}`);
  const needsYou = page.getByRole("region", { name: "Needs you" });
  const subject = "Waiting to send to the built-in client on nas";
  await expect(needsYou).toContainText(subject, { timeout: 60_000 });
  const wait = isPhone(page) ? page.getByRole("dialog", { name: subject }) : needsYou;
  if (isPhone(page)) {
    await needsYou.getByRole("button", { name: `Waiting ${subject}` }).click();
  }
  await expect(wait).toContainText(
    "nas is offline. This download goes there once nas is back online.",
  );
  await clickWhenStill(wait.getByRole("button", { name: "Send to another client" }));
  const sheet = page.getByRole("dialog", { name: "Send to another client" });
  await expect(sheet.getByRole("radio", { name: "Built-in client on nas" })).toHaveCount(0);
  await sheet.getByRole("radio", { name: "Built-in client on den" }).click();
  await sheet.getByRole("button", { name: "Send", exact: true }).click();
  await expect(toasts(page)).toContainText(
    "Sent Harbor.Lights.2019.1080p.WEB-DL.H264-FKW to the built-in client on den",
  );
  await expect(needsYou.getByText("Waiting to send")).toHaveCount(0);
  await expect
    .poll(async () => (await grabOf(world, harbor))?.nodeId, {
      message: "Harbor Lights downloads on den",
      timeout: 60_000,
    })
    .toBe(den);
});
