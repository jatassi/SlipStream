import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentityAccount, IdentityPinChange } from "../api/gen/model";
import { ChangePINBody } from "../api/gen/zod/identity/identity.zod";
import { ada, grace, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

function pinRequiredServer(account: IdentityAccount = ada) {
  let pinRequired = true;
  const changes: IdentityPinChange[] = [];
  return {
    handlers: [
      http.get("*/api/v1/session", () => HttpResponse.json({ account, pinRequired })),
      http.post("*/api/v1/account/pin", async ({ request }) => {
        changes.push(ChangePINBody.parse(await request.json()));
        pinRequired = false;
        return new HttpResponse(null, { status: 204 });
      }),
      http.all("*/api/v1/*", ({ request }) =>
        pinRequired && !request.url.endsWith("/status")
          ? problem(403, "pin_required", "Set a new PIN to continue")
          : undefined,
      ),
    ],
    changes,
  };
}

const heading = () => page.getByRole("heading", { name: "Set a new PIN" });

async function typeTwice(pin: string) {
  await userEvent.click(page.getByRole("textbox", { name: "New PIN", exact: true }));
  await userEvent.keyboard(pin);
  await expect.element(page.getByText("Type it again")).toBeVisible();
  await userEvent.keyboard(pin);
}

test("an Admin whose PIN was turned off sees only Set a new PIN, then the app once it's set", async () => {
  const server = pinRequiredServer();
  const { router } = renderApp({ path: "/activity", handlers: server.handlers });
  await expect.element(heading()).toBeVisible();
  expect(router.state.location.href).toBe("/new-pin");
  await expect.element(page.getByRole("navigation", { name: "Main" })).not.toBeInTheDocument();

  await typeTwice("2468");
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/library");
  expect(server.changes).toEqual([{ new: "2468" }]);
});

test("a Member on a phone sets it on the keypad and lands in the portal", async () => {
  await asPhone();
  const server = pinRequiredServer(grace);
  const { router } = renderApp({ path: "/portal/requests", handlers: server.handlers });
  await expect.element(heading()).toBeVisible();
  const keypad = page.getByRole("group", { name: "Keypad" });
  for (const digit of "13571357") {
    // oxlint-disable-next-line no-await-in-loop -- a person presses one key after another
    await userEvent.click(keypad.getByRole("button", { name: digit, exact: true }));
  }
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/portal");
  expect(server.changes).toEqual([{ new: "1357" }]);
});

test("a PIN the Node refuses says why and the page stays", async () => {
  const server = pinRequiredServer();
  renderApp({
    path: "/",
    handlers: [
      http.post("*/api/v1/account/pin", () =>
        problem(429, "too_many_attempts", "Too many failed attempts from this address."),
      ),
      ...server.handlers,
    ],
  });
  await expect.element(heading()).toBeVisible();
  await typeTwice("2468");
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("Too many failed attempts from this address.");
  await expect.element(heading()).toBeVisible();
});

test("Sign out on Set a new PIN asks first", async () => {
  const server = pinRequiredServer();
  renderApp({ path: "/", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Sign out" }));
  await expect
    .element(page.getByRole("dialog", { name: "Sign out of this device?" }))
    .toBeVisible();
  await userEvent.keyboard("{Escape}");
});

test("a session that needs no new PIN is taken home from Set a new PIN", async () => {
  const { router } = renderApp({ path: "/new-pin" });
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/library");
});
