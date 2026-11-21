import { HttpResponse, http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { BackupsBackup, RestoreSummary } from "../../api/gen/model";
import { backupsNow, backupsServer, listOf, nightly } from "../../test/backups";
import { problem } from "../../test/handlers";
import { home, nodesAre, seedbox } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { signedIn } from "../../test/setup-node";

afterEach(() => {
  vi.useRealTimers();
});

const summary: RestoreSummary = {
  takenAt: nightly.takenAt,
  version: "2.1.0",
  controller: "home",
  nodes: [
    { id: home.id, name: "home" },
    { id: seedbox.id, name: "seedbox" },
  ],
  newMachine: false,
};

function inPlace(backup: BackupsBackup) {
  const server = backupsServer({ list: listOf({ backups: [backup] }) });
  const calls: { path: string; body: unknown }[] = [];
  let restored = false;
  let startedAt = "2026-11-17T09:00:00Z";
  const record = async (request: Request) => {
    const text = await request.text();
    calls.push({
      path: new URL(request.url).pathname,
      body: text === "" ? null : (JSON.parse(text) as unknown),
    });
  };
  const handlers = [
    http.get("*/api/v1/session", () =>
      restored ? problem(401, "signed_out", "Sign in to continue") : HttpResponse.json(signedIn),
    ),
    http.get("*/api/v1/status", () =>
      HttpResponse.json({ claim: "controller", passkeys: { available: false }, startedAt }),
    ),
    http.post("*/api/v1/backups/:id/restore", async ({ request }) => {
      await record(request);
      return HttpResponse.json({ id: "r1", summary }, { status: 201 });
    }),
    http.post("*/api/v1/backups/:id/pull", async ({ request }) => {
      await record(request);
      return new HttpResponse(null, { status: 202 });
    }),
    http.post("*/api/v1/backups/restore/r1/confirm", async ({ request }) => {
      await record(request);
      restored = true;
      startedAt = "2026-11-17T09:05:00Z";
      return new HttpResponse(null, { status: 202 });
    }),
    nodesAre(home, seedbox),
    ...server.handlers,
  ];
  return { calls, handlers };
}

async function openRestore(server: ReturnType<typeof inPlace>) {
  vi.useFakeTimers({ toFake: ["Date"], now: backupsNow, shouldAdvanceTime: true });
  const app = renderApp({ path: "/settings/backups", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Actions for the Backup from 04:01" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Restore…" }));
  return { app, dialog: page.getByRole("dialog", { name: "Restore the Backup from 04:01?" }) };
}

test("Restore on a local Backup stages it, says what is lost and signs in after the restart", async () => {
  const server = inPlace(nightly);
  const { app, dialog } = await openRestore(server);
  await expect
    .element(
      dialog.getByText(
        "SlipStream backs up the library as it is now, replaces it and restarts. Changes since Nov 17 are lost.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Restore", exact: true }));
  await expect.poll(() => app.router.state.location.pathname, { timeout: 10_000 }).toBe("/sign-in");
  expect(server.calls).toEqual([
    { path: `/api/v1/backups/${nightly.id}/restore`, body: null },
    {
      path: "/api/v1/backups/restore/r1/confirm",
      body: { oldControllerOff: false, remove: [] },
    },
  ]);
});

test("Restore on a Backup kept only at a destination copies it back first", async () => {
  const server = inPlace({ ...nightly, local: false });
  const { dialog } = await openRestore(server);
  await expect
    .element(
      dialog.getByText(
        "This Backup's file isn't on this machine. SlipStream copies it back from seedbox first.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Copy it back" }));
  await expect.element(dialog.getByText("Copying the Backup back from seedbox…")).toBeVisible();
  await expect
    .poll(() => server.calls)
    .toEqual([{ path: `/api/v1/backups/${nightly.id}/pull`, body: { destinationId: "d1" } }]);
});
