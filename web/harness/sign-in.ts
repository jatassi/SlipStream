import type { Page } from "playwright";

export type Identity = { email: string; pin: string };

export async function signIn(page: Page, identity: Identity): Promise<void> {
  await page.goto("/sign-in");
  await page.getByLabel("Email").fill(identity.email);
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(identity.pin);
  await declinePasskeyPrompt(page);
  await page.getByRole("heading", { name: "Library" }).waitFor();
}

// A Node reached by name offers a passkey after each PIN sign-in, and the
// browsers driven here hold no authenticator.
export async function declinePasskeyPrompt(page: Page): Promise<void> {
  const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" });
  await prompt.getByRole("button", { name: "Not now" }).click();
  await prompt.waitFor({ state: "hidden" });
}
