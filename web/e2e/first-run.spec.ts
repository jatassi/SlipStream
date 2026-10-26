import type { BrowserContext, Page } from "@playwright/test";
import { z } from "zod";

import { node } from "../harness/manifest.ts";
import { type World, expect, expectAccessible, test } from "./fixtures.ts";

test.use({ scenario: "first-run" });
test.describe.configure({ timeout: 180_000 });

const staleLink =
  "This setup link is from an earlier start of this Node. Read the new Setup code from its log.";
const workerHeading = "This Node is a Worker. Manage it from its Controller.";

const setupCode = (page: Page) => page.getByLabel("Setup code");
const choiceList = (page: Page) => page.getByRole("list", { name: "What this Node becomes" });
const startNew = (page: Page) => page.getByRole("button", { name: "Start a new SlipStream" });
const joinExisting = (page: Page) =>
  page.getByRole("button", { name: "Join an existing SlipStream" });
const addANode = (page: Page) => page.getByRole("dialog", { name: "Add a Node" });

// The page makes the call, because only the browser applies the host
// resolver rules that map a Node's name.
function call(
  page: Page,
  method: string,
  path: `/api/v1/${string}`,
): Promise<{ status: number; body: unknown }> {
  return page.evaluate(
    async ([m, p]) => {
      const response = await fetch(p, { method: m });
      return {
        status: response.status,
        body: response.status === 204 ? null : ((await response.json()) as unknown),
      };
    },
    [method, path] as const,
  );
}

async function enterCode(page: Page, code: string): Promise<void> {
  await setupCode(page).pressSequentially(code);
  await page.getByRole("button", { name: "Continue" }).click();
}

async function startNewSlipStream(page: Page): Promise<void> {
  await startNew(page).click();
  await page.getByLabel("Email").fill("grace@example.test");
  await page.getByLabel("Display name").fill("Grace Hopper");
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially("1357");
  await page.getByRole("textbox", { name: "PIN again", exact: true }).pressSequentially("1357");
  await page.getByRole("button", { name: "Create Admin" }).click();
  await expect(page).not.toHaveURL(/\/setup$/u);
}

// The Join tests finish first setup by API, because its guided steps have
// their own spec.
async function claimHome(page: Page, world: World): Promise<void> {
  await page.goto("/setup");
  await enterCode(page, await world.setupCode("home"));
  await startNewSlipStream(page);
  expect((await call(page, "POST", "/api/v1/first-run/finish")).status).toBe(204);
  await page.goto("/nodes");
}

async function mintJoinCode(page: Page): Promise<string> {
  await page.getByRole("button", { name: "Add a Node", exact: true }).click();
  const code = addANode(page).getByRole("textbox", { name: "Join Code" });
  await expect(code).toHaveValue(/^SLP1-/u);
  return code.inputValue();
}

async function openDen(
  newContext: (at?: string) => Promise<BrowserContext>,
  world: World,
): Promise<Page> {
  const den = await (await newContext("den")).newPage();
  await den.goto("/setup");
  await enterCode(den, await world.setupCode("den"));
  return den;
}

async function joinWith(den: Page, code: string): Promise<void> {
  await joinExisting(den).click();
  await den.getByLabel("Join Code").fill(code);
  await den.getByRole("button", { name: "Join", exact: true }).click();
}

async function expectWorkerOfHome(den: Page): Promise<void> {
  await expect(den.getByRole("heading", { name: workerHeading })).toBeVisible();
  await expect(den.getByText("home", { exact: true })).toBeVisible();
  await expect(den).toHaveURL(/\/worker$/u);
}

test("a wrong Setup code fails on its own, before any other field", async ({ page, world }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Set up SlipStream" })).toBeVisible();
  await expectAccessible(page);

  await enterCode(page, "0000 0000 0000");
  await expect(setupCode(page)).toHaveAccessibleDescription("The Setup code is wrong");
  await expect(page.getByLabel("Email")).toHaveCount(0);

  await setupCode(page).clear();
  await enterCode(page, (await world.setupCode()).toLowerCase().replaceAll("-", " "));
  await expect(startNew(page)).toBeVisible();
  await expect(choiceList(page).getByRole("button").first()).toHaveText("Start a new SlipStream");
  await expectAccessible(page);
});

test("a setup link fills in and checks the code, and drops it from the address bar", async ({
  page,
  world,
}) => {
  await page.goto(`/setup#code=${await world.setupCode()}`);
  await expect(page.getByText("Setup code filled in")).toBeVisible();
  await expect(startNew(page)).toBeVisible();
  await expect.poll(() => page.url()).toBe(`${world.baseURL}/setup`);
});

test("a setup link from an earlier start of the Node says to read the new code", async ({
  page,
}) => {
  await page.goto("/setup#code=0000-0000-0000");
  await expect(page.getByText(staleLink)).toBeVisible();
  await expect(setupCode(page)).toHaveValue("");
  await expect.poll(() => page.url()).toMatch(/\/setup$/u);
});

test("a reload keeps the code this tab already entered", async ({ page, world }) => {
  await page.goto("/setup");
  await enterCode(page, await world.setupCode());
  await expect(startNew(page)).toBeVisible();
  await page.reload();
  await expect(startNew(page)).toBeVisible();
  await expect(setupCode(page)).toHaveCount(0);
});

test("a second setup link opened in the same tab fills in", async ({ page, world }) => {
  await page.goto("/setup");
  await expect(setupCode(page)).toBeVisible();
  const code = await world.setupCode();
  await page.evaluate((linked) => {
    window.location.hash = `code=${linked}`;
  }, code);
  await expect(page.getByText("Setup code filled in")).toBeVisible();
  await expect.poll(() => page.url()).toBe(`${world.baseURL}/setup`);
});

test("Start a new SlipStream claims the Node and leaves the first-run page", async ({
  page,
  world,
}) => {
  await page.goto("/setup");
  await enterCode(page, await world.setupCode());
  await startNewSlipStream(page);
  expect((await world.api("GET", "/api/v1/status")).body).toMatchObject({ claim: "controller" });
  await page.goto("/setup");
  await expect(page).not.toHaveURL(/\/setup$/u);
});

test("den joins home with a Join Code from Add a Node", async ({ page, world, newContext }) => {
  await claimHome(page, world);
  const code = await mintJoinCode(page);
  const den = await openDen(newContext, world);
  await joinWith(den, code);
  await expectWorkerOfHome(den);
  await expect(addANode(page).getByText("den paired.")).toBeVisible();
});

test("den that can't reach home waits on its page until home dials the pasted Node address", async ({
  page,
  world,
  newContext,
}) => {
  await world.fw.fault.set({ id: "home-behind-cgnat", target: "link/home", fault: "refuse" });
  await claimHome(page, world);
  const code = await mintJoinCode(page);
  const den = await openDen(newContext, world);
  await joinWith(den, code);

  const nodeAddress = den.getByRole("textbox", { name: "Node address" });
  await expect(nodeAddress).toHaveValue(/^SLN1-/u);
  await expect(nodeAddress).toHaveAccessibleDescription(/^Expires in \d+:\d\d(:\d\d)?\.$/u);
  await expect(
    den.getByText("Paste this Node address into Add a Node on your SlipStream."),
  ).toBeVisible();
  await expectAccessible(den);
  const address = await nodeAddress.inputValue();

  await den.reload();
  await expect(nodeAddress).toHaveValue(address);

  await addANode(page).getByRole("button", { name: "The new Node can't reach home?" }).click();
  await addANode(page).getByRole("textbox", { name: "Node address" }).fill(address);
  await addANode(page).getByRole("button", { name: "home dials it" }).click();
  await expect(addANode(page).getByText("den paired.")).toBeVisible();
  await expectWorkerOfHome(den);
});

const Nodes = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });
const Status = z.object({
  removed: z.object({ controller: z.string(), at: z.string() }).optional(),
});

test("a Node removed from home says so and leads with Join", async ({
  page,
  world,
  newContext,
}) => {
  await claimHome(page, world);
  const code = await mintJoinCode(page);
  const den = await openDen(newContext, world);
  await joinWith(den, code);
  await expectWorkerOfHome(den);
  const firstCode = await world.setupCode("den");

  const { nodes } = Nodes.parse((await call(page, "GET", "/api/v1/nodes")).body);
  const id = nodes.find(({ name }) => name === "den")?.id;
  expect(id).toBeDefined();
  expect((await call(page, "DELETE", `/api/v1/nodes/${id ?? ""}`)).status).toBe(204);

  // den restarts Unclaimed, so its status is read until it responds again.
  const denStatus = new URL("/api/v1/status", node(world.manifest, "den").url);
  await expect
    .poll(
      () =>
        fetch(denStatus)
          .then(async (response) => Status.parse(await response.json()).removed)
          .catch(() => null),
      { timeout: 60_000 },
    )
    .toEqual({ controller: "home", at: expect.any(String) });
  await expect.poll(() => world.setupCode("den"), { timeout: 30_000 }).not.toBe(firstCode);

  await den.goto("/setup");
  const removed = den.getByText(
    /^This Node was removed from home on .+\. Join with a new Join Code to use it again\.$/u,
  );
  await expect(removed).toBeVisible();
  await enterCode(den, await world.setupCode("den"));
  await expect(removed).toBeVisible();
  await expect(choiceList(den).getByRole("button").first()).toHaveText(
    "Join an existing SlipStream",
  );
});
