import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentityAddress, IdentitySettings, MailSettings } from "../../../api/gen/model";
import {
  getGetAPIKeyMockHandler,
  getGetAddressMockHandler,
  getRegenerateAPIKeyMockHandler,
} from "../../../api/gen/query/identity/identity.msw";
import {
  getGetMailSettingsMockHandler,
  getSendTestMailMockHandler,
} from "../../../api/gen/query/mail/mail.msw";
import { problem } from "../../../test/handlers";
import { renderApp } from "../../../test/render-app";

const noAddress: IdentityAddress = { address: "", suggestedAddress: "http://media-pc:8484" };

const about = {
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
} as const;
const wrongPINs = { one: "wrong PIN", many: "wrong PINs" };

const count = (key: string, label: string, value: number, min: number, max: number) => ({
  ...about,
  kind: "count" as const,
  key: `identity.${key}`,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  unit: wrongPINs,
});

const duration = (key: string, label: string, value: number) => ({
  ...about,
  kind: "duration" as const,
  key: `identity.${key}`,
  label,
  value,
  running: value,
  default: value,
  min: 60,
  max: 2_592_000,
  step: 60,
});

const identitySettings: IdentitySettings = {
  pinWaitAfter: count("pinWaitAfter", "Wrong PINs before a wait", 5, 3, 10),
  pinWait: duration("pinWait", "Wait after too many wrong PINs", 900),
  pinOffAfter: count("pinOffAfter", "Wrong PINs in a row before the PIN turns off", 10, 6, 20),
  addressFailures: count("addressFailures", "Failed attempts per address", 30, 10, 1000),
  addressWindow: duration("addressWindow", "Per-address window", 900),
  reconfirmWindow: duration("reconfirmWindow", "Re-confirm window", 900),
  sessionLifetime: duration("sessionLifetime", "Session lifetime", 2_592_000),
  signupLinkLifetime: duration("signupLinkLifetime", "Signup link lifetime", 604_800),
  resetLinkLifetime: duration("resetLinkLifetime", "Reset link lifetime", 86_400),
  emailLinkLifetime: duration("emailLinkLifetime", "Email confirm link lifetime", 86_400),
};

const mail: MailSettings = {
  host: "smtp.example.test",
  port: 587,
  security: "starttls",
  username: "slipstream",
  passwordSet: true,
  fromAddress: "slipstream@example.test",
  fromName: "SlipStream",
};

const field = (name: string) => page.getByRole("textbox", { name, exact: true });

test("/settings opens on Address, which suggests the address in use while none is set", async () => {
  const { router } = renderApp({
    path: "/settings",
    handlers: [getGetAddressMockHandler(noAddress)],
  });
  await expect.element(page.getByRole("heading", { name: "Settings", level: 1 })).toBeVisible();
  await expect.poll(() => router.state.location.pathname).toBe("/settings/address");
  const tabs = page.getByRole("navigation", { name: "Settings" });
  expect(
    tabs
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Address", "Email", "Sign-in", "API key", "Targets", "Profiles"]);
  await expect
    .element(tabs.getByRole("link", { name: "Address" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(field("Address")).toHaveValue("http://media-pc:8484");
  await expect
    .element(
      page.getByText(
        "A plain-HTTP address can't hold passkeys. Use https:// for passkeys to work.",
      ),
    )
    .toBeVisible();
});

test("a plain-HTTP address at a .localhost name holds passkeys, so it gets no warning", async () => {
  renderApp({
    path: "/settings/address",
    handlers: [getGetAddressMockHandler({ ...noAddress, address: "http://home.localhost:8484" })],
  });
  await expect.element(field("Address")).toHaveValue("http://home.localhost:8484");
  await expect
    .element(page.getByText("Signup, Reset and email links are built on this address."))
    .toBeVisible();
  await userEvent.fill(field("Address"), "http://media-pc:8484");
  await expect
    .element(
      page.getByText(
        "A plain-HTTP address can't hold passkeys. Use https:// for passkeys to work.",
      ),
    )
    .toBeVisible();
});

test("saving the address sends it, and an IP address is refused on the field", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/address",
    handlers: [
      getGetAddressMockHandler(noAddress),
      http.put("*/api/v1/identity/address", async ({ request }) => {
        const body: unknown = await request.json();
        sent.push(body);
        return sent.length === 1
          ? problem(422, "address_ip", "The address must be a name, not an IP address")
          : HttpResponse.json({ ...noAddress, address: "https://media.example.test" });
      }),
    ],
  });
  await userEvent.fill(field("Address"), "http://192.168.1.4:8484");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByText("The address must be a name, not an IP address"))
    .toBeVisible();

  await userEvent.fill(field("Address"), "https://media.example.test");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved the address")).toBeVisible();
  expect(sent).toEqual([
    { address: "http://192.168.1.4:8484" },
    { address: "https://media.example.test" },
  ]);
});

test("Email shows the saved server, keeps the password write-only and sends a test", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/email",
    handlers: [
      getGetMailSettingsMockHandler(mail),
      http.patch("*/api/v1/mail/settings", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json({ ...mail, port: 465, security: "tls" });
      }),
      getSendTestMailMockHandler({ outcome: "sent" }),
    ],
  });
  await expect
    .element(page.getByText("Every email SlipStream sends goes through smtp.example.test."))
    .toBeVisible();
  await expect.element(field("Server")).toHaveValue("smtp.example.test");
  await expect.element(page.getByLabelText("Password")).toHaveValue("");
  await expect
    .element(page.getByText("A password is saved. Type a new one to replace it."))
    .toBeVisible();

  await userEvent.fill(field("Port"), "465");
  await userEvent.click(page.getByRole("combobox", { name: "Security" }));
  await userEvent.click(page.getByRole("option", { name: "TLS", exact: true }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved the email settings")).toBeVisible();
  expect(sent).toEqual([
    {
      host: "smtp.example.test",
      port: 465,
      security: "tls",
      username: "slipstream",
      fromAddress: "slipstream@example.test",
      fromName: "SlipStream",
    },
  ]);

  await userEvent.click(page.getByRole("button", { name: "Send a test email" }));
  await expect.element(page.getByText("Sent a test email to ada@example.test.")).toBeVisible();
});

test("a typed password is sent, and a failed test shows the mail server's reply", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/email",
    handlers: [
      getGetMailSettingsMockHandler({ ...mail, passwordSet: false }),
      http.patch("*/api/v1/mail/settings", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json({ ...mail, passwordSet: true });
      }),
      getSendTestMailMockHandler({ outcome: "failed", detail: "535 5.7.8 Authentication failed" }),
    ],
  });
  await userEvent.fill(page.getByLabelText("Password"), "hunter22");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved the email settings")).toBeVisible();
  expect(sent).toEqual([expect.objectContaining({ password: "hunter22" })]);
  await expect.element(page.getByLabelText("Password")).toHaveValue("");

  await userEvent.click(page.getByRole("button", { name: "Send a test email" }));
  await expect
    .element(
      page.getByText(
        "The test email failed. The mail server said: 535 5.7.8 Authentication failed",
      ),
    )
    .toBeVisible();
});

test("Regenerate asks first, then shows the new key once with Copy", async () => {
  renderApp({
    path: "/settings/api-key",
    handlers: [
      getGetAPIKeyMockHandler({ createdAt: "2026-10-01T09:00:00Z" }),
      getRegenerateAPIKeyMockHandler({
        apiKey: "ssk_0123456789abcdefghijABCDEFGHIJ0123456789xyz",
        createdAt: "2026-10-16T09:00:00Z",
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Regenerate" }));
  const dialog = page.getByRole("dialog", { name: "Regenerate the API key?" });
  await expect
    .element(
      dialog.getByText(
        "The old key stops working at once. Anything that uses it needs the new one.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Regenerate" }));

  await expect
    .element(field("New API key"))
    .toHaveValue("ssk_0123456789abcdefghijABCDEFGHIJ0123456789xyz");
  await expect.element(page.getByRole("button", { name: "Copy New API key" })).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
});

test("Sign-in groups its settings and saves one, reading them again after", async () => {
  const sent: unknown[] = [];
  let reads = 0;
  renderApp({
    path: "/settings/sign-in",
    handlers: [
      http.get("*/api/v1/identity/settings", () => {
        reads += 1;
        return HttpResponse.json(identitySettings);
      }),
      http.patch("*/api/v1/identity/settings", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json(identitySettings);
      }),
    ],
  });
  await expect
    .element(
      page.getByRole("navigation", { name: "Settings" }).getByRole("link", { name: "Sign-in" }),
    )
    .toHaveAttribute("aria-current", "page");
  expect(
    page
      .getByRole("heading", { level: 2 })
      .all()
      .map((heading) => heading.element().textContent),
  ).toEqual([
    "Wrong PINs",
    "Failed attempts from one address",
    "Sessions and re-confirm",
    "Link lifetimes",
  ]);
  const readsBefore = reads;
  await userEvent.fill(page.getByRole("textbox", { name: "Wrong PINs before a wait" }), "4");
  await userEvent.click(
    page.getByRole("region", { name: "Wrong PINs" }).getByRole("button", { name: "Save" }).first(),
  );
  await expect.poll(() => sent).toEqual([{ pinWaitAfter: 4 }]);
  await expect.poll(() => reads).toBe(readsBefore + 1);
});

test("an unknown section is not found", async () => {
  renderApp({ path: "/settings/passwords" });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});
