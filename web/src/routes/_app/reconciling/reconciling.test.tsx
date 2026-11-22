import { expect, test } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import type { ReconcilingReviewed } from "../../../api/gen/model";
import { liveServer } from "../../../test/live-server";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import {
  filmsStep,
  homeRow,
  nodeRow,
  reconcilingPage,
  restore,
  mintedCode,
  seedboxRow,
  serveReconciling,
} from "../../../test/reconciling";
import { renderApp } from "../../../test/render-app";
import { filmsRoot, media, serveRoots } from "../../../test/roots";
import { asPhone } from "../../../test/viewport";

const textOf = (locator: Locator) => () => locator.element().textContent;

const banner = () => page.getByRole("complementary", { name: "Reconciling" });
const lead =
  "SlipStream is matching the restored library with the torrents and files on your Nodes. Searches and removals wait until it finishes.";

test("the banner leads with what SlipStream is doing, then the next step, and opens Reconciling", async () => {
  serveReconciling(reconcilingPage());
  const app = renderApp({ path: "/library", handlers: [nodesAre(home, seedbox)] });
  await expect.poll(textOf(banner())).toContain(lead);
  await expect
    .poll(textOf(banner()))
    .toContain("Pair seedbox again and wait for 1 library root to finish scanning.");

  await userEvent.click(banner().getByRole("button", { name: "Open Reconciling" }));
  await expect.element(page.getByRole("heading", { name: "Reconciling", level: 1 })).toBeVisible();
  expect(app.router.state.location.pathname).toBe("/reconciling");
  await expect.element(banner()).not.toBeInTheDocument();
});

test("on phone the banner sits above the tab bar and opens Reconciling", async () => {
  await asPhone();
  serveReconciling(reconcilingPage());
  renderApp({ path: "/library", handlers: [nodesAre(home, seedbox)] });
  const line = banner().getByRole("link");
  await expect.poll(textOf(line)).toContain("Pair seedbox again");
  await userEvent.click(line);
  await expect.element(page.getByRole("button", { name: "New Join Code" })).toBeVisible();
});

test("when only a Node offline past the grace is left, the banner recommends continuing without it", async () => {
  const live = liveServer();
  const laptop = nodeRow({ id: seedboxId, name: "laptop", link: "lapsed" });
  const { sent } = serveReconciling(
    reconcilingPage({
      restore: { ...restore, newMachine: false },
      nodes: [homeRow, laptop],
      roots: [{ ...filmsStep, scanned: true }],
    }),
    { live },
  );
  renderApp({ path: "/library", handlers: live.handlers });
  await expect
    .poll(() => banner().element().textContent)
    .toMatch(
      /SlipStream is waiting only for laptop, which has been offline since .+\. Continue without it so searches start again\./u,
    );
  await userEvent.click(banner().getByRole("button", { name: "Continue without laptop" }));
  await expect.poll(() => sent.continued).toEqual([seedboxId]);
  await expect.poll(textOf(banner())).toContain("SlipStream finishes in a moment.");
});

test("after a restore on a new machine every page opens the old Controller's folders until Save all", async () => {
  const live = liveServer();
  const { sent } = serveReconciling(
    reconcilingPage({
      restore: { ...restore, rehome: true },
      roots: [{ ...filmsStep, saved: false }],
    }),
    { live },
  );
  serveRoots();
  const app = renderApp({
    path: "/library",
    handlers: [...live.handlers, nodesAre(media, seedbox)],
  });
  await expect
    .element(page.getByRole("heading", { name: "Find the old Controller's folders", level: 1 }))
    .toBeVisible();
  expect(app.router.state.location.pathname).toBe("/reconciling/folders");
  await expect
    .element(
      page.getByText("The folder is at the same path on this machine. Save to start the scans."),
    )
    .toBeVisible();
  await expect
    .poll(textOf(page.getByRole("list", { name: "Folders" })))
    .toContain("The folder is there.");

  await userEvent.click(page.getByRole("button", { name: "Save all" }));
  await expect
    .poll(() => sent.rehomed)
    .toEqual([{ roots: [{ rootId: filmsRoot.id }], done: true }]);
  await expect.element(page.getByRole("heading", { name: "Reconciling", level: 1 })).toBeVisible();
  expect(app.router.state.location.pathname).toBe("/reconciling");
});

test("Later ends the step and leaves the folder waiting on the Reconciling page", async () => {
  const live = liveServer();
  const { sent } = serveReconciling(
    reconcilingPage({
      restore: { ...restore, rehome: true },
      roots: [{ ...filmsStep, saved: false }],
    }),
    { live },
  );
  serveRoots({ check: ({ path }) => ({ usable: false, path, refusal: { reason: "not_found" } }) });
  renderApp({ path: "/library", handlers: [...live.handlers, nodesAre(media, seedbox)] });
  await expect
    .element(
      page.getByText(
        "0 of 1 folder is at the same path on this machine. Give a path for the Films library root on home, or leave it for later.",
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Later" }));
  await expect.poll(() => sent.rehomed).toEqual([{ roots: [], done: true }]);
  const roots = page.getByRole("list", { name: "Library roots" });
  await expect.poll(textOf(roots)).toContain("Waiting for its folder.");
  await expect.element(roots.getByRole("button", { name: "Find its folder" })).toBeVisible();
});

test("the Reconciling page leads with the next step, names each Node's step and the Role a Node was given", async () => {
  const granted = nodeRow({
    id: homeId,
    name: "home",
    held: 1240,
    reported: true,
    adopted: true,
    granted: "home now has the Downloading Role for the 1,240 torrents it reported.",
  });
  serveReconciling(reconcilingPage({ nodes: [granted, seedboxRow] }));
  renderApp({ path: "/reconciling", handlers: [nodesAre(home, seedbox), mintedCode] });
  await expect.element(page.getByRole("button", { name: "Reconciling" })).toBeVisible();
  await expect
    .element(page.getByText("Pair seedbox again and wait for 1 library root to finish scanning."))
    .toBeVisible();
  const nodes = page.getByRole("list", { name: "Nodes" });
  await expect
    .poll(textOf(nodes))
    .toContain("seedbox is still paired with the old Controller. Pair it again.");
  await expect
    .poll(textOf(nodes))
    .toContain("home now has the Downloading Role for the 1,240 torrents it reported.");

  await userEvent.click(nodes.getByRole("button", { name: "New Join Code" }));
  const dialog = page.getByRole("dialog", { name: "Pair seedbox again" });
  await expect
    .element(dialog.getByRole("textbox", { name: "Join Code" }))
    .toHaveValue("K7Q2-M4XP-9TRA");
  await userEvent.click(dialog.getByRole("button", { name: "Details" }));
  await expect.poll(textOf(dialog)).toContain("slipstream join K7Q2-M4XP-9TRA");
});

test("outside a restore the page lists only the Nodes that keep their torrents, with no Finish now", async () => {
  serveReconciling(
    reconcilingPage({
      restore: null,
      nodes: [nodeRow({ id: seedboxId, name: "seedbox" })],
      roots: [],
    }),
  );
  renderApp({ path: "/reconciling", handlers: [nodesAre(home, seedbox)] });
  await expect
    .element(
      page.getByText(
        "A newly paired Node keeps every torrent until SlipStream has matched its list of torrents with the library.",
      ),
    )
    .toBeVisible();
  await expect
    .poll(textOf(page.getByRole("list", { name: "Nodes" })))
    .toContain("Waiting for seedbox to report its torrents.");
  expect(page.getByRole("button", { name: "Finish now" }).query()).toBeNull();
  expect(page.getByRole("button", { name: /^Continue without/u }).query()).toBeNull();
  await expect.element(banner()).not.toBeInTheDocument();
});

const reviewed = (index: number, found: ReconcilingReviewed["found"]): ReconcilingReviewed => ({
  id: `0199f0a1-3333-7000-8000-${String(index).padStart(12, "0")}`,
  name: `Torrent ${index}`,
  nodeId: seedboxId,
  reason: "not_held",
  found,
  since: "2026-11-14T10:20:00Z",
  readd: null,
});

test("the review list groups torrents by what was found, pages by the cursor and re-adds a group", async () => {
  const withFiles = [1, 2, 3].map((index) => reviewed(index, "files"));
  const noFiles = Array.from({ length: 60 }, (_, index) => reviewed(100 + index, "no_files"));
  const { sent } = serveReconciling(
    reconcilingPage({
      review: [
        { reason: "not_held", found: "files", count: 3 },
        { reason: "not_held", found: "no_files", count: 60 },
      ],
    }),
    { torrents: [...withFiles, ...noFiles] },
  );
  renderApp({ path: "/reconciling/torrents", handlers: [nodesAre(home, seedbox)] });
  const found = page.getByRole("region", { name: "Files found" });
  await expect
    .poll(textOf(found))
    .toContain(
      "3 torrents have their files on disk. Re-add them to keep seeding from those files.",
    );
  await expect.poll(() => found.getByRole("listitem").elements().length).toBe(3);
  await expect
    .poll(textOf(found.getByRole("listitem").first()))
    .toContain("Torrent 1Was on seedbox");

  const none = page.getByRole("region", { name: "No files" });
  await expect.poll(() => none.getByRole("listitem").elements().length).toBe(50);
  await userEvent.click(none.getByRole("button", { name: "Load more" }));
  await expect.poll(() => none.getByRole("listitem").elements().length).toBe(60);
  expect(sent.pages).toContain(noFiles[49]?.id);

  await userEvent.click(found.getByRole("button", { name: "Re-add these 3" }));
  await expect.poll(() => sent.readd).toEqual([{ reason: "not_held", found: "files" }]);
  await expect
    .element(page.getByText("Re-adding 3 torrents once Reconciling finishes."))
    .toBeVisible();
});
