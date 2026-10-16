import { HttpResponse, type RequestHandler, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { updateOwnAccount } from "../api/gen/query/identity/identity";
import { ceremony, passkeysAt, passkeysHere, stubPasskeys } from "../test/account";
import { ada, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";

const answer = { account: { ...ada, displayName: "Ada O." }, pinRequired: false, keyPrompt: "ask" };

function lapsedServer() {
  let confirmed = false;
  let tries = 0;
  return {
    tries: () => tries,
    confirm: () => {
      confirmed = true;
    },
    handlers: [
      http.patch("*/api/v1/account", () => {
        tries += 1;
        return confirmed
          ? HttpResponse.json(answer)
          : problem(403, "needs_reconfirm", "Confirm it's you to continue");
      }),
      http.post("*/api/v1/session/reconfirm", () => {
        confirmed = true;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  };
}

const dialog = () => page.getByRole("dialog", { name: "Confirm it's you" });

async function opened(handlers: RequestHandler[]) {
  renderApp({ path: "/library", handlers });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
}

test("at an address that holds passkeys, Use a passkey confirms and the action runs once more", async () => {
  const server = lapsedServer();
  const { get } = stubPasskeys();
  get.mockResolvedValue({ id: "cred-1", type: "public-key" });
  const finished: unknown[] = [];
  await opened([
    passkeysAt(passkeysHere),
    ...server.handlers,
    http.post("*/api/v1/session/reconfirm/passkey", () => ceremony("c-7")),
    http.post("*/api/v1/session/reconfirm/passkey/c-7", async ({ request }) => {
      finished.push(await request.json());
      server.confirm();
      return new HttpResponse(null, { status: 204 });
    }),
  ]);
  const saved = updateOwnAccount({ displayName: "Ada O." });
  await userEvent.click(dialog().getByRole("button", { name: "Use a passkey" }));

  await expect(saved).resolves.toEqual(answer);
  expect(server.tries()).toBe(2);
  expect(get.mock.calls[0]?.[0]).toEqual({ challenge: "Y2hhbGxlbmdl" });
  expect(finished).toEqual([{ id: "cred-1", type: "public-key" }]);
});

test("Use a passkey with no passkey for this address says to enter the PIN, which still works", async () => {
  const server = lapsedServer();
  stubPasskeys();
  await opened([
    passkeysAt(passkeysHere),
    ...server.handlers,
    http.post("*/api/v1/session/reconfirm/passkey", () =>
      problem(409, "passkey_unavailable", "A passkey can't be used at this address"),
    ),
  ]);
  const saved = updateOwnAccount({ displayName: "Ada O." });
  await userEvent.click(dialog().getByRole("button", { name: "Use a passkey" }));
  await expect
    .element(dialog().getByRole("alert"))
    .toHaveTextContent("You have no passkey for this address, so enter your PIN.");

  await userEvent.click(dialog().getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("1234");
  await expect(saved).resolves.toEqual(answer);
});

test("an address that can't hold a passkey offers only the PIN", async () => {
  const server = lapsedServer();
  await opened(server.handlers);
  const refused = expect(updateOwnAccount({ displayName: "Ada O." })).rejects.toMatchObject({
    code: "needs_reconfirm",
  });
  await expect.element(dialog().getByRole("button", { name: "Cancel" })).toBeVisible();
  await expect
    .element(dialog().getByRole("button", { name: "Use a passkey" }))
    .not.toBeInTheDocument();
  await userEvent.click(dialog().getByRole("button", { name: "Cancel" }));
  await refused;
});
