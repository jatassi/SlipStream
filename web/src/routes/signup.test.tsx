import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { RedeemSignupLinkBody } from "../api/gen/zod/identity/identity.zod";
import { grace, problem } from "../test/handlers";
import { graceSignedIn, linkServer, liveLink } from "../test/links";
import { mediaServerAccessHandlers, myAccess } from "../test/media-server-access";
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
    signIn: server.signIn,
    handlers: [
      ...server.handlers,
      ...mediaServerAccessHandlers({ ...myAccess, rows: [] }),
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

test("at an address that can't hold a passkey, signup counts on into /welcome without the passkey step", async () => {
  const { router } = renderApp({ path: "/signup#token=tok-1", handlers: joinServer().handlers });
  await expect.element(page.getByText("Step 1 of 4")).toBeVisible();
  await nameAndPin();
  await expect
    .element(page.getByText(`Your Account is ready, ${grace.displayName}.`))
    .toBeVisible();
  await expect.element(page.getByText("Step 4 of 4")).toBeVisible();
  expect(router.state.location.pathname).toBe("/welcome/done");
});

test("/welcome opened outside signup moves to a step that shows, with no count", async () => {
  const server = joinServer();
  server.signIn();
  const { router } = renderApp({ path: "/welcome/passkey", handlers: server.handlers });
  await expect
    .element(page.getByText(`Your Account is ready, ${grace.displayName}.`))
    .toBeVisible();
  expect(router.state.location.pathname).toBe("/welcome/done");
  await expect.element(page.getByText(/^Step \d+ of/u)).not.toBeInTheDocument();
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
      ...mediaServerAccessHandlers({ ...myAccess, rows: [] }),
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
