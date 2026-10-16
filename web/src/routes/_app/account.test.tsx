import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { clock } from "../../format";
import { paletteShortcut } from "../../shell/palette-state";
import {
  accountDialog as dialog,
  accountLists,
  accountMain as main,
  iPhonePasskey,
  livingRoomTv,
  openAccount,
  passkeysAt,
  passkeysHere,
  recorder,
} from "../../test/account";
import { ada, problem } from "../../test/handlers";
import { renderApp } from "../../test/render-app";

test("Your Account shows the profile, passkeys and devices", async () => {
  await openAccount([passkeysAt(passkeysHere)]);
  const profile = page.getByRole("list", { name: "Profile" });
  await expect.element(profile.getByText("Ada Okafor")).toBeVisible();
  await expect.element(profile.getByText("ada@example.test")).toBeVisible();
  await expect
    .element(page.getByRole("list", { name: "Passkeys" }).getByText("iCloud Keychain"))
    .toBeVisible();
  await expect
    .element(
      page.getByText(
        `localhost · Added ${clock(iPhonePasskey.createdAt)} · Last used ${clock("2026-10-15T07:45:00Z")}`,
      ),
    )
    .toBeVisible();
  const devices = page.getByRole("list", { name: "Devices" });
  expect(
    devices
      .getByRole("listitem")
      .all()
      .map((row) => row.element().textContent),
  ).toEqual([
    "Chrome on macOS192.168.1.20 · This deviceSign out",
    `Safari on Apple TV192.168.1.31 · Active ${clock(livingRoomTv.lastSeenAt)}Sign out`,
  ]);
  await expect.element(main().getByRole("button", { name: "Add a passkey" })).toBeVisible();
});

test("changing the display name saves it and shows it", async () => {
  const { seen, record } = recorder();
  await openAccount([
    http.patch("*/api/v1/account", async ({ request }) => {
      await record(request);
      return HttpResponse.json({
        account: { ...ada, displayName: "Ada O." },
        pinRequired: false,
        keyPrompt: "ask",
      });
    }),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Change display name" }));
  await userEvent.fill(dialog("Change display name").getByLabelText("Display name"), "  Ada O. ");
  await userEvent.click(dialog("Change display name").getByRole("button", { name: "Save" }));

  await expect
    .element(page.getByRole("list", { name: "Profile" }).getByText("Ada O."))
    .toBeVisible();
  expect(seen).toEqual([
    { method: "PATCH", url: "/api/v1/account", body: { displayName: "Ada O." } },
  ]);
});

test("an email change says where the confirm link went", async () => {
  const { seen, record } = recorder();
  await openAccount([
    http.post("*/api/v1/account/email", async ({ request }) => {
      await record(request);
      return HttpResponse.json({ email: { outcome: "sent" } });
    }),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Change email" }));
  await userEvent.fill(dialog("Change email").getByLabelText("New email"), "ada@okafor.example");
  await userEvent.click(dialog("Change email").getByRole("button", { name: "Send confirm link" }));

  await expect
    .element(dialog("Change email").getByRole("status"))
    .toHaveTextContent(
      "We sent a confirm link to ada@okafor.example. Your email changes once you follow it.",
    );
  expect(seen).toEqual([
    { method: "POST", url: "/api/v1/account/email", body: { email: "ada@okafor.example" } },
  ]);
});

test("an email change before the address is set says email changes don't work yet", async () => {
  await openAccount([
    http.post("*/api/v1/account/email", () =>
      problem(409, "address_unset", "Set SlipStream's address first"),
    ),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Change email" }));
  await userEvent.fill(dialog("Change email").getByLabelText("New email"), "ada@okafor.example");
  await userEvent.click(dialog("Change email").getByRole("button", { name: "Send confirm link" }));

  await expect
    .element(dialog("Change email").getByRole("alert"))
    .toHaveTextContent(
      "Your Admin hasn't set SlipStream's address yet, so email changes don't work.",
    );
});

async function typePin(scope: ReturnType<typeof dialog>, label: string, pin: string) {
  await userEvent.click(scope.getByRole("textbox", { name: label, exact: true }));
  await userEvent.keyboard(pin);
}

test("Change PIN takes the current PIN, then the new one twice", async () => {
  const { seen, record } = recorder();
  await openAccount([
    http.post("*/api/v1/account/pin", async ({ request }) => {
      await record(request);
      return new HttpResponse(null, { status: 204 });
    }),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Change PIN" }));
  const change = dialog("Change PIN");
  await typePin(change, "Current PIN", "1234");
  await typePin(change, "New PIN", "5678");
  await typePin(change, "Type it again", "5678");

  await expect.element(page.getByText("Your PIN is changed.")).toBeVisible();
  await expect.element(change).not.toBeInTheDocument();
  expect(seen).toEqual([
    { method: "POST", url: "/api/v1/account/pin", body: { current: "1234", new: "5678" } },
  ]);
});

test("a wrong current PIN says so and asks for it again", async () => {
  await openAccount([
    http.post("*/api/v1/account/pin", () => problem(401, "pin_wrong", "The email or PIN is wrong")),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Change PIN" }));
  const change = dialog("Change PIN");
  await typePin(change, "Current PIN", "1235");
  await typePin(change, "New PIN", "5678");
  await typePin(change, "Type it again", "5678");

  await expect.element(change.getByRole("alert")).toHaveTextContent("That PIN is wrong.");
  await expect
    .element(change.getByRole("textbox", { name: "Current PIN", exact: true }))
    .toHaveValue("");
});

test("signing out another device asks first and ends that session", async () => {
  const { seen, record } = recorder();
  await openAccount([
    http.delete("*/api/v1/sessions/s-tv", async ({ request }) => {
      await record(request);
      return new HttpResponse(null, { status: 204 });
    }),
    http.delete("*/api/v1/sessions", async ({ request }) => {
      await record(request);
      return new HttpResponse(null, { status: 204 });
    }),
  ]);
  await userEvent.click(page.getByRole("button", { name: "Sign out Safari on Apple TV" }));
  await userEvent.click(
    dialog("Sign out Safari on Apple TV?").getByRole("button", { name: "Sign out" }),
  );
  await expect.element(dialog("Sign out Safari on Apple TV?")).not.toBeInTheDocument();

  await userEvent.click(page.getByRole("button", { name: "Sign out every other device" }));
  await userEvent.click(
    dialog("Sign out every other device?").getByRole("button", { name: "Sign out others" }),
  );
  await expect.element(dialog("Sign out every other device?")).not.toBeInTheDocument();

  expect(seen).toEqual([
    { method: "DELETE", url: "/api/v1/sessions/s-tv", body: null },
    { method: "DELETE", url: "/api/v1/sessions?others=true", body: null },
  ]);
});

test("the palette opens Your Account", async () => {
  const { router } = renderApp({ path: "/library", handlers: accountLists() });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  const [modifier] = paletteShortcut.keys.split("+");
  await userEvent.keyboard(`{${modifier}>}k{/${modifier}}`);
  await userEvent.keyboard("Your Account");
  await userEvent.keyboard("{Enter}");
  await expect.poll(() => router.state.location.href).toBe("/account");
});
