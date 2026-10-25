import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { RedeemSignupLinkBody } from "../api/gen/zod/identity/identity.zod";
import { passkeysAt, passkeysHere, stubPasskeys } from "../test/account";
import { problem } from "../test/handlers";
import { graceSignedIn, linkServer, liveLink } from "../test/links";
import { renderApp } from "../test/render-app";

async function typePin(label: string, pin: string) {
  await userEvent.click(page.getByRole("textbox", { name: label, exact: true }));
  await userEvent.keyboard(pin);
}

function joinServer() {
  const server = linkServer("signup-links", [liveLink()]);
  const joined: unknown[] = [];
  return {
    joined,
    handlers: [
      ...server.handlers,
      http.post("*/api/v1/signup-links/by-token/tok-1/account", async ({ request }) => {
        joined.push(RedeemSignupLinkBody.parse(await request.json()));
        server.signIn();
        return HttpResponse.json(graceSignedIn);
      }),
    ],
  };
}

async function nameAndPin() {
  await expect.element(page.getByText("Ada Okafor invited you to forge.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Get started" }));
  await expect.element(page.getByText("You'll sign in as grace@example.test.")).toBeVisible();
  await userEvent.fill(page.getByLabelText("Your name"), "Grace");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await typePin("PIN", "2468");
  await typePin("Type it again", "2468");
}

test("a Signup link welcomes, takes a name and a PIN, offers a passkey and lands in the portal", async () => {
  stubPasskeys();
  const server = joinServer();
  const { router } = renderApp({
    path: "/signup#token=tok-1",
    handlers: [passkeysAt(passkeysHere), ...server.handlers],
  });
  await expect.element(page.getByText("Step 1 of 5")).toBeVisible();
  expect(router.state.location.hash).toBe("");
  await nameAndPin();

  await expect.element(page.getByRole("button", { name: "Add a passkey" })).toBeVisible();
  expect(server.joined).toEqual([{ displayName: "Grace", pin: "2468" }]);
  await userEvent.click(page.getByRole("button", { name: "Skip for now" }));
  await expect.element(page.getByText("Your Account is ready, Grace.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Open SlipStream" }));
  await expect.poll(() => router.state.location.href).toBe("/portal");
});

test("at an address that can't hold a passkey, signup leaves the passkey step out", async () => {
  renderApp({ path: "/signup#token=tok-1", handlers: joinServer().handlers });
  await expect.element(page.getByText("Step 1 of 4")).toBeVisible();
  await nameAndPin();
  await expect.element(page.getByText("Your Account is ready, Grace.")).toBeVisible();
});

test("a link used while the page was open shows its card", async () => {
  const server = linkServer("signup-links", [
    liveLink(),
    {
      state: "dead",
      reason: "used",
      adminName: "Ada Okafor",
      issuedAt: "2026-10-16T10:00:00Z",
      expiresAt: "2026-10-23T10:00:00Z",
    },
  ]);
  const { router } = renderApp({
    path: "/signup#token=tok-1",
    handlers: [
      ...server.handlers,
      http.post("*/api/v1/signup-links/by-token/tok-1/account", () =>
        problem(410, "link_dead", "This link no longer works"),
      ),
    ],
  });
  await nameAndPin();
  await expect
    .element(
      page.getByText(
        "This link was already used, and each link works only once. Sign in, or ask Ada Okafor for a new one.",
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Sign in" }));
  await expect.poll(() => router.state.location.pathname).toBe("/sign-in");
});

test("someone already signed in is asked to sign out first, and then sees the welcome", async () => {
  const server = linkServer("signup-links", [liveLink()]);
  let signedOut = false;
  renderApp({
    path: "/signup#token=tok-1",
    handlers: [
      ...server.handlers.slice(1),
      http.delete("*/api/v1/session", () => {
        signedOut = true;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await expect
    .element(page.getByText("You're signed in as Ada Okafor. Sign out to use this link."))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Sign out" }));
  await userEvent.click(
    page
      .getByRole("dialog", { name: "Sign out of this device?" })
      .getByRole("button", { name: "Sign out" }),
  );
  await expect.element(page.getByText("Ada Okafor invited you to forge.")).toBeVisible();
  expect(signedOut).toBe(true);
});
