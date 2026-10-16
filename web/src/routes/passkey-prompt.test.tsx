import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentitySignedIn } from "../api/gen/model";
import { passkeysAt, passkeysHere, recorder, stubPasskeys } from "../test/account";
import { ada, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";

const answer: IdentitySignedIn = {
  account: ada,
  pinRequired: false,
  keyPrompt: "ask",
  passkeyPrompt: true,
};

function promptingServer() {
  let signedIn = false;
  return [
    passkeysAt(passkeysHere),
    http.get("*/api/v1/session", () =>
      signedIn ? HttpResponse.json(answer) : problem(401, "signed_out", "Sign in to continue"),
    ),
    http.post("*/api/v1/session", () => {
      signedIn = true;
      return HttpResponse.json(answer);
    }),
  ];
}

const prompt = () => page.getByRole("dialog", { name: "Sign in faster here with a passkey" });

async function signIn() {
  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("1234");
}

test("a sign-in that asks for the prompt offers a passkey, and Not now closes it", async () => {
  stubPasskeys();
  renderApp({ path: "/sign-in", handlers: promptingServer() });
  await signIn();
  await expect.element(prompt()).toBeVisible();
  await expect
    .element(
      prompt().getByText(
        "Sign in at localhost with Face ID, Touch ID or your device's screen lock instead of your PIN.",
      ),
    )
    .toBeVisible();
  await userEvent.click(prompt().getByRole("button", { name: "Not now" }));
  await expect.element(prompt()).not.toBeInTheDocument();
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
});

test("Don't ask again turns the prompt off for the Account", async () => {
  stubPasskeys();
  const { seen, record } = recorder();
  renderApp({
    path: "/sign-in",
    handlers: [
      ...promptingServer(),
      http.patch("*/api/v1/account", async ({ request }) => {
        await record(request);
        return HttpResponse.json({ account: ada, pinRequired: false, keyPrompt: "never" });
      }),
    ],
  });
  await signIn();
  await userEvent.click(prompt().getByRole("button", { name: "Don't ask again" }));
  await expect.element(prompt()).not.toBeInTheDocument();
  expect(seen).toEqual([{ method: "PATCH", url: "/api/v1/account", body: { keyPrompt: "never" } }]);
});
