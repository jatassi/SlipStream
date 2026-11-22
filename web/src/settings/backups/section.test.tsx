import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  backupsNow,
  backupsServer,
  listOf,
  nightly,
  olderNightly,
  relayDestination,
  said,
  shown,
  words,
} from "../../test/backups";
import { home, nodesAre, seedbox } from "../../test/nodes";
import { renderApp } from "../../test/render-app";

const desktop = { width: window.innerWidth, height: window.innerHeight };

afterEach(async () => {
  vi.useRealTimers();
  await page.viewport(desktop.width, desktop.height);
});

const atBackupsNow = () => {
  vi.useFakeTimers({ toFake: ["Date"], now: backupsNow });
};

function open(server: ReturnType<typeof backupsServer>, nodes = [home, seedbox]) {
  atBackupsNow();
  renderApp({ path: "/settings/backups", handlers: [nodesAre(...nodes), ...server.handlers] });
}

const main = () => page.getByRole("main");

test("the lead names the schedule, the newest good Backup and the Node with its copy", async () => {
  open(backupsServer());
  await expect
    .element(
      main().getByText(
        `SlipStream backs up your library every night at 04:00 and keeps scheduled Backups for 28 days. The newest good Backup was taken ${said(nightly.takenAt, "at")}, and seedbox has a copy.`,
      ),
    )
    .toBeVisible();
});

test("after a failure the lead says when it failed and links to the Backup task", async () => {
  open(
    backupsServer({
      list: listOf({
        failure: { at: "2026-11-17T04:00:00Z", since: "2026-11-17T04:00:00Z" },
        backups: [olderNightly],
      }),
    }),
  );
  await expect
    .element(
      main().getByText(
        `The last Backup failed ${said("2026-11-17T04:00:00Z", "at")}. The newest good Backup was taken ${said(olderNightly.takenAt, "at")}, and seedbox has a copy. Open the Backup task for the cause.`,
      ),
    )
    .toBeVisible();
  await expect
    .element(main().getByRole("link", { name: "Open the Backup task" }))
    .toHaveAttribute("href", "/tasks/backup");
});

test("with no destination and no other Node it recommends another disk or a download", async () => {
  open(
    backupsServer({ destinations: [], list: listOf({ backups: [{ ...nightly, copies: [] }] }) }),
    [home],
  );
  await expect
    .element(
      main().getByText(
        "Add a Backup destination on another disk, or download a Backup and keep it on another device. If this disk fails, Backups kept only on it are lost too.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      main().getByText(`The newest good Backup was taken ${said(nightly.takenAt, "at")}.`, {
        exact: false,
      }),
    )
    .toBeVisible();
});

test("an unsaved passphrase leads with saving it", async () => {
  open(backupsServer({ list: listOf({ passphrase: { savedAt: null } }) }));
  await expect
    .element(
      main().getByText(
        "Save your Backup passphrase somewhere other than this machine. You need it to restore a Backup if this machine is lost.",
      ),
    )
    .toBeVisible();
});

test("a recovery sheet that misses a destination asks for a new download", async () => {
  open(
    backupsServer({
      list: listOf({
        sheet: { downloadedAt: "2026-11-02T10:00:00Z", unlisted: [relayDestination.id] },
      }),
    }),
  );
  await expect
    .element(
      page.getByText(
        "Your recovery sheet doesn't list seedbox yet. Download it again and replace your saved copy.",
      ),
    )
    .toBeVisible();
});

test("the list shows each Backup's time, kind, version, size, check and copies", async () => {
  open(backupsServer());
  const list = main().getByRole("list", { name: "Backups" });
  const facts = "CheckedScheduled · SlipStream 2.1.0 · 96 MBCopies: seedbox copied";
  await expect
    .element(list.getByRole("listitem").first())
    .toHaveTextContent(`${shown(nightly.takenAt, "clock")}${facts}`);
  await expect
    .element(list.getByRole("listitem").nth(1))
    .toHaveTextContent(`${shown(olderNightly.takenAt, "clock")}${facts}`);
});

test("Show passphrase asks for the PIN first, then shows the words with Copy", async () => {
  open(backupsServer({ reconfirmed: false }));
  await userEvent.click(main().getByRole("button", { name: "Show passphrase" }));
  const prompt = page.getByRole("dialog", { name: "Confirm it's you" });
  await expect.element(prompt).toBeVisible();
  await userEvent.click(prompt.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("2468");
  await expect
    .element(main().getByRole("textbox", { name: "Backup passphrase" }))
    .toHaveValue(words);
  await expect
    .element(main().getByRole("button", { name: "Copy Backup passphrase" }))
    .toBeVisible();
});

test("the words show after the PIN even when the shell switched while the prompt was open", async () => {
  open(backupsServer({ reconfirmed: false }));
  await userEvent.click(main().getByRole("button", { name: "Show passphrase" }));
  const prompt = page.getByRole("dialog", { name: "Confirm it's you" });
  await expect.element(prompt).toBeVisible();
  await page.viewport(390, desktop.height);
  await expect.poll(() => document.querySelector('nav[aria-label="Tabs"]')).not.toBeNull();
  await page.viewport(desktop.width, desktop.height);
  await expect.poll(() => document.querySelector('nav[aria-label="Main"]')).not.toBeNull();
  await userEvent.click(prompt.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("2468");
  await expect
    .element(main().getByRole("textbox", { name: "Backup passphrase" }))
    .toHaveValue(words);
});

test("Replace deletes older Backups by default, and the box can keep them", async () => {
  const server = backupsServer();
  open(server);
  await userEvent.click(main().getByRole("button", { name: "Replace…" }));
  const dialog = page.getByRole("dialog", { name: "Replace the passphrase?" });
  await expect
    .element(
      dialog.getByText("Backups you already have still need the passphrase you have now.", {
        exact: false,
      }),
    )
    .toBeVisible();
  const deleteOlder = dialog.getByRole("checkbox", {
    name: "Delete older Backups and their copies once the new Backup is checked",
  });
  await expect.element(deleteOlder).toBeChecked();
  await userEvent.click(dialog.getByRole("button", { name: "Replace" }));
  await expect.element(dialog).not.toBeInTheDocument();
  await userEvent.click(main().getByRole("button", { name: "Replace…" }));
  await userEvent.click(deleteOlder);
  await userEvent.click(dialog.getByRole("button", { name: "Replace" }));
  await expect
    .poll(() => server.writes)
    .toEqual([
      { path: "replace", body: { deleteOlder: true } },
      { path: "replace", body: { deleteOlder: false } },
    ]);
});

test("each destination reads its newest copy and its check", async () => {
  open(backupsServer());
  await expect
    .element(main().getByRole("listitem", { name: "seedbox: /var/lib/slipstream/backups" }))
    .toMatchTextContent(
      `Newest copy ${said("2026-11-17T04:05:00Z", "at")}, checked ${said("2026-11-17T04:06:00Z", "at")}.`,
    );
});

test("adding a destination sends the Node and folder, and warns when the folder lacks room", async () => {
  const server = backupsServer({ destinations: [] });
  open(server);
  await userEvent.click(main().getByRole("button", { name: "Add a Backup destination" }).first());
  const dialog = page.getByRole("dialog", { name: "Add a Backup destination" });
  await userEvent.fill(
    dialog.getByRole("textbox", { name: "Folder" }),
    "/var/lib/slipstream/backups",
  );
  await userEvent.click(dialog.getByRole("button", { name: "Add" }));
  await expect
    .element(
      page.getByText(
        "Keeping these Backups needs about 3.1 GB on seedbox, which has 18 GB free. Free some space on seedbox, or add a Backup destination with more room.",
      ),
    )
    .toBeVisible();
  expect(server.writes).toEqual([
    { path: "addDestination", body: { nodeId: seedbox.id, folder: "/var/lib/slipstream/backups" } },
  ]);
});
