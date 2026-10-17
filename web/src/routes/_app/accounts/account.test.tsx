import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IdentityAccountDetail } from "../../../api/gen/model";
import {
  getGetAccountMockHandler,
  getListAccountsMockHandler,
  getListSignupLinksMockHandler,
} from "../../../api/gen/query/identity/identity.msw";
import { clock } from "../../../format";
import { ada, grace, problem } from "../../../test/handlers";
import { renderApp } from "../../../test/render-app";

const graceDetail: IdentityAccountDetail = {
  account: {
    ...grace,
    disabled: false,
    passkeyCount: 1,
    pin: { state: "off" },
    lastSignInAt: "2026-10-15T08:30:00Z",
  },
  passkeys: [
    {
      id: "3e5a7c9b-2d4f-4b6a-8c0e-1f3a5b7c9d2e",
      name: "iCloud Keychain",
      host: "media.example.test",
      createdAt: "2026-10-02T12:00:00Z",
    },
  ],
  sessions: [
    {
      id: "9b1d3f5a-7c2e-4a6b-8d0f-2e4a6c8b0d1f",
      device: "Safari on iPhone",
      address: "192.168.1.20",
      createdAt: "2026-10-10T09:00:00Z",
      lastSeenAt: "2026-10-16T07:45:00Z",
      current: false,
    },
  ],
};

const adaDetail: IdentityAccountDetail = {
  account: { ...ada, disabled: false, passkeyCount: 0, pin: { state: "fine" } },
  passkeys: [],
  sessions: [],
};

const listItems = (name: string) =>
  page
    .getByRole("list", { name })
    .getByRole("listitem")
    .all()
    .map((item) => item.element().textContent);

test("an Account shows its profile, PIN state, passkeys and sessions", async () => {
  renderApp({
    path: `/accounts/${grace.id}`,
    handlers: [getGetAccountMockHandler(graceDetail)],
  });
  await expect.element(page.getByRole("heading", { name: "Grace Lindqvist" })).toBeVisible();
  expect(listItems("Profile")).toEqual([
    "Grace LindqvistName",
    "grace@example.testEmailChange email",
    "MemberRoleMake Admin",
    "OnSign-inTurn off",
    `${clock("2026-10-15T08:30:00Z")}Last sign-in`,
  ]);
  expect(listItems("PIN")).toEqual([
    "Too many wrong PINs turned the PIN off. Send a Reset link to set a new one.Send a Reset link",
  ]);
  expect(listItems("Passkeys")).toEqual([
    `iCloud Keychainmedia.example.test · Added ${clock("2026-10-02T12:00:00Z")}`,
  ]);
  expect(listItems("Sessions")).toEqual([
    `Safari on iPhone192.168.1.20 · Last seen ${clock("2026-10-16T07:45:00Z")}`,
  ]);
});

test("the last Admin's demotion is refused inside the dialog", async () => {
  renderApp({
    path: `/accounts/${ada.id}`,
    handlers: [
      getGetAccountMockHandler(adaDetail),
      http.patch("*/api/v1/accounts/:id", () =>
        problem(409, "last_admin", "SlipStream needs at least one Admin who is turned on"),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Make Member" }));
  const dialog = page.getByRole("dialog", { name: "Make Ada Okafor a Member?" });
  await userEvent.click(dialog.getByRole("button", { name: "Make Member" }));
  await expect
    .element(dialog.getByRole("alert"))
    .toHaveTextContent("SlipStream needs at least one Admin who is turned on");
});

test("Turn off and Sign out everywhere ask first, then send the change", async () => {
  const calls: string[] = [];
  renderApp({
    path: `/accounts/${grace.id}`,
    handlers: [
      getGetAccountMockHandler(graceDetail),
      http.patch("*/api/v1/accounts/:id", async ({ request }) => {
        calls.push(`PATCH ${JSON.stringify(await request.json())}`);
        return HttpResponse.json({ ...graceDetail.account, disabled: true });
      }),
      http.delete("*/api/v1/accounts/:id/sessions", () => {
        calls.push("DELETE sessions");
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Turn off" }));
  const off = page.getByRole("dialog", { name: "Turn off Grace Lindqvist?" });
  await expect
    .element(
      off.getByText(
        "They're signed out everywhere and can't sign in until an Admin turns them on.",
      ),
    )
    .toBeVisible();
  await userEvent.click(off.getByRole("button", { name: "Turn off" }));
  await expect.element(off).not.toBeInTheDocument();

  await userEvent.click(page.getByRole("button", { name: "Sign out everywhere" }));
  const signOut = page.getByRole("dialog", { name: "Sign Grace Lindqvist out everywhere?" });
  await userEvent.click(signOut.getByRole("button", { name: "Sign out everywhere" }));
  await expect.element(signOut).not.toBeInTheDocument();
  expect(calls).toEqual(['PATCH {"disabled":true}', "DELETE sessions"]);
});

test("Change email sends the new address and shows a taken one on the field", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: `/accounts/${grace.id}`,
    handlers: [
      getGetAccountMockHandler(graceDetail),
      http.patch("*/api/v1/accounts/:id", async ({ request }) => {
        sent.push(await request.json());
        return sent.length === 1
          ? problem(409, "email_taken", "Another Account has that email")
          : HttpResponse.json({ ...graceDetail.account, email: "grace@new.example.test" });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Change email" }));
  const dialog = page.getByRole("dialog", { name: "Change Grace Lindqvist's email" });
  const email = dialog.getByRole("textbox", { name: "Email" });
  await userEvent.fill(email, "ada@example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Change email" }));
  await expect.element(dialog.getByText("Another Account has that email")).toBeVisible();

  await userEvent.fill(email, "grace@new.example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Change email" }));
  await expect.element(dialog).not.toBeInTheDocument();
  await expect
    .element(page.getByText("Changed Grace Lindqvist's email to grace@new.example.test"))
    .toBeVisible();
  expect(sent).toEqual([{ email: "ada@example.test" }, { email: "grace@new.example.test" }]);
});

test("Send a Reset link shows the link once and says it was emailed", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: `/accounts/${grace.id}`,
    handlers: [
      getGetAccountMockHandler(graceDetail),
      http.post("*/api/v1/accounts/:id/reset-link", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json(
          { url: "https://media.example.test/reset#token=xyz789", email: { outcome: "sent" } },
          { status: 201 },
        );
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Send a Reset link" }));
  const dialog = page.getByRole("dialog", { name: "Send Grace Lindqvist a Reset link" });
  await expect
    .element(dialog.getByText("Turn this on if someone else may have had access."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("checkbox", { name: "Also remove passkeys" }));
  await userEvent.click(dialog.getByRole("button", { name: "Send the Reset link" }));

  await expect
    .element(dialog.getByRole("textbox", { name: "Reset link" }))
    .toHaveValue("https://media.example.test/reset#token=xyz789");
  await expect.element(dialog.getByText("Emailed it to grace@example.test.")).toBeVisible();
  await expect.element(dialog.getByText("SlipStream shows this link only once.")).toBeVisible();
  expect(sent).toEqual([{ removePasskeys: true }]);
});

test("Delete asks first, then returns to Accounts", async () => {
  const { router } = renderApp({
    path: `/accounts/${grace.id}`,
    handlers: [
      getGetAccountMockHandler(graceDetail),
      http.delete("*/api/v1/accounts/:id", () => new HttpResponse(null, { status: 204 })),
      getListAccountsMockHandler({ accounts: [adaDetail.account] }),
      getListSignupLinksMockHandler({ links: [] }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Delete Account" }));
  const dialog = page.getByRole("dialog", { name: "Delete Grace Lindqvist?" });
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect.poll(() => router.state.location.pathname).toBe("/accounts");
  await expect.element(page.getByRole("heading", { name: "Accounts", level: 1 })).toBeVisible();
});
