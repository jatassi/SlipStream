import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentityAccount } from "../api/gen/model";
import { accountLists } from "../test/account";
import { ada, grace, problem } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";
import { paletteShortcut } from "./palette-state";

function sessionServer(account: IdentityAccount = ada) {
  let signedIn = true;
  let deletes = 0;
  return {
    handlers: [
      http.get("*/api/v1/session", () =>
        signedIn
          ? HttpResponse.json({ account, pinRequired: false, keyPrompt: "ask" })
          : problem(401, "signed_out", "Sign in to continue"),
      ),
      http.delete("*/api/v1/session", () => {
        deletes += 1;
        signedIn = false;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
    deletes: () => deletes,
  };
}

const confirm = () => page.getByRole("dialog", { name: "Sign out of this device?" });

test("Sign out in the Account menu asks first, and Cancel keeps this device signed in", async () => {
  const server = sessionServer();
  const { router } = renderApp({ path: "/library", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: /^Account/u }));
  await userEvent.click(page.getByRole("menuitem", { name: "Sign out" }));
  await expect.element(confirm()).toBeVisible();

  await userEvent.click(confirm().getByRole("button", { name: "Cancel" }));
  await expect.element(confirm()).not.toBeInTheDocument();
  expect(server.deletes()).toBe(0);
  expect(router.state.location.href).toBe("/library");

  await userEvent.click(page.getByRole("button", { name: /^Account/u }));
  await userEvent.click(page.getByRole("menuitem", { name: "Sign out" }));
  await userEvent.click(confirm().getByRole("button", { name: "Sign out" }));
  await expect.element(page.getByText("You signed out.")).toBeVisible();
  expect(router.state.location.href).toBe("/sign-in?reason=signed-out");
  expect(server.deletes()).toBe(1);
  await expect.element(confirm()).not.toBeInTheDocument();
});

test("the palette's Sign out asks the same question", async () => {
  const server = sessionServer();
  renderApp({ path: "/library", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  const [modifier] = paletteShortcut.keys.split("+");
  await userEvent.keyboard(`{${modifier}>}k{/${modifier}}`);
  await userEvent.keyboard("sign out");
  await userEvent.keyboard("{Enter}");
  await expect.element(confirm()).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(confirm()).not.toBeInTheDocument();
  expect(server.deletes()).toBe(0);
});

test("on a phone, More's Sign out asks in a sheet", async () => {
  await asPhone();
  const server = sessionServer();
  const { router } = renderApp({ path: "/library", handlers: server.handlers });
  await userEvent.click(
    page.getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" }),
  );
  await userEvent.click(
    page.getByRole("dialog", { name: "More" }).getByRole("button", { name: "Sign out" }),
  );
  await expect.element(confirm()).toBeVisible();
  await userEvent.click(confirm().getByRole("button", { name: "Sign out" }));
  await expect.poll(() => router.state.location.href).toBe("/sign-in?reason=signed-out");
});

test("a Member's Sign out in the portal asks first", async () => {
  const server = sessionServer(grace);
  const { router } = renderApp({
    path: "/portal/account",
    handlers: [...server.handlers, ...accountLists()],
  });
  await userEvent.click(page.getByRole("button", { name: "Sign out of this device" }));
  await expect.element(confirm()).toBeVisible();
  await userEvent.click(confirm().getByRole("button", { name: "Sign out" }));
  await expect.poll(() => router.state.location.href).toBe("/sign-in?reason=signed-out");
});

test("a Sign out the Node refuses says so in the dialog and keeps it open", async () => {
  const server = sessionServer();
  renderApp({
    path: "/library",
    handlers: [
      http.delete("*/api/v1/session", () => problem(500, "internal", "Something failed")),
      ...server.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: /^Account/u }));
  await userEvent.click(page.getByRole("menuitem", { name: "Sign out" }));
  await userEvent.click(confirm().getByRole("button", { name: "Sign out" }));
  await expect.element(confirm().getByRole("alert")).toHaveTextContent("Something failed");
  await userEvent.click(confirm().getByRole("button", { name: "Cancel" }));
});
