import { HttpResponse, http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { RestoreSummary } from "../../api/gen/model";
import { readPref, writePref } from "../../shell/prefs";
import { backupsNow, saidInRestore } from "../../test/backups";
import { problem } from "../../test/handlers";
import { homeId, seedboxId } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { forgetKeptCode, rightCode, unclaimedNode } from "../../test/setup-node";

forgetKeptCode();

afterEach(() => {
  vi.useRealTimers();
});

const before = "2026-11-17T09:00:00Z";

const summary: RestoreSummary = {
  takenAt: "2026-11-14T04:00:00Z",
  version: "2.1.0",
  controller: "media-pc",
  nodes: [
    { id: homeId, name: "media-pc" },
    { id: seedboxId, name: "vps-relay" },
  ],
  newMachine: true,
};

type Answers = {
  upload?: () => Response | Promise<Response>;
  open?: () => Response;
};

function restoring({ upload, open }: Answers = {}) {
  const server = unclaimedNode();
  let startedAt = before;
  const calls: { path: string; code: string | null; type: string | null; body: unknown }[] = [];
  const record = async (request: Request, json: boolean) => {
    calls.push({
      path: new URL(request.url).pathname,
      code: request.headers.get("X-Setup-Code"),
      type: request.headers.get("Content-Type"),
      body: json ? await request.json() : (await request.blob()).size,
    });
  };
  const handlers = [
    // A restore deletes every session, so the restored Controller reads as signed out.
    http.get("*/api/v1/session", () => problem(401, "signed_out", "Sign in to continue")),
    http.get("*/api/v1/status", () =>
      HttpResponse.json({ claim: server.node.claim, passkeys: { available: false }, startedAt }),
    ),
    http.post("*/api/v1/setup/restore", async ({ request }) => {
      await record(request, false);
      return upload === undefined ? HttpResponse.json({ id: "u1" }, { status: 201 }) : upload();
    }),
    http.post("*/api/v1/setup/restore/u1/open", async ({ request }) => {
      await record(request, true);
      return open === undefined ? HttpResponse.json(summary) : open();
    }),
    http.post("*/api/v1/setup/restore/u1/confirm", async ({ request }) => {
      await record(request, true);
      server.node.claim = "controller";
      startedAt = "2026-11-17T09:05:00Z";
      return new HttpResponse(null, { status: 202 });
    }),
    ...server.handlers,
  ];
  return { calls, handlers, node: server.node };
}

const backup = new File(
  ["age-encryption.org/v1"],
  "slipstream-20261114T040000Z-2.1.0-scheduled-7k3m.tar.zst.age",
);

async function openRestore(server: ReturnType<typeof restoring>) {
  vi.useFakeTimers({ toFake: ["Date"], now: backupsNow, shouldAdvanceTime: true });
  writePref("setupCode", rightCode);
  const app = renderApp({ path: "/setup", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Restore a Backup" }));
  await expect
    .element(
      page.getByText("Pick the newest Backup you have. Its name holds the date it was taken."),
    )
    .toBeVisible();
  return app;
}

async function uploadAndOpen(passphrase = "acid bench cobra") {
  await userEvent.upload(page.getByLabelText("Backup file"), backup);
  await userEvent.fill(page.getByRole("textbox", { name: "Backup passphrase" }), passphrase);
  await userEvent.click(page.getByRole("button", { name: "Open the Backup" }));
}

test("a restore uploads the file, opens it, confirms the old Controller is off and signs in after the restart", async () => {
  const server = restoring();
  const { router } = await openRestore(server);
  await uploadAndOpen();
  await expect
    .element(
      page.getByText(
        `This Backup was taken ${saidInRestore(summary.takenAt, "at")} by SlipStream 2.1.0. It's a Backup of media-pc and lists 2 Nodes.`,
      ),
    )
    .toBeVisible();
  await expect
    .element(
      page.getByText(
        `After the restore, sign in with the PIN you had ${saidInRestore(summary.takenAt, "on")}.`,
      ),
    )
    .toBeVisible();
  const restore = page.getByRole("button", { name: "Restore", exact: true });
  await expect.element(restore).toBeDisabled();
  await userEvent.click(
    page.getByRole("checkbox", { name: "media-pc is off, and I'll wipe it if it starts again" }),
  );
  await userEvent.click(page.getByRole("checkbox", { name: "vps-relay" }));
  await userEvent.click(restore);
  await expect.poll(() => router.state.location.pathname, { timeout: 10_000 }).toBe("/sign-in");
  expect(server.calls).toEqual([
    {
      path: "/api/v1/setup/restore",
      code: rightCode,
      type: "application/octet-stream",
      body: backup.size,
    },
    {
      path: "/api/v1/setup/restore/u1/open",
      code: rightCode,
      type: "application/json",
      body: { passphrase: "acid bench cobra" },
    },
    {
      path: "/api/v1/setup/restore/u1/confirm",
      code: rightCode,
      type: "application/json",
      body: { oldControllerOff: true, remove: [seedboxId] },
    },
  ]);
  expect(readPref("setupCode")).toBe("");
});

test("a wrong passphrase stays on the passphrase with the Node's sentence", async () => {
  const wrong =
    "That passphrase doesn't open this Backup. If you replaced the passphrase after this Backup was taken, use the earlier one.";
  await openRestore(
    restoring({
      open: () => problem(422, "backup_wrong_passphrase", "Wrong Backup passphrase", wrong),
    }),
  );
  await uploadAndOpen();
  await expect.element(page.getByText(wrong)).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Backup passphrase" })).toBeVisible();
});

test("a damaged Backup goes back to picking a file with the Node's sentence", async () => {
  const damaged = "This Backup is damaged or incomplete. Copy it again, or pick another Backup.";
  await openRestore(
    restoring({ open: () => problem(422, "backup_damaged", "Damaged Backup", damaged) }),
  );
  await uploadAndOpen();
  await expect.element(page.getByText(damaged)).toBeVisible();
  await expect.element(page.getByLabelText("Backup file")).toBeInTheDocument();
});

test("an upload refused for space says how much it needs", async () => {
  const space =
    "Restoring needs 1.2 GB free in /config, and its volume has 800 MB free. Free some space, then try again.";
  await openRestore(
    restoring({ upload: () => problem(422, "backup_no_space", "Not enough space", space) }),
  );
  await userEvent.upload(page.getByLabelText("Backup file"), backup);
  await expect.element(page.getByText(space)).toBeVisible();
});

test("while its own upload holds the Setup gate the page asks nothing else behind the code", async () => {
  let release!: () => void;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  const server = restoring({
    upload: async () => {
      await held;
      return HttpResponse.json({ id: "u1" }, { status: 201 });
    },
  });
  await openRestore(server);
  const checks = server.node.headers.length;
  await userEvent.upload(page.getByLabelText("Backup file"), backup);
  await expect
    .element(page.getByRole("progressbar", { name: "Uploading the Backup" }))
    .toBeVisible();
  window.dispatchEvent(new Event("focus"));
  document.dispatchEvent(new Event("visibilitychange"));
  release();
  await expect.element(page.getByRole("textbox", { name: "Backup passphrase" })).toBeVisible();
  expect(server.node.headers.slice(checks)).toEqual([]);
});
