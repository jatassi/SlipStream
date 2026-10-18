import { expect, expectAccessible, test } from "./fixtures.ts";
import { declinePasskeyPrompt, submitPin } from "../harness/sign-in.ts";
import { signOut } from "./shell.ts";

test("sign-in returns to the page asked for and sign-out ends the session", async ({
  page,
  world,
}) => {
  const { email, pin } = world.admin();
  const signInHeading = page.getByRole("heading", { name: "Sign in to SlipStream" });
  await page.goto("/system");
  await expect(signInHeading).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/sign-in?redirect=%2Fsystem`);

  await submitPin(page, email, "1357");
  await expect(page.getByRole("alert")).toHaveText("The email or PIN is wrong");
  await expect(page.getByRole("textbox", { name: "PIN", exact: true })).toHaveValue("");
  await expectAccessible(page);

  await page.getByLabel("Email").clear();
  await declinePasskeyPrompt(page, await submitPin(page, `  ${email.toUpperCase()}  `, pin));
  await expect(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  expect(new URL(page.url()).pathname).toBe("/nodes");
  await expectAccessible(page);

  await page.goto("/");
  await signOut(page);
  await expect(page.getByRole("status").filter({ hasText: "You signed out." })).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/sign-in?reason=signed-out`);
  await page.goBack();
  await expect(signInHeading).toBeVisible();
  await page.reload();
  await expect(signInHeading).toBeVisible();
});

test("sign-in ignores a redirect to another site", async ({ page, world }) => {
  const { email, pin } = world.admin();
  await page.goto("/sign-in?redirect=//evil.example");
  await declinePasskeyPrompt(page, await submitPin(page, email, pin));
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(page.url()).toBe(`${world.baseURL}/library`);
});
