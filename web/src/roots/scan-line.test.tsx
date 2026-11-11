import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { RootsRoot, RootsScan } from "../api/gen/model";
import { liveServer, synced } from "../test/live-server";
import { act, openRoots } from "../test/roots-page";
import { archiveRoot, filmsRoot, notScanned, refused, scanRun, serveRoots } from "../test/roots";

const hoursAgo = (hours: number) => new Date(Date.now() - hours * 3_600_000).toISOString();

const scanned: RootsScan = {
  ...notScanned,
  lastAt: hoursAgo(2),
  summary: "It found 3 new files and 1 gone file.",
};

const stopped: RootsScan = {
  lastAt: hoursAgo(1),
  summary:
    "Library scan stopped because 1,912 of 2,000 files are gone. Check the disk or share it is on, then scan again.",
  stopped: { gone: 1912, of: 2000 },
  running: null,
};

const rowOf = (root: RootsRoot) => page.getByRole("row").filter({ hasText: root.path });

const menuItems = async (root: RootsRoot) => {
  await userEvent.click(page.getByRole("button", { name: `Actions for ${root.path}` }));
  await expect.element(page.getByRole("menu")).toBeVisible();
  const items = page
    .getByRole("menuitem")
    .all()
    .map((item) => item.element().textContent);
  await userEvent.keyboard("{Escape}");
  return items;
};

test("a root never scanned says so, and a scanned root says when and what the scan found", async () => {
  serveRoots({ roots: [filmsRoot, { ...archiveRoot, scan: scanned }] });
  openRoots();
  await expect.element(rowOf(filmsRoot).getByText("Not scanned yet.")).toBeVisible();
  await expect
    .element(
      rowOf(archiveRoot).getByText("Scanned 2 hours ago. It found 3 new files and 1 gone file."),
    )
    .toBeVisible();
});

test("each root's row is the anchor Library Health's stopped item links to", async () => {
  serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await expect.element(rowOf(archiveRoot)).toHaveAttribute("id", `root-${archiveRoot.id}`);
  await expect.element(rowOf(filmsRoot)).toHaveAttribute("id", `root-${filmsRoot.id}`);
});

test("Scan now scans the root and names it", async () => {
  const { sent } = serveRoots({ roots: [{ ...filmsRoot, scan: scanned }] });
  openRoots();
  expect(await menuItems(filmsRoot)).toEqual(["Scan now", "Edit", "Delete"]);
  await act(filmsRoot, "Scan now");
  await expect
    .element(page.getByText("Library scan started for the Films library root on home."))
    .toBeVisible();
  expect(sent.scanned).toEqual([filmsRoot.id]);
});

test("while its unit runs, the line follows each step from progress frames, and the run's end clears it", async () => {
  const live = liveServer();
  serveRoots({ roots: [{ ...filmsRoot, scan: scanned }] });
  const app = openRoots({ handlers: live.handlers });
  await synced(app);
  const row = rowOf(filmsRoot);
  await expect.element(row.getByText(/^Scanned 2 hours ago\./u)).toBeVisible();
  const step = (unit: { key: string; phase: string; done: number; total: number | null }) => {
    live.clients[0]?.send({
      event: "tasks.progress",
      data: { taskId: "library-scan", runId: scanRun, done: 0, total: 1, unit },
    });
  };
  const key = `root:${filmsRoot.id}`;

  step({ key, phase: "listing", done: 1204, total: null });
  await expect.element(row.getByText("Listing files, 1,204 so far")).toBeVisible();
  await expect.element(row.getByRole("meter")).not.toBeInTheDocument();

  step({ key, phase: "comparing", done: 0, total: 2014 });
  await expect.element(row.getByText("Comparing 2,014 files")).toBeVisible();

  step({ key, phase: "probing", done: 12, total: 40 });
  await expect.element(row.getByText("Reading media info, 12 of 40 files")).toBeVisible();
  await expect
    .element(row.getByRole("meter", { name: "Reading media info, 12 of 40 files" }))
    .toHaveAttribute("aria-valuenow", "12");

  step({ key: `root:${archiveRoot.id}`, phase: "listing", done: 3, total: null });
  live.clients[0]?.send({
    event: "tasks.progress",
    data: {
      taskId: "metadata-refresh",
      runId: 7,
      done: 1,
      total: 2,
      unit: { key, phase: "listing", done: 9, total: null },
    },
  });
  step({ key, phase: "probing", done: 13, total: 40 });
  await expect.element(row.getByText("Reading media info, 13 of 40 files")).toBeVisible();
  expect(await menuItems(filmsRoot)).toEqual(["Cancel library scan", "Edit", "Delete"]);

  live.clients[0]?.send({ event: "tasks.changed", data: { taskId: "library-scan" } });
  await expect
    .element(row.getByText("Scanned 2 hours ago. It found 3 new files and 1 gone file."))
    .toBeVisible();
});

test("Cancel library scan cancels the whole run the root's unit is in", async () => {
  const running: RootsRoot = {
    ...filmsRoot,
    scan: { ...scanned, running: { runId: scanRun, phase: "comparing", done: 10, total: 2014 } },
  };
  const { sent } = serveRoots({ roots: [running] });
  openRoots();
  await expect.element(rowOf(filmsRoot).getByText("Comparing 2,014 files")).toBeVisible();
  await act(filmsRoot, "Cancel library scan");
  await expect
    .element(
      page.getByText(
        "Library scan canceled for every root in the run. It keeps what it already recorded.",
      ),
    )
    .toBeVisible();
  expect(sent.canceled).toEqual([scanRun]);
});

test("a stopped scan shows why, and Mark them gone states the count before it marks them", async () => {
  const { sent } = serveRoots({ roots: [{ ...filmsRoot, scan: stopped }] });
  openRoots();
  await expect.element(rowOf(filmsRoot).getByText(stopped.summary)).toBeVisible();
  expect(await menuItems(filmsRoot)).toEqual(["Mark them gone", "Scan now", "Edit", "Delete"]);
  await act(filmsRoot, "Mark them gone");
  const dialog = page.getByRole("dialog", { name: "Mark 1,912 files gone?" });
  await expect
    .element(
      dialog.getByText(
        "Library scan didn't find 1,912 of the 2,000 files in the Films library root on home.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Mark them gone" }));
  await expect.element(dialog).not.toBeInTheDocument();
  await expect
    .element(
      page.getByText("Library scan is marking 1,912 files gone in the Films library root on home."),
    )
    .toBeVisible();
  expect(sent.marked).toEqual([filmsRoot.id]);
});

test("Mark them gone after the stop cleared says no files are waiting", async () => {
  serveRoots({
    roots: [{ ...filmsRoot, scan: stopped }],
    gone: () => refused(409, "scan_not_stopped", "The root's last library scan didn't stop"),
  });
  openRoots();
  await act(filmsRoot, "Mark them gone");
  const dialog = page.getByRole("dialog", { name: "Mark 1,912 files gone?" });
  await userEvent.click(dialog.getByRole("button", { name: "Mark them gone" }));
  await expect
    .element(
      dialog.getByText(
        "This root's last library scan didn't stop, so no files are waiting to be marked gone.",
      ),
    )
    .toBeVisible();
});
