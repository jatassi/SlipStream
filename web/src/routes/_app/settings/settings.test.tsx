import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentitySettings, MailSettings } from "../../../api/gen/model";
import {
  getGetAPIKeyMockHandler,
  getGetIdentitySettingsMockHandler,
  getRegenerateAPIKeyMockHandler,
} from "../../../api/gen/query/identity/identity.msw";
import {
  getGetMailSettingsMockHandler,
  getSendTestMailMockHandler,
} from "../../../api/gen/query/mail/mail.msw";
import { problem } from "../../../test/handlers";
import { renderApp } from "../../../test/render-app";

const identitySettings: IdentitySettings = {
  address: "",
  suggestedAddress: "http://media-pc:8484",
  pinWaitAfter: 5,
  pinWaitSeconds: 900,
  pinOffAfter: 10,
  addressFailures: 30,
  addressWindowSeconds: 900,
  reconfirmWindowSeconds: 900,
  sessionLifetimeSeconds: 2_592_000,
  signupLinkLifetimeSeconds: 604_800,
  resetLinkLifetimeSeconds: 86_400,
  emailLinkLifetimeSeconds: 86_400,
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
    handlers: [getGetIdentitySettingsMockHandler(identitySettings)],
  });
  await expect.element(page.getByRole("heading", { name: "Address" })).toBeVisible();
  expect(router.state.location.pathname).toBe("/settings/address");
  expect(
    page
      .getByRole("radio")
      .all()
      .map((option) => option.element().textContent),
  ).toEqual(["Address", "Email", "API key"]);
  await expect.element(field("Address")).toHaveValue("http://media-pc:8484");
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
      getGetIdentitySettingsMockHandler(identitySettings),
      http.patch("*/api/v1/identity/settings", async ({ request }) => {
        const body: unknown = await request.json();
        sent.push(body);
        return sent.length === 1
          ? problem(422, "address_ip", "The address must be a name, not an IP address")
          : HttpResponse.json({ ...identitySettings, address: "https://media.example.test" });
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
  await expect.element(page.getByRole("heading", { name: "API key" })).toBeVisible();
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

test("an unknown section is not found", async () => {
  renderApp({ path: "/settings/sign-in" });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});
