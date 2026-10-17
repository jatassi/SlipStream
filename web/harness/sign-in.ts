import type { Page } from "playwright";
import { z } from "zod";

import type { Identity } from "./manifest.ts";

// An Admin lands on the Library and a Member on the portal's Home.
export async function signIn(page: Page, identity: Identity): Promise<void> {
  await page.goto("/sign-in");
  await page.getByLabel("Email").fill(identity.email);
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(identity.pin);
  await declinePasskeyPrompt(page);
  await page.getByRole("heading", { name: identity.admin ? "Library" : "Home" }).waitFor();
}

const Status = z.object({ passkeys: z.object({ available: z.boolean() }) });

// The Node answers for the address the request reached it at. The fetch
// runs in the page because only the browser applies the host-resolver rules
// that map the Node's name, and Firefox and WebKit reach it by IP address,
// where it offers no passkey.
export async function offersPasskeys(page: Page): Promise<boolean> {
  const body = await page.evaluate(async () => (await fetch("/api/v1/status")).text());
  return Status.parse(JSON.parse(body)).passkeys.available;
}

// A Node that can hold a passkey offers one after each PIN sign-in, and the
// browsers driven here hold no authenticator.
export async function declinePasskeyPrompt(page: Page): Promise<void> {
  if (!(await offersPasskeys(page))) {
    return;
  }
  const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" });
  await prompt.getByRole("button", { name: "Not now" }).click();
  await prompt.waitFor({ state: "hidden" });
}
