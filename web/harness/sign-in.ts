import type { Page, Response } from "playwright";
import { z } from "zod";

import type { Identity } from "./manifest.ts";

// An Admin lands on the Library and a Member on the portal's Home.
export async function signIn(page: Page, identity: Identity): Promise<void> {
  await page.goto("/sign-in");
  await declinePasskeyPrompt(page, await submitPin(page, identity.email, identity.pin));
  await page.getByRole("heading", { name: identity.admin ? "Library" : "Home" }).waitFor();
}

// The PIN field submits on its last digit.
export async function submitPin(page: Page, email: string, pin: string): Promise<Response> {
  await page.getByLabel("Email").fill(email);
  const answer = page.waitForResponse(
    (r) => r.request().method() === "POST" && new URL(r.url()).pathname === "/api/v1/session",
  );
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(pin);
  return answer;
}

const SignedIn = z.object({ passkeyPrompt: z.boolean() });

// The Node prompts only where the address can hold a passkey and the
// Account holds none there yet, and says so in the sign-in answer.
export async function declinePasskeyPrompt(page: Page, answer: Response): Promise<void> {
  if (!answer.ok() || !SignedIn.parse(await answer.json()).passkeyPrompt) {
    return;
  }
  const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" });
  await prompt.getByRole("button", { name: "Not now" }).click();
  await prompt.waitFor({ state: "hidden" });
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
