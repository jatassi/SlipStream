import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  accountDialog as dialog,
  accountLists,
  accountMain as main,
  ceremony,
  iPhonePasskey,
  openAccount,
  passkeysAt,
  passkeysHere,
  recorder,
  stubPasskeys,
} from "../../test/account";
import { signedInAsGrace } from "../../test/handlers";
import { renderApp } from "../../test/render-app";

test("an address that can't hold a passkey says why and offers no Add a passkey", async () => {
  await openAccount([passkeysAt({ available: false, reason: "ip_address" })]);
  await expect
    .element(
      page.getByText(
        "Passkeys need SlipStream's name, not an IP address. Open SlipStream by its name to use one.",
      ),
    )
    .toBeVisible();
  await expect
    .element(main().getByRole("button", { name: "Add a passkey" }))
    .not.toBeInTheDocument();
});

test("a passkey can be renamed and deleted", async () => {
  const { seen, record } = recorder();
  await openAccount([
    http.patch("*/api/v1/passkeys/p-iphone", async ({ request }) => {
      await record(request);
      return HttpResponse.json({ ...iPhonePasskey, name: "Ada's iPhone" });
    }),
    http.delete("*/api/v1/passkeys/p-iphone", async ({ request }) => {
      await record(request);
      return new HttpResponse(null, { status: 204 });
    }),
  ]);
  await userEvent.click(page.getByRole("button", { name: "iCloud Keychain actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Rename" }));
  await userEvent.fill(dialog("Rename passkey").getByLabelText("Name"), "Ada's iPhone");
  await userEvent.click(dialog("Rename passkey").getByRole("button", { name: "Save" }));
  await expect.element(dialog("Rename passkey")).not.toBeInTheDocument();

  await userEvent.click(page.getByRole("button", { name: "iCloud Keychain actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  await expect
    .element(
      dialog("Delete this passkey?").getByText(
        "iCloud Keychain will no longer sign you in at localhost.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog("Delete this passkey?").getByRole("button", { name: "Delete" }));
  await expect.element(dialog("Delete this passkey?")).not.toBeInTheDocument();

  expect(seen).toEqual([
    { method: "PATCH", url: "/api/v1/passkeys/p-iphone", body: { name: "Ada's iPhone" } },
    { method: "DELETE", url: "/api/v1/passkeys/p-iphone", body: null },
  ]);
});

test("Add a passkey runs registration and names what it added", async () => {
  const { create } = stubPasskeys();
  create.mockResolvedValue({ id: "cred-2", type: "public-key" });
  const { seen, record } = recorder();
  const { router } = await openAccount([
    passkeysAt(passkeysHere),
    http.post("*/api/v1/passkeys/registration", () => ceremony("r-1")),
    http.post("*/api/v1/passkeys/registration/r-1", async ({ request }) => {
      await record(request);
      return HttpResponse.json({ ...iPhonePasskey, id: "p-new", name: "Chrome on Mac" });
    }),
  ]);
  await userEvent.click(main().getByRole("button", { name: "Add a passkey" }));
  expect(router.state.location.href).toBe("/account?add=passkey");
  await userEvent.click(dialog("Add a passkey").getByRole("button", { name: "Continue" }));

  await expect.element(page.getByText("Added Chrome on Mac.")).toBeVisible();
  await expect.element(dialog("Add a passkey")).not.toBeInTheDocument();
  expect(router.state.location.href).toBe("/account");
  expect(create.mock.calls[0]?.[0]).toEqual({ challenge: "Y2hhbGxlbmdl" });
  expect(seen).toEqual([
    {
      method: "POST",
      url: "/api/v1/passkeys/registration/r-1",
      body: { id: "cred-2", type: "public-key" },
    },
  ]);
});

test.each([
  ["an Admin", [], "/account?add=passkey"],
  ["a Member", [signedInAsGrace], "/portal/account?add=passkey"],
])("the passkey link opens Add a passkey on %s's Your Account", async (_who, session, href) => {
  stubPasskeys();
  const { router } = renderApp({
    path: "/account/passkeys/new",
    handlers: [...session, passkeysAt(passkeysHere), ...accountLists()],
  });
  await expect.element(dialog("Add a passkey")).toBeVisible();
  expect(router.state.location.href).toBe(href);
});
