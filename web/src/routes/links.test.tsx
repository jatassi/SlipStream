import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentityLinkView } from "../api/gen/model";
import { clock } from "../format";
import { iPhonePasskey, passkeysAt, passkeysHere, recorder, stubPasskeys } from "../test/account";
import { problem } from "../test/handlers";
import { graceSignedIn, linkServer, liveLink } from "../test/links";
import { renderApp } from "../test/render-app";

const sent = "2026-10-01T10:00:00Z";
const expired = "2026-10-08T10:00:00Z";

test.each<[string, string, IdentityLinkView, string]>([
  [
    "an expired Signup link",
    "/signup#token=tok-1",
    {
      state: "dead",
      reason: "expired",
      adminName: "Ada Okafor",
      issuedAt: sent,
      expiresAt: expired,
    },
    `This link expired on ${clock(expired)}, 7 days after it was sent, so ask Ada Okafor for a new one.`,
  ],
  [
    "an expired Reset link the host command printed",
    "/reset#token=tok-1",
    { state: "dead", reason: "expired", issuedAt: sent, expiresAt: "2026-10-02T10:00:00Z" },
    `This Reset link expired on ${clock("2026-10-02T10:00:00Z")}, 24 hours after it was sent, so ask your Admin for a new one.`,
  ],
  [
    "a withdrawn confirm link",
    "/confirm-email#token=tok-1",
    {
      state: "dead",
      reason: "withdrawn",
      adminName: "Ada Okafor",
      issuedAt: sent,
      expiresAt: expired,
    },
    "This confirm link was replaced or cancelled, so ask Ada Okafor whether a new one is on its way.",
  ],
  [
    "a made-up link",
    "/signup#token=nope",
    { state: "dead", reason: "unknown" },
    "This link does not work, so check that you have all of it or ask the person who sent it for a new one.",
  ],
  [
    "a link with no token",
    "/reset",
    { state: "dead", reason: "unknown" },
    "This Reset link does not work, so check that you have all of it or ask the person who sent it for a new one.",
  ],
])("%s says why in one sentence", async (_case, path, view, sentence) => {
  const kind = path.startsWith("/signup")
    ? "signup-links"
    : path.startsWith("/reset")
      ? "reset-links"
      : "email-links";
  renderApp({ path, handlers: linkServer(kind, [view]).handlers });
  await expect.element(page.getByText(sentence)).toBeVisible();
});

test("a link the server didn't answer for offers Try again", async () => {
  let answer = false;
  renderApp({
    path: "/signup#token=tok-1",
    handlers: [
      ...linkServer("signup-links", [liveLink()]).handlers.slice(0, 1),
      http.get("*/api/v1/signup-links/by-token/tok-1", () =>
        answer ? HttpResponse.json(liveLink()) : HttpResponse.error(),
      ),
    ],
  });
  await expect
    .element(page.getByText("The server did not answer, so check your connection and try again."))
    .toBeVisible();
  answer = true;
  await userEvent.click(page.getByRole("button", { name: "Try again" }));
  await expect.element(page.getByText("Ada Okafor invited you to forge.")).toBeVisible();
});

test("a second link opened in the same tab replaces the first", async () => {
  const { router } = renderApp({
    path: "/signup#token=tok-1",
    handlers: linkServer("signup-links", [liveLink()]).handlers,
  });
  await expect.element(page.getByText("Ada Okafor invited you to forge.")).toBeVisible();
  router.history.push("/signup#token=nope");
  await expect
    .element(
      page.getByText(
        "This link does not work, so check that you have all of it or ask the person who sent it for a new one.",
      ),
    )
    .toBeVisible();
});

async function typePin(label: string, pin: string) {
  await userEvent.click(page.getByRole("textbox", { name: label, exact: true }));
  await userEvent.keyboard(pin);
}

test("a Reset link sets a new PIN, shows the passkeys to delete, offers a new one and signs in", async () => {
  stubPasskeys();
  const server = linkServer("reset-links", [liveLink()]);
  const { seen, record } = recorder();
  const elsewhere = {
    ...iPhonePasskey,
    id: "p-old",
    name: "YubiKey",
    host: "slipstream.example.com",
  };
  const { router } = renderApp({
    path: "/reset#token=tok-1",
    handlers: [
      passkeysAt(passkeysHere),
      ...server.handlers,
      http.post("*/api/v1/reset-links/by-token/tok-1/pin", async ({ request }) => {
        await record(request);
        server.signIn();
        return HttpResponse.json(graceSignedIn);
      }),
      http.get("*/api/v1/passkeys", () =>
        HttpResponse.json({ passkeys: [iPhonePasskey, elsewhere] }),
      ),
      http.delete("*/api/v1/passkeys/p-iphone", async ({ request }) => {
        await record(request);
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await expect.element(page.getByText(/for grace@example\.test/u)).toBeVisible();
  await typePin("New PIN", "1357");
  await typePin("Type it again", "1357");

  await expect
    .element(page.getByRole("list", { name: "Passkeys" }).getByText("YubiKey"))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Delete iCloud Keychain" }));
  await expect.element(page.getByText("iCloud Keychain")).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await userEvent.click(page.getByRole("button", { name: "Skip for now" }));
  await expect.element(page.getByText("Your new PIN is set, and you're signed in.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Open SlipStream" }));
  await expect.poll(() => router.state.location.href).toBe("/portal");
  expect(seen).toEqual([
    { method: "POST", url: "/api/v1/reset-links/by-token/tok-1/pin", body: { pin: "1357" } },
    { method: "DELETE", url: "/api/v1/passkeys/p-iphone", body: null },
  ]);
});

test("a confirm link changes the email and says so", async () => {
  const seen: string[] = [];
  renderApp({
    path: "/confirm-email#token=tok-1",
    handlers: [
      ...linkServer("email-links", [liveLink("grace@lindqvist.example")]).handlers,
      http.post("*/api/v1/email-links/by-token/:token/confirm", ({ params }) => {
        seen.push(String(params["token"]));
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Confirm" }));
  await expect
    .element(page.getByRole("status"))
    .toHaveTextContent("Your SlipStream email is now grace@lindqvist.example.");
  expect(seen).toEqual(["tok-1"]);
});

test("a confirm link for an email another Account has says so", async () => {
  renderApp({
    path: "/confirm-email#token=tok-1",
    handlers: [
      ...linkServer("email-links", [liveLink("ada@example.test")]).handlers,
      http.post("*/api/v1/email-links/by-token/tok-1/confirm", () =>
        problem(409, "email_taken", "Another Account has that email"),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Confirm" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("Another Account has that email");
});
