import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentitySignedIn } from "../api/gen/model";
import { ceremony, passkeysAt, passkeysHere, stubPasskeys } from "../test/account";
import { ada, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";

const signedIn: IdentitySignedIn = {
  account: ada,
  pinRequired: false,
  keyPrompt: "ask",
  passkeyPrompt: false,
};

function passkeyServer(answer: () => Response = () => HttpResponse.json(signedIn)) {
  let session: IdentitySignedIn | null = null;
  const finished: unknown[] = [];
  return {
    finished,
    handlers: [
      passkeysAt(passkeysHere),
      http.get("*/api/v1/session", () =>
        session === null
          ? problem(401, "signed_out", "Sign in to continue")
          : HttpResponse.json(session),
      ),
      http.post("*/api/v1/passkeys/assertion", () => ceremony("a-1")),
      http.post("*/api/v1/passkeys/assertion/a-1", async ({ request }) => {
        finished.push(await request.json());
        const response = answer();
        if (response.ok) {
          session = signedIn;
        }
        return response;
      }),
    ],
  };
}

const credential = { id: "cred-1", type: "public-key" };

test("Sign in with a passkey signs in and goes where the person was headed", async () => {
  const { get } = stubPasskeys();
  get.mockResolvedValue(credential);
  const server = passkeyServer();
  const { router } = renderApp({ path: "/activity", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Sign in with a passkey" }));

  await expect.poll(() => router.state.location.href).toBe("/activity");
  expect(server.finished).toEqual([credential]);
});

test("the button aborts the ceremony waiting on autofill, which waits again after", async () => {
  const { get } = stubPasskeys({ autofill: true });
  const signals: { mediation: unknown; aborted: () => boolean }[] = [];
  get.mockImplementation((_options, { mediation, signal }) => {
    signals.push({ mediation, aborted: () => signal?.aborted === true });
    if (mediation === "conditional") {
      return new Promise((resolve) => {
        signal?.addEventListener("abort", () => {
          resolve(null);
        });
      });
    }
    return Promise.resolve(null);
  });
  renderApp({ path: "/sign-in", handlers: passkeyServer().handlers });
  await expect.poll(() => signals.length).toBe(1);
  await userEvent.click(page.getByRole("button", { name: "Sign in with a passkey" }));

  await expect
    .poll(() => signals.map((call) => [call.mediation, call.aborted()]))
    .toEqual([
      ["conditional", true],
      [undefined, false],
      ["conditional", false],
    ]);
});

test("a passkey chosen from autofill signs in", async () => {
  const { get } = stubPasskeys({ autofill: true });
  get.mockResolvedValue(credential);
  const { router } = renderApp({ path: "/sign-in", handlers: passkeyServer().handlers });
  await expect.poll(() => router.state.location.href).toBe("/library");
  expect(get.mock.calls[0]?.[1]).toMatchObject({ mediation: "conditional" });
});

test("a passkey that fails says so and leaves the PIN to use", async () => {
  const { get } = stubPasskeys();
  get.mockResolvedValue(credential);
  renderApp({
    path: "/sign-in",
    handlers: passkeyServer(() =>
      problem(401, "passkey_failed", "The passkey couldn't be verified"),
    ).handlers,
  });
  await userEvent.click(page.getByRole("button", { name: "Sign in with a passkey" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("The passkey couldn't be verified");
  await expect.element(page.getByRole("textbox", { name: "PIN", exact: true })).toBeVisible();
});

test("a passkey sign-in on a turned-off PIN goes to Set a new PIN", async () => {
  const { get } = stubPasskeys();
  get.mockResolvedValue(credential);
  const { router } = renderApp({
    path: "/sign-in",
    handlers: [
      ...passkeyServer(() => HttpResponse.json({ ...signedIn, pinRequired: true })).handlers,
      http.get("*/api/v1/session", () => HttpResponse.json({ ...signedIn, pinRequired: true })),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Sign in with a passkey" }));
  await expect.poll(() => router.state.location.href).toBe("/new-pin");
  await expect.element(page.getByRole("heading", { name: "Set a new PIN" })).toBeVisible();
});

test("an address that can't hold a passkey shows no passkey button", async () => {
  renderApp({
    path: "/sign-in",
    handlers: [http.get("*/api/v1/session", () => problem(401, "signed_out", "Sign in"))],
  });
  await expect.element(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Sign in with a passkey" }))
    .not.toBeInTheDocument();
});
