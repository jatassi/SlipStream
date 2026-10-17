import { isIP } from "node:net";

import type { Page } from "playwright";

import type { Identity } from "./manifest.ts";

// An Admin lands on the Library and a Member on the portal's Home.
export async function signIn(page: Page, identity: Identity): Promise<void> {
  await page.goto("/sign-in");
  await page.getByLabel("Email").fill(identity.email);
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(identity.pin);
  await declinePasskeyPrompt(page);
  await page.getByRole("heading", { name: identity.admin ? "Library" : "Home" }).waitFor();
}

// A Node reached by name can hold a passkey. One reached by IP address,
// as Firefox and WebKit reach it, can't, and never offers one.
export const offersPasskeys = (page: Page): boolean => isIP(new URL(page.url()).hostname) === 0;

// A Node that can hold a passkey offers one after each PIN sign-in, and the
// browsers driven here hold no authenticator.
export async function declinePasskeyPrompt(page: Page): Promise<void> {
  if (!offersPasskeys(page)) {
    return;
  }
  const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" });
  await prompt.getByRole("button", { name: "Not now" }).click();
  await prompt.waitFor({ state: "hidden" });
}
