import type { BrowserContext, Page } from "@playwright/test";
import { z } from "zod";

import { offersPasskeys, signIn } from "../harness/sign-in.ts";
import { expect, test } from "./fixtures.ts";
import { signOut } from "./shell.ts";

test.use({ scenario: "home" });

const mailRecord = z.object({
  data: z.object({ to: z.array(z.string()), subject: z.string(), text: z.string() }),
});

const issued = z.object({ url: z.string() });

// A link is built on the Controller's name, which only Chromium resolves
// here, so a page opens it by its path and fragment on the world's base URL.
const pathOf = (link: string) => {
  const { pathname, hash } = new URL(link);
  return `${pathname}${hash}`;
};

async function enterPin(page: Page, label: string, pin: string) {
  const field = page.getByRole("textbox", { name: label, exact: true });
  await field.waitFor();
  const keypad = page.getByRole("group", { name: "Keypad" });
  if (await keypad.isVisible()) {
    for (const digit of pin) {
      // oxlint-disable-next-line no-await-in-loop -- each tap lands before the next, as a person's does
      await keypad.getByRole("button", { name: digit, exact: true }).click();
    }
    return;
  }
  await field.pressSequentially(pin);
}

async function newPin(page: Page, label: string, pin: string) {
  await enterPin(page, label, pin);
  await enterPin(page, "Type it again", pin);
}

async function openAccount(admin: Page, name: string) {
  await admin.goto("/accounts");
  await admin.getByRole("table", { name: "Accounts" }).getByRole("link", { name }).click();
  await expect(admin.getByRole("heading", { name, level: 1 })).toBeVisible();
}

async function addVirtualAuthenticator(context: BrowserContext, page: Page) {
  const cdp = await context.newCDPSession(page);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
}

async function issueSignupLink(admin: Page, email: string): Promise<string> {
  await admin.goto("/accounts");
  await admin.getByRole("button", { name: "New Signup link" }).click();
  const dialog = admin.getByRole("dialog", { name: "New Signup link" });
  await dialog.getByRole("textbox", { name: "Email" }).fill(email);
  await dialog.getByRole("button", { name: "Make the link" }).click();
  await expect(dialog.getByText(`Emailed it to ${email}.`)).toBeVisible();
  const link = await dialog.getByRole("textbox", { name: "Signup link" }).inputValue();
  await dialog.getByRole("button", { name: "Done" }).click();
  await expect(
    admin.getByRole("list", { name: "Pending Signup links" }).getByText(email),
  ).toBeVisible();
  return link;
}

async function join(page: Page, link: string, inviter: string, passkeyStep: "skip" | "add") {
  await page.goto(pathOf(link));
  await expect(
    page.getByText(`${inviter} invited you to ${new URL(link).hostname}.`),
  ).toBeVisible();
  await page.getByRole("button", { name: "Get started" }).click();
  await expect(page.getByText("You'll sign in as lin@example.test.")).toBeVisible();
  await page.getByLabel("Your name").fill("Lin Wei");
  await page.getByRole("button", { name: "Continue" }).click();
  await newPin(page, "PIN", "8642");
  if (await offersPasskeys(page)) {
    await page
      .getByRole("button", { name: passkeyStep === "add" ? "Add a passkey" : "Skip for now" })
      .click();
  }
  await expect(page.getByText("You're all set, Lin Wei.")).toBeVisible();
  await page.getByRole("button", { name: "Open SlipStream" }).click();
  await expect(page.getByRole("heading", { name: "Home" })).toBeVisible();
}

const lin = { email: "lin@example.test", pin: "8642", admin: false, displayName: "Lin Wei" };

test("a Signup link is emailed, and the person joins and signs in with a PIN", async ({
  adminPage,
  newContext,
  world,
}) => {
  const shown = await issueSignupLink(adminPage, "lin@example.test");
  expect(shown).toMatch(new RegExp(`^${world.node.browserUrl}/signup#token=[\\w-]+$`, "u"));
  const { data: mail } = mailRecord.parse(
    await world.fw.capture("mail").wait({ where: { to: '["lin@example.test"]' } }),
  );
  expect(mail.subject).toBe(`Ada Okafor invited you to ${new URL(world.node.browserUrl).hostname}`);
  expect(mail.text).toContain(shown);

  const page = await (await newContext()).newPage();
  await join(page, shown, "Ada Okafor", "skip");
  expect(page.url()).toBe(`${world.baseURL}/portal`);
  await page.goto(pathOf(shown));
  await expect(
    page.getByText("You're signed in as Lin Wei. Sign out to use this link."),
  ).toBeVisible();

  await page.goto("/portal");
  await signOut(page);
  await page.goto(pathOf(shown));
  await expect(
    page.getByText(
      "This link was already used, and each link works once, so sign in or ask Ada Okafor for a new one.",
    ),
  ).toBeVisible();
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(`${world.baseURL}/sign-in`);
  await signIn(page, lin);
});

test(
  "a passkey added while joining signs in from the email field's autofill, and a PIN sign-in elsewhere doesn't offer another",
  { tag: "@cdp" },
  async ({ newContext, world }) => {
    const reply = await world.api("POST", "/api/v1/signup-links", { email: "lin@example.test" });
    expect(reply.status).toBe(201);
    const { url } = issued.parse(reply.body);

    const context = await newContext();
    const page = await context.newPage();
    await addVirtualAuthenticator(context, page);
    await join(page, url, "Your Admin", "add");

    // The authenticator answers a conditional ceremony at once, as a person
    // picking the passkey from the email field's autofill does.
    const asserted = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        /\/api\/v1\/passkeys\/assertion\/[^/]+$/u.test(new URL(response.url()).pathname),
    );
    await signOut(page);
    expect((await asserted).status()).toBe(200);
    await expect(page.getByRole("heading", { name: "Home" })).toBeVisible();
    expect(page.url()).toBe(`${world.baseURL}/portal`);

    const elsewhere = await (await newContext()).newPage();
    await signIn(elsewhere, lin);
    await expect(
      elsewhere.getByRole("dialog", { name: "Sign in faster here with a passkey" }),
    ).toHaveCount(0);
  },
);

test("a Reset link signs the person out in their open tab and sets a new PIN", async ({
  adminPage,
  newContext,
  world,
}) => {
  const grace = world.member();
  const tab = await (await newContext()).newPage();
  await signIn(tab, grace);

  await openAccount(adminPage, grace.displayName);
  await adminPage.getByRole("button", { name: "Send a Reset link" }).click();
  const dialog = adminPage.getByRole("dialog", { name: `Send ${grace.displayName} a Reset link` });
  await dialog.getByRole("button", { name: "Send the Reset link" }).click();
  await expect(dialog.getByText(`Emailed it to ${grace.email}.`)).toBeVisible();
  const link = await dialog.getByRole("textbox", { name: "Reset link" }).inputValue();

  await expect(
    tab.getByRole("status").filter({ hasText: "This device was signed out." }),
  ).toBeVisible();
  expect(tab.url()).toBe(`${world.baseURL}/sign-in?reason=revoked`);

  await tab.goto(pathOf(link));
  await expect(tab.getByText(new RegExp(`for ${grace.email}`, "u"))).toBeVisible();
  await newPin(tab, "New PIN", "9753");
  if (await offersPasskeys(tab)) {
    await tab.getByRole("button", { name: "Skip for now" }).click();
  }
  await expect(tab.getByText("Your new PIN is set, and you're signed in.")).toBeVisible();
  await tab.getByRole("button", { name: "Open SlipStream" }).click();
  await expect(tab.getByRole("heading", { name: "Home" })).toBeVisible();

  const old = await world.api("POST", "/api/v1/session", { email: grace.email, pin: grace.pin });
  expect(old.status).toBe(401);
});

test("making an Admin a Member moves their open tab to the portal", async ({
  adminPage,
  newContext,
  world,
}) => {
  const [grace] = world.manifest.members;
  if (grace === undefined) {
    throw new Error("the home Scenario has no Member");
  }
  const promoted = await world.api("PATCH", `/api/v1/accounts/${grace.id}`, { admin: true });
  expect(promoted.status).toBe(200);
  const tab = await (await newContext()).newPage();
  await signIn(tab, { ...grace, admin: true });
  await tab.goto("/system");
  await expect(tab.getByRole("heading", { name: "Overview" })).toBeVisible();

  await openAccount(adminPage, grace.displayName);
  await adminPage.getByRole("button", { name: "Make Member" }).click();
  await adminPage
    .getByRole("dialog", { name: `Make ${grace.displayName} a Member?` })
    .getByRole("button", { name: "Make Member" })
    .click();

  await expect(tab).toHaveURL(`${world.baseURL}/portal`);
  await expect(tab.getByRole("heading", { name: "Home", exact: true })).toBeVisible();
});
