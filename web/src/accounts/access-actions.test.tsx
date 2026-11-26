import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  IdentityAccountDetail,
  MediaserversAccessView as Access,
  MediaserversAccessViewActionsItem as Action,
} from "../api/gen/model";
import { getGetAccountMockHandler } from "../api/gen/query/identity/identity.msw";
import { grace } from "../test/handlers";
import { accessRows, linkableUsers, mediaServerAccessHandlers } from "../test/media-server-access";
import { denJellyfin, mediaServerHandlers } from "../test/media-servers";
import { renderApp } from "../test/render-app";

const graceDetail: IdentityAccountDetail = {
  account: { ...grace, disabled: false, passkeyCount: 0, pin: { state: "fine" } },
  passkeys: [],
  sessions: [],
};

const linkable: Access = {
  id: "0199f3c0-4444-7000-8000-000000000009",
  server: { id: denJellyfin.id, name: denJellyfin.name, kind: "jellyfin", access: "users" },
  member: { accountId: grace.id, name: grace.displayName, email: grace.email, disabled: false },
  state: "offered",
  needsYou: false,
  actions: ["link", "remove"],
  libraries: ["Films"],
  changedAt: "2026-11-24T10:00:00Z",
};

const rows: Access[] = [...Object.values(accessRows), linkable].map((row, index) =>
  Object.assign(structuredClone(row), {
    server: { ...row.server, name: `Server ${String(index + 1)}` },
  }),
);

const rowOf = (state: Access["state"], actions: readonly Action[]) => {
  const found = rows.find((row) => row.state === state && row.actions.join() === actions.join());
  if (found === undefined) {
    throw new Error(`no ${state} row with ${actions.join()}`);
  }
  return found;
};

type Sent = { method: string; path: string; body: unknown };

function graceAccess(sent: Sent[], plexHomeUsed = false) {
  const record = async (request: Request) => {
    const text = await request.text();
    sent.push({
      method: request.method,
      path: new URL(request.url).pathname,
      body: text === "" ? null : JSON.parse(text),
    });
    return HttpResponse.json({});
  };
  return [
    getGetAccountMockHandler(graceDetail),
    http.get("/api/v1/media-servers/access", () =>
      HttpResponse.json({ rows, offerable: [], plexPass: true, plexHomeUsed }),
    ),
    http.delete("/api/v1/media-servers/access/:id", ({ request }) => record(request)),
    http.put("/api/v1/accounts/:id/media-servers/:serverId", ({ request }) => record(request)),
    http.post("/api/v1/media-servers/access/:id/link", ({ request }) => record(request)),
    http.post("/api/v1/media-servers/access/:id/share", ({ request }) => record(request)),
    http.post("/api/v1/media-servers/access/:id/mark-added", ({ request }) => record(request)),
    http.post("/api/v1/media-servers/access/:id/check", ({ request }) => record(request)),
    ...mediaServerAccessHandlers(),
    ...mediaServerHandlers(),
  ];
}

const section = () => page.getByRole("list", { name: "Media Servers" });
const rowItem = (row: Access) =>
  section().getByRole("listitem").filter({ hasText: row.server.name });

type Press = { in: "dialog" | "row"; role: "button" | "radio"; name: string };
const press = (where: Press["in"], role: Press["role"], name: string): Press => ({
  in: where,
  role,
  name,
});

test.each<{ row: Access; open: string; next: Press[]; sent: (row: Access) => Sent }>([
  {
    row: rowOf("active", ["remove"]),
    open: "Remove access",
    next: [press("dialog", "button", "Remove access")],
    sent: (row: Access) => ({
      method: "DELETE",
      path: `/api/v1/media-servers/access/${row.id}`,
      body: null,
    }),
  },
  {
    row: rowOf("removing", ["retryRemove"]),
    open: "Try again",
    next: [],
    sent: (row: Access) => ({
      method: "DELETE",
      path: `/api/v1/media-servers/access/${row.id}`,
      body: null,
    }),
  },
  {
    row: rowOf("removed", ["offer"]),
    open: "Offer again",
    next: [],
    sent: (row: Access) => ({
      method: "PUT",
      path: `/api/v1/accounts/${row.member.accountId ?? ""}/media-servers/${row.server.id}`,
      body: {},
    }),
  },
  {
    row: rowOf("waiting", ["shareFriend", "shareHome", "markAdded", "remove"]),
    open: "Share by hand instead",
    next: [press("row", "button", "Mark as added")],
    sent: (row: Access) => ({
      method: "POST",
      path: `/api/v1/media-servers/access/${row.id}/mark-added`,
      body: null,
    }),
  },
  {
    row: rowOf("shared", ["check", "remove"]),
    open: "Check again",
    next: [],
    sent: (row: Access) => ({
      method: "POST",
      path: `/api/v1/media-servers/access/${row.id}/check`,
      body: null,
    }),
  },
  {
    row: rowOf("offered", ["link", "remove"]),
    open: "Link an existing account",
    next: [press("dialog", "radio", "kids"), press("dialog", "button", "Link")],
    sent: (row: Access) => ({
      method: "POST",
      path: `/api/v1/media-servers/access/${row.id}/link`,
      body: { userId: linkableUsers[1]?.id },
    }),
  },
])("$open on a $row.state row sends its call", async ({ row, open, next, sent }) => {
  const calls: Sent[] = [];
  renderApp({ path: `/accounts/${grace.id}`, handlers: graceAccess(calls) });
  await userEvent.click(rowItem(row).getByRole("button", { name: open }));
  for (const step of next) {
    const scope = step.in === "dialog" ? page.getByRole("dialog") : rowItem(row);
    // oxlint-disable-next-line no-await-in-loop -- each press follows the last
    await userEvent.click(scope.getByRole(step.role, { name: step.name }));
  }
  await expect.poll(() => calls).toEqual([sent(row)]);
});

test.each([
  { plexHomeUsed: false, warned: true },
  { plexHomeUsed: true, warned: false },
])(
  "Add to Plex Home warns first only while nobody is in the Plex Home (used: $plexHomeUsed)",
  async ({ plexHomeUsed, warned }) => {
    const calls: Sent[] = [];
    const row = rowOf("waiting", ["shareFriend", "shareHome", "markAdded", "remove"]);
    renderApp({ path: `/accounts/${grace.id}`, handlers: graceAccess(calls, plexHomeUsed) });
    await userEvent.click(rowItem(row).getByRole("button", { name: "Add to Plex Home" }));
    if (warned) {
      expect(calls).toEqual([]);
      await userEvent.click(
        page.getByRole("dialog").getByRole("button", { name: "Add to Plex Home" }),
      );
    }
    await expect
      .poll(() => calls)
      .toEqual([
        {
          method: "POST",
          path: `/api/v1/media-servers/access/${row.id}/share`,
          body: { as: "home" },
        },
      ]);
  },
);
