import type { Page } from "@playwright/test";
import { z } from "zod";

import { offersPasskeys } from "../harness/sign-in.ts";
import { type World, expect, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "media-servers", allow: ["plextv-auth"] });
test.describe.configure({ timeout: 120_000 });

const noor = { email: "noor@example.test", name: "Noor Haddad", plex: "noor", pin: "8642" };
const plex = "Living room Plex";
const jellyfin = "Den Jellyfin";

const Issued = z.object({ url: z.string() });
const Access = z.object({
  rows: z.array(
    z.object({
      state: z.string(),
      member: z.object({ name: z.string() }),
      server: z.object({ name: z.string() }),
    }),
  ),
});
const Mail = z.object({ seq: z.int(), data: z.object({ to: z.array(z.string()) }) });
const Users = z.object({
  data: z.object({ users: z.array(z.object({ name: z.string(), disabled: z.boolean() })) }),
});

async function accessOf(world: World): Promise<Record<string, string>> {
  const { rows } = Access.parse((await world.api("GET", "/api/v1/media-servers/access")).body);
  return Object.fromEntries(
    rows.filter((row) => row.member.name === noor.name).map((row) => [row.server.name, row.state]),
  );
}

async function issueLink(admin: Page): Promise<string> {
  await admin.goto("/accounts");
  await admin.getByRole("button", { name: "New Signup link" }).click();
  const dialog = admin.getByRole("dialog", { name: "New Signup link" });
  await dialog.getByRole("textbox", { name: "Email" }).fill(noor.email);
  const servers = dialog.getByRole("group", { name: "Media Servers" });
  await servers.getByRole("checkbox", { name: "Den Emby" }).setChecked(false);
  await servers.getByRole("checkbox", { name: plex }).setChecked(true);
  await servers.getByRole("checkbox", { name: jellyfin }).setChecked(true);
  await expect(servers.getByRole("checkbox", { checked: true })).toHaveCount(2);
  const issued = admin.waitForResponse(
    (r) => r.request().method() === "POST" && new URL(r.url()).pathname === "/api/v1/signup-links",
  );
  await dialog.getByRole("button", { name: "Make the link" }).click();
  const { url } = Issued.parse(await (await issued).json());
  await dialog.getByRole("button", { name: "Done" }).click();
  return url;
}

async function enterPin(page: Page, label: string): Promise<void> {
  const field = page.getByRole("textbox", { name: label, exact: true });
  await field.waitFor();
  const keypad = page.getByRole("group", { name: "Keypad" });
  if (await keypad.isVisible()) {
    for (const digit of noor.pin) {
      // oxlint-disable-next-line no-await-in-loop -- each tap lands before the next, as a person's does
      await keypad.getByRole("button", { name: digit, exact: true }).click();
    }
    return;
  }
  await field.pressSequentially(noor.pin);
}

// Joins by the link and goes on to the Watch step.
async function join(page: Page, link: string): Promise<void> {
  const { pathname, hash } = new URL(link);
  await page.goto(`${pathname}${hash}`);
  await page.getByRole("button", { name: "Get started" }).click();
  await page.getByLabel("Your name").fill(noor.name);
  await page.getByRole("button", { name: "Continue" }).click();
  await enterPin(page, "PIN");
  await enterPin(page, "Type it again");
  await page.waitForURL(/\/welcome\//u);
  if (await offersPasskeys(page)) {
    await page.getByRole("button", { name: "Skip for now" }).click();
  }
  await expect(page).toHaveURL(/\/welcome\/watch$/u);
}

const section = (page: Page, name: string) => page.getByRole("region", { name });

// Signs in on plex.tv's page as noor: in a new tab on a wide screen, in
// this tab on a phone. plex.tv sends either back to the Watch step.
async function signInToPlex(page: Page): Promise<void> {
  const opened = isPhone(page) ? undefined : page.context().waitForEvent("page");
  await section(page, plex).getByRole("button", { name: "Continue with Plex" }).click();
  const auth = opened === undefined ? page : await opened;
  await auth.getByRole("combobox", { name: "Plex account" }).selectOption(noor.plex);
  await auth.getByRole("button", { name: "Allow" }).click();
  await expect(auth).toHaveURL(/\/welcome\/watch$/u);
  if (auth !== page) {
    await auth.close();
  }
}

test(
  "a Member offered Plex and Jellyfin by a Signup link signs in to Plex, makes a Jellyfin account, is shared on Plex, and sees how to watch both",
  { tag: "@named-origin" },
  async ({ adminPage, newContext, world }) => {
    const mail = world.fw.capture("mail");
    const link = await issueLink(adminPage);
    const invite = Mail.parse(await mail.wait({ where: { to: JSON.stringify([noor.email]) } }));

    const page = await (await newContext()).newPage();
    await join(page, link);
    await expect.poll(() => accessOf(world)).toEqual({ [plex]: "offered", [jellyfin]: "offered" });

    await signInToPlex(page);
    await expect.poll(() => accessOf(world)).toMatchObject({ [plex]: "waiting" });
    const admin = world.admin();
    await mail.wait({ where: { to: JSON.stringify([admin.email]) }, after: invite.seq });

    const users = section(page, jellyfin);
    await users.getByRole("textbox", { name: "Username" }).fill(noor.plex);
    await users.getByLabel("Password").fill("orchard-lantern");
    await users.getByRole("button", { name: "Make my Jellyfin account" }).click();
    await expect(users.getByRole("list", { name: `How to watch ${jellyfin}` })).toBeVisible();
    const { data } = Users.parse((await world.fw.capture("jellyfin-users").list()).at(-1));
    expect(data.users).toContainEqual(
      expect.objectContaining({ name: noor.plex, disabled: false }),
    );

    await adminPage.goto("/accounts");
    const waiting = adminPage
      .getByRole("list", { name: "Needs you" })
      .getByRole("listitem")
      .filter({ hasText: noor.name });
    await waiting.getByRole("button", { name: "Share on Plex" }).click();
    await expect(waiting).toHaveCount(0);
    await mail.wait({ where: { to: JSON.stringify([noor.email]) }, after: invite.seq });
    await expect.poll(() => accessOf(world)).toEqual({ [plex]: "active", [jellyfin]: "active" });

    const howTo = section(page, plex).getByRole("list", { name: `How to watch ${plex}` });
    await expect(howTo).toBeVisible();
    await expect(howTo).toContainText(noor.plex);
    await page.getByRole("button", { name: "Continue" }).click();
    await page.getByRole("button", { name: "Open SlipStream" }).click();
    await expect(page.getByRole("heading", { name: "Home" })).toBeVisible();
  },
);
