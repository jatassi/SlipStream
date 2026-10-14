import type { Page } from "playwright";

export type Identity = { email: string; pin: string };

export async function signIn(page: Page, identity: Identity): Promise<void> {
  await page.goto("/sign-in");
  await page.getByLabel("Email").fill(identity.email);
  await page.getByRole("textbox", { name: "PIN", exact: true }).pressSequentially(identity.pin);
  await page.getByRole("heading", { name: "Library" }).waitFor();
}
