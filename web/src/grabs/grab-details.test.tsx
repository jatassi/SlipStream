import { HttpResponse, http } from "msw";
import { afterEach, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../api/gen/model";
import { getListGrabsMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { getListIndexersMockHandler } from "../api/gen/query/indexers/indexers.msw";
import { getListTrackersMockHandler } from "../api/gen/query/trackers/trackers.msw";
import { getListLibraryQueryKey } from "../api/gen/query/works/works";
import { getListLibraryMockHandler } from "../api/gen/query/works/works.msw";
import { downloading, fetchFailure, waitingToFetch } from "../test/grabs";
// oxlint-disable-next-line import/max-dependencies -- Details read Grabs, Works, Targets, Indexers, Trackers and Torrents, and count the Library's reads
import { problem } from "../test/handlers";
import { torrentLeech, torrentLeechTrackerId, trackers } from "../test/indexers";
import { orchardRow } from "../test/library";
import { homeId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";
import { actionsServer, torrentsServer } from "../test/torrents";
import { worker } from "../test/worker";

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

afterEach(() => {
  worker.events.removeAllListeners();
});

function countLibraryReads() {
  let reads = 0;
  worker.events.on("request:start", ({ request }) => {
    if (request.method === "GET" && new URL(request.url).pathname === "/api/v1/library") {
      reads += 1;
    }
  });
  return () => reads;
}

const menu = () => page.getByRole("button", { name: "The Quiet Orchard actions" });

async function open(item: string) {
  await userEvent.click(menu());
  await userEvent.click(page.getByRole("menuitem", { name: item }));
}

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
      /^Grab of The Quiet OrchardImportedFound by Search .*ClientBuilt-in client on home/u,
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

test("a cancel the server refuses keeps the dialog open with its sentence", async () => {
  const busy =
    "This Grab is waiting for Den to come back online. You can cancel it after it imports its file.";
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
      cause: "An Admin canceled this Grab.",
      recovery: "Search again to pick another release.",
      message: "Ada Okafor canceled this Grab.",
      needsYou: false,
      current: true,
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

test("a failure whose detail says only its cause shows no Detail row", async () => {
  const failed: GrabsGrab = {
    ...manual,
    stage: "failed",
    failure: { ...fetchFailure, message: fetchFailure.cause },
    actions: { cancel: false, blocklistable: false, sendTo: false },
  };
  renderApp({ path: "/activity", handlers: grabServer(failed).handlers });
  await expect
    .element(page.getByRole("article", { name: "The Quiet Orchard" }))
    .toMatchTextContent(
      /CauseSlipStream couldn't download the release's \.torrent file or magnet link from the Indexer\.RecoverySearch again to pick another release\.ReleaseThe\.Quiet\.Orchard\.2021\.1080p\.WEB-DL$/u,
    );
});

test("Cancel on a downloading Grab names it once, lists only the other Grabs its torrent's removal cancels and reads no Library", async () => {
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
  const { queryClient } = renderApp({
    path: "/activity",
    handlers: [actions.handler, ...grabServer(grab).handlers],
  });
  await expect.element(menu()).toBeVisible();
  queryClient.removeQueries({ queryKey: getListLibraryQueryKey() });
  const libraryReads = countLibraryReads();
  await open("Cancel Grab");
  const cancel = page.getByRole("dialog", { name: "Cancel this Grab?" });
  await expect
    .element(cancel)
    .toMatchTextContent(
      "It also cancels another Grab still downloading this torrent, and SlipStream searches for a release again:",
    );
  expect(cancel.getByText(downloading.releaseTitle).elements()).toHaveLength(1);
  await expect.element(cancel.getByText(other.release)).toBeVisible();
  expect(libraryReads()).toBe(0);
});
