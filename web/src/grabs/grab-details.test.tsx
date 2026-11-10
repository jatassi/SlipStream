import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../api/gen/model";
import { getListGrabsMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { getListIndexersMockHandler } from "../api/gen/query/indexers/indexers.msw";
import { getListTrackersMockHandler } from "../api/gen/query/trackers/trackers.msw";
import { getListLibraryMockHandler } from "../api/gen/query/works/works.msw";
import { clock } from "../format";
import { downloading, waitingToFetch } from "../test/grabs";
// oxlint-disable-next-line import/max-dependencies -- Details read Grabs, Works, Targets, Indexers, Trackers and Torrents
import { problem } from "../test/handlers";
import { torrentLeech, torrentLeechTrackerId, trackers } from "../test/indexers";
import { orchardRow } from "../test/library";
import { homeId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";
import { actionsServer, torrentsServer } from "../test/torrents";

const override = "Ada Okafor grabbed it although its score of 5 is under 1080p's minimum of 10.";

const manual: GrabsGrab = {
  ...waitingToFetch,
  indexerId: torrentLeech.id,
  trackerId: torrentLeechTrackerId,
  source: "manual",
  grabbedBy: "Ada Okafor",
  overrides: [override],
  actions: { cancel: true, blocklistable: true, sendTo: false },
};

function grabServer(grab: GrabsGrab) {
  const cancels: unknown[] = [];
  return {
    cancels,
    handlers: [
      ...targetHandlers([{ ...starter, id: grab.targetIds[0] ?? "" }]),
      getListLibraryMockHandler({ rows: [orchardRow] }),
      getListGrabsMockHandler({ grabs: [grab] }),
      getListIndexersMockHandler({ indexers: [torrentLeech] }),
      getListTrackersMockHandler({ trackers }),
      ...torrentsServer([]).handlers,
      http.post("*/api/v1/grabs/:grabId/cancel", async ({ request }) => {
        cancels.push(await request.json());
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  };
}

const menu = () => page.getByRole("button", { name: "The Quiet Orchard actions" });

async function open(item: string) {
  await userEvent.click(menu());
  await userEvent.click(page.getByRole("menuitem", { name: item }));
}

test("a Grab's Details say who grabbed it and how, each override, where it went and each stage with its time", async () => {
  renderApp({ path: "/activity", handlers: grabServer(manual).handlers });
  await open("Details");
  const details = page.getByRole("dialog", { name: "Grab of The Quiet Orchard" });
  await expect.element(details).toMatchTextContent(/^Grab of The Quiet OrchardFetchingManual/u);
  await expect
    .element(details)
    .toMatchTextContent(`Manual search by Ada Okafor, ${clock(manual.createdAt)}.`);
  await expect.element(details.getByText(override)).toBeVisible();
  await expect
    .element(details)
    .toMatchTextContent(
      /ReleaseWEBDL-1080p · 700 MBThe\.Quiet\.Orchard\.2021\.1080p\.WEB-DL.*IndexerTorrentLeechTrackerTorrentLeechClientChosen once SlipStream fetches the Release/u,
    );
  await expect
    .element(details.getByRole("region", { name: "Stages" }))
    .toHaveTextContent(`Fetching${clock("2026-10-12T09:01:00Z")}`);
  await expect.element(details.getByRole("button", { name: "Cancel Grab" })).toBeVisible();
});

test("an automatic search's Grab says so and offers no cancel once it can't be canceled", async () => {
  const imported: GrabsGrab = {
    ...downloading,
    nodeId: homeId,
    stage: "imported",
    history: [...downloading.history, { stage: "imported", at: "2026-10-12T09:03:02Z" }],
  };
  renderApp({ path: "/activity", handlers: grabServer(imported).handlers });
  await open("Details");
  const details = page.getByRole("dialog", { name: "Grab of The Quiet Orchard" });
  await expect
    .element(details)
    .toMatchTextContent(
      /^Grab of The Quiet OrchardImportedAutomatic search, .*ClientBuilt-in client on home/u,
    );
  await expect
    .element(details.getByRole("region", { name: "Stages" }))
    .toMatchTextContent(/^Fetching.*Downloading.*Imported/u);
  await expect
    .element(details.getByRole("button", { name: "Cancel Grab" }))
    .not.toBeInTheDocument();
  await userEvent.click(details.getByRole("button", { name: "Close" }));
  await expect.element(details).not.toBeInTheDocument();
  await userEvent.click(menu());
  await expect.element(page.getByRole("menuitem", { name: "Cancel Grab" })).not.toBeInTheDocument();
});

test("Cancel with Also blocklist this Release cancels the Grab and blocklists its Release", async () => {
  const server = grabServer(manual);
  renderApp({ path: "/activity", handlers: server.handlers });
  await open("Details");
  await userEvent.click(page.getByRole("button", { name: "Cancel Grab" }));
  const cancel = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect
    .element(cancel)
    .toMatchTextContent("SlipStream stops fetching the Release. Its Item goes back to wanted.");
  await userEvent.click(cancel.getByRole("checkbox", { name: "Also blocklist this Release" }));
  await userEvent.click(cancel.getByRole("button", { name: "Cancel Grab" }));
  await expect
    .element(
      page
        .getByRole("region", { name: "Notifications" })
        .getByText("Canceled the Grab of The Quiet Orchard."),
    )
    .toBeVisible();
  expect(server.cancels).toEqual([{ blocklist: true }]);
});

test("a cancel the server refuses keeps the dialog open with its sentence", async () => {
  const busy =
    "This Grab is waiting for Den to come back. It can't be canceled until it places its file.";
  renderApp({
    path: "/activity",
    handlers: [
      http.post("*/api/v1/grabs/:grabId/cancel", () =>
        problem(409, "grab_busy", "This Grab is placing its file in the library.", busy),
      ),
      ...grabServer(manual).handlers,
    ],
  });
  await open("Cancel Grab");
  const cancel = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await userEvent.click(cancel.getByRole("button", { name: "Cancel Grab" }));
  await expect.element(cancel.getByRole("alert")).toHaveTextContent(busy);
  await userEvent.click(cancel.getByRole("button", { name: "Keep Grab" }));
  await expect.element(cancel).not.toBeInTheDocument();
});

test("a canceled Grab sits under Jobs with who canceled it, never under Needs you", async () => {
  const canceled: GrabsGrab = {
    ...manual,
    stage: "failed",
    failure: {
      stage: "fetching",
      code: "canceled",
      message: "Ada Okafor canceled this Grab.",
      needsYou: false,
    },
    actions: { cancel: false, blocklistable: false, sendTo: false },
  };
  renderApp({ path: "/activity", handlers: grabServer(canceled).handlers });
  await expect
    .element(page.getByRole("row", { name: /^The Quiet Orchard /u }))
    .toMatchTextContent(/1080pAda Okafor canceled this Grab\./u);
  await expect.element(page.getByRole("article")).not.toBeInTheDocument();
  await expect.element(page.getByRole("heading", { name: "Needs you" })).not.toBeInTheDocument();
});

test("a fetching Grab with no torrent yet has a menu with Cancel Grab, and no blocklist without an info hash", async () => {
  renderApp({
    path: "/activity",
    handlers: grabServer({ ...manual, actions: { ...manual.actions, blocklistable: false } })
      .handlers,
  });
  await open("Cancel Grab");
  const cancel = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect.element(cancel).toBeVisible();
  await expect
    .element(cancel.getByRole("checkbox", { name: "Also blocklist this Release" }))
    .not.toBeInTheDocument();
});

test("Details on a Needs you card opens its Grab's record", async () => {
  const failed: GrabsGrab = {
    ...manual,
    stage: "failed",
    failure: {
      stage: "fetching",
      code: "torrent_unreachable",
      message: "the Indexer answered 404 for the .torrent",
      needsYou: true,
    },
    actions: { cancel: false, blocklistable: false, sendTo: false },
  };
  renderApp({ path: "/activity", handlers: grabServer(failed).handlers });
  await userEvent.click(
    page
      .getByRole("article", { name: "The Quiet Orchard" })
      .getByRole("button", { name: "Details" }),
  );
  await expect
    .element(page.getByRole("dialog", { name: "Grab of The Quiet Orchard" }))
    .toMatchTextContent(
      /^Grab of The Quiet OrchardSlipStream couldn't download the Release's \.torrent file or magnet link from the Indexer\./u,
    );
});

test("Cancel on a downloading Grab names it once and lists only the other Grabs its torrent's removal cancels", async () => {
  const other = {
    grabId: "7d9f1b3c-5e7a-4c9b-8d2f-6a8c0e2f4a6b",
    workId: orchardRow.id,
    release: "The.Quiet.Orchard.2021.720p.WEB-DL",
  };
  const actions = actionsServer({
    preview: {
      cancels: [
        { grabId: downloading.id, workId: orchardRow.id, release: downloading.releaseTitle },
        other,
      ],
      stopSeeding: [],
      frees: 0,
      linked: false,
      hitAndRun: [],
    },
  });
  const grab: GrabsGrab = {
    ...downloading,
    actions: { cancel: true, blocklistable: true, sendTo: false },
  };
  renderApp({ path: "/activity", handlers: [actions.handler, ...grabServer(grab).handlers] });
  await open("Cancel Grab");
  const cancel = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect
    .element(cancel)
    .toMatchTextContent(
      "It also cancels another Grab still downloading this torrent, and its Item goes back to wanted:",
    );
  expect(cancel.getByText(downloading.releaseTitle).elements()).toHaveLength(1);
  await expect.element(cancel.getByText(other.release)).toBeVisible();
});
