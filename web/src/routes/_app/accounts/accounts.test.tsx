import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  IdentityAccountRow,
  IdentityIssuedSignupLink,
  IdentityPendingLink,
} from "../../../api/gen/model";
import {
  getGetAddressMockHandler,
  getGetSignupLinkMockHandler,
  getListAccountsMockHandler,
  getListSignupLinksMockHandler,
} from "../../../api/gen/query/identity/identity.msw";
import { clock } from "../../../format";
import { ada, grace, problem } from "../../../test/handlers";
import { renderApp } from "../../../test/render-app";

const adaRow: IdentityAccountRow = {
  ...ada,
  disabled: false,
  passkeyCount: 2,
  pin: { state: "fine" },
  lastSignInAt: "2026-10-15T08:30:00Z",
};

const graceRow: IdentityAccountRow = {
  ...grace,
  disabled: true,
  passkeyCount: 0,
  pin: { state: "off" },
};

const pending: IdentityPendingLink = {
  id: "7a2e4c6b-1d3f-4a5b-9c8d-0e1f2a3b4c5d",
  email: "lin@example.test",
  issuedAt: "2026-10-14T10:00:00Z",
  expiresAt: "2026-10-21T10:00:00Z",
  issuedBy: "Ada Okafor",
};

const issued = (email: string): IdentityIssuedSignupLink => ({
  link: { ...pending, email },
  url: "https://media.example.test/signup#token=abc123",
  email: { outcome: "not-set-up" },
});

const accountsPage = (links: IdentityPendingLink[] = [pending]) => [
  getListAccountsMockHandler({ accounts: [adaRow, graceRow] }),
  getListSignupLinksMockHandler({ links }),
];

const noAddress = { address: "", suggestedAddress: "http://media-pc:8484" };

const actions = () => page.getByRole("button", { name: "Actions for lin@example.test" });

const rowTexts = () =>
  page
    .getByRole("table", { name: "Accounts" })
    .getByRole("row")
    .all()
    .map((row) => row.element().textContent);

test("Accounts lists every Account and the pending Signup links", async () => {
  renderApp({ path: "/accounts", handlers: accountsPage() });
  await expect.element(page.getByRole("heading", { name: "Accounts", level: 1 })).toBeVisible();
  await expect.poll(rowTexts).toHaveLength(3);
  expect(rowTexts().slice(1)).toEqual([
    `Ada Okaforada@example.testAdminFine2${clock("2026-10-15T08:30:00Z")}On`,
    "Grace Lindqvistgrace@example.testMemberTurned off0NeverTurned off",
  ]);
  await expect
    .element(page.getByRole("link", { name: "Grace Lindqvist" }))
    .toHaveAttribute("href", `/accounts/${grace.id}`);
  await expect
    .element(page.getByRole("list", { name: "Pending Signup links" }).getByRole("listitem"))
    .toHaveTextContent(
      `lin@example.testSent ${clock(pending.issuedAt)} by Ada Okafor · Expires ${clock(pending.expiresAt)}`,
    );
});

test("New Signup link shows the link to copy and that email is not set up", async () => {
  const sent: unknown[] = [];
  const { router } = renderApp({
    path: "/accounts",
    handlers: [
      ...accountsPage([]),
      http.post("*/api/v1/signup-links", async ({ request }) => {
        sent.push(await request.json());
        return HttpResponse.json(issued("lin@example.test"), { status: 201 });
      }),
    ],
  });
  await expect
    .element(page.getByText("No Signup links are waiting.", { exact: false }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "New Signup link" }));
  const dialog = page.getByRole("dialog", { name: "New Signup link" });
  await expect.poll(() => router.state.location.search).toEqual({ newLink: true });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Email" }), " lin@example.test ");
  await userEvent.click(dialog.getByRole("button", { name: "Make the link" }));

  await expect
    .element(dialog.getByRole("textbox", { name: "Signup link" }))
    .toHaveValue("https://media.example.test/signup#token=abc123");
  await expect
    .element(dialog.getByText("Email is not set up, so copy the link and send it yourself."))
    .toBeVisible();
  expect(sent).toEqual([{ email: "lin@example.test" }]);

  await userEvent.click(dialog.getByRole("button", { name: "Done" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(router.state.location.search).toEqual({});
});

test("an email that already has an Account is refused on the field", async () => {
  renderApp({
    path: "/accounts?newLink=true",
    handlers: [
      ...accountsPage(),
      http.post("*/api/v1/signup-links", () =>
        problem(409, "account_exists", "An Account already has that email"),
      ),
    ],
  });
  const dialog = page.getByRole("dialog", { name: "New Signup link" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Email" }), "grace@example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Make the link" }));
  await expect.element(dialog.getByText("An Account already has that email")).toBeVisible();
});

test("with no address set, the dialog asks for it, saves it and then makes the link", async () => {
  const calls: string[] = [];
  let addressSet = false;
  renderApp({
    path: "/accounts?newLink=true",
    handlers: [
      ...accountsPage(),
      getGetAddressMockHandler(noAddress),
      http.put("*/api/v1/identity/address", async ({ request }) => {
        calls.push(`PUT ${JSON.stringify(await request.json())}`);
        addressSet = true;
        return HttpResponse.json({});
      }),
      http.post("*/api/v1/signup-links", () => {
        calls.push("POST signup-links");
        return addressSet
          ? HttpResponse.json(issued("lin@example.test"), { status: 201 })
          : problem(
              409,
              "address_unset",
              "Set SlipStream's address first, so links point somewhere people can reach",
            );
      }),
    ],
  });
  const dialog = page.getByRole("dialog", { name: "New Signup link" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Email" }), "lin@example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Make the link" }));

  await expect
    .element(dialog.getByRole("alert"))
    .toHaveTextContent("Set SlipStream's address first, so links point somewhere people can reach");
  const address = dialog.getByRole("textbox", { name: "SlipStream's address" });
  await expect.element(address).toHaveValue("http://media-pc:8484");
  await userEvent.fill(address, "https://media.example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Make the link" }));

  await expect
    .element(dialog.getByRole("textbox", { name: "Signup link" }))
    .toHaveValue("https://media.example.test/signup#token=abc123");
  expect(calls).toEqual([
    "POST signup-links",
    'PUT {"address":"https://media.example.test"}',
    "POST signup-links",
  ]);
});

const pendingLinkHandlers = (calls: string[]) => [
  ...accountsPage(),
  getGetSignupLinkMockHandler({
    link: pending,
    url: "https://media.example.test/signup#token=first",
  }),
  http.post("*/api/v1/signup-links/:id/resend", ({ params }) => {
    calls.push(`resend ${String(params["id"])}`);
    return HttpResponse.json(
      { ...issued("lin@example.test"), email: { outcome: "sent" } },
      { status: 201 },
    );
  }),
  http.delete("*/api/v1/signup-links/:id", ({ params }) => {
    calls.push(`withdraw ${String(params["id"])}`);
    return new HttpResponse(null, { status: 204 });
  }),
];

test("a pending link copies, sends again and withdraws", async () => {
  const calls: string[] = [];
  renderApp({ path: "/accounts", handlers: pendingLinkHandlers(calls) });

  await userEvent.click(actions());
  await userEvent.click(page.getByRole("menuitem", { name: "Copy link" }));
  const copy = page.getByRole("dialog", { name: "Signup link for lin@example.test" });
  await expect
    .element(copy.getByRole("textbox", { name: "Signup link" }))
    .toHaveValue("https://media.example.test/signup#token=first");
  await userEvent.click(copy.getByRole("button", { name: "Done" }));
  await expect.element(copy).not.toBeInTheDocument();

  await userEvent.click(actions());
  await userEvent.click(page.getByRole("menuitem", { name: "Send again" }));
  const resend = page.getByRole("dialog", { name: "Send a new Signup link?" });
  await expect
    .element(
      resend.getByText("A new link goes to lin@example.test, and the old one stops working."),
    )
    .toBeVisible();
  await userEvent.click(resend.getByRole("button", { name: "Send again" }));
  const resent = page.getByRole("dialog", { name: "New Signup link" });
  await expect.element(resent.getByText("Emailed it to lin@example.test.")).toBeVisible();
  await userEvent.click(resent.getByRole("button", { name: "Done" }));
  await expect.element(resent).not.toBeInTheDocument();

  await userEvent.click(actions());
  await userEvent.click(page.getByRole("menuitem", { name: "Withdraw" }));
  const withdraw = page.getByRole("dialog", { name: "Withdraw the Signup link?" });
  await userEvent.click(withdraw.getByRole("button", { name: "Withdraw" }));
  await expect.element(withdraw).not.toBeInTheDocument();
  expect(calls).toEqual([`resend ${pending.id}`, `withdraw ${pending.id}`]);
});

test("the palette's New Signup link opens the dialog on Accounts from any page", async () => {
  const { router } = renderApp({ path: "/activity", handlers: accountsPage() });
  await userEvent.click(
    page.getByRole("banner").getByRole("button", { name: "Search Works or run a command" }),
  );
  await userEvent.click(page.getByRole("option", { name: "New Signup link" }));
  await expect.poll(() => router.state.location.href).toBe("/accounts?newLink=true");
  await expect
    .element(
      page.getByRole("dialog", { name: "New Signup link" }).getByRole("textbox", { name: "Email" }),
    )
    .toBeVisible();
});
