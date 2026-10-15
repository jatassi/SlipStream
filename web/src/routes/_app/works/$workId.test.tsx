import { HttpResponse, http } from "msw";
import { expect, onTestFinished, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsGrab } from "../../../api/gen/model";
import { downloading, imported } from "../../../test/grabs";
import { problem } from "../../../test/handlers";
import { orchardId } from "../../../test/library";
import { renderApp } from "../../../test/render-app";
import { filled, grabbed, searching, workServer } from "../../../test/work-server";

const row = () => page.getByRole("row", { name: /^The Quiet Orchard /u });
const searchButton = () => page.getByRole("button", { name: "Search", exact: true });
const notifications = () => page.getByRole("region", { name: "Notifications" });

test("the Work page shows its fields and one cell per Target", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  expect(
    page
      .getByRole("term")
      .all()
      .map((term) => term.element().textContent),
  ).toEqual(["Year", "Released", "Runtime"]);
  expect(
    page
      .getByRole("definition")
      .all()
      .map((value) => value.element().textContent),
  ).toEqual(["2021", "2021-04-09", "1h 52m"]);
  expect(
    page
      .getByRole("columnheader")
      .all()
      .map((header) => header.element().textContent),
  ).toEqual(["Film", "1080p", "2160p"]);
  await expect.element(row()).toHaveTextContent("The Quiet OrchardMissingMissing");
  await expect
    .element(page.getByText("No files yet. A Grab that imports puts its file here."))
    .toBeVisible();
  await expect
    .element(page.getByText("No searches yet. Search looks for a release on every Indexer."))
    .toBeVisible();
  await expect
    .element(page.getByText("No Grabs yet. A search that finds a release grabs it."))
    .toBeVisible();
});

test("files.changed shows the new file without a reload", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect.element(row()).toHaveTextContent("The Quiet OrchardMissingMissing");

  server.fileArrives(filled);

  await expect
    .element(row())
    .toHaveTextContent("The Quiet OrchardMetThe Quiet Orchard (2021) - 1080p.mkvDownloading");
  await expect
    .element(page.getByRole("list", { name: "Files" }).getByRole("listitem"))
    .toHaveTextContent(
      "The Quiet Orchard (2021) - 1080p.mkvmovies/The Quiet Orchard (2021)1920×1080 · 700 MB",
    );
});

test("a Work that is gone says so", async () => {
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.get("*/api/v1/works/:workId", () =>
        problem(404, "work_not_found", "No Work has that id"),
      ),
    ],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("No Work has that id");
});

test("Search starts a search and says so, and the run and its Grab show without a reload", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });

  await userEvent.click(searchButton());
  await expect.poll(() => server.searched).toEqual([orchardId]);
  await expect.element(notifications().getByText("Search started")).toBeVisible();

  server.searchRuns(searching);
  const searches = page.getByRole("list", { name: "Searches" });
  await expect.element(searches).toMatchTextContent(/^Searching/u);

  server.searchRuns(grabbed);
  await expect.element(searches).toMatchTextContent(/^Grabbed a release2 releases found/u);

  server.grabMoves(downloading);
  const grabs = page.getByRole("table", { name: "Grabs" });
  await expect
    .element(grabs)
    .toMatchTextContent("The.Quiet.Orchard.2021.1080p.WEB-DLDownloading25%175 MB of 700 MB");
  await expect
    .element(
      page.getByRole("progressbar", { name: "Downloading The.Quiet.Orchard.2021.1080p.WEB-DL" }),
    )
    .toHaveAttribute("aria-valuenow", "25");

  server.grabMoves(imported, filled);
  await expect.element(grabs).toMatchTextContent("The.Quiet.Orchard.2021.1080p.WEB-DLImported");
  await expect
    .element(row())
    .toHaveTextContent("The Quiet OrchardMetThe Quiet Orchard (2021) - 1080p.mkvDownloading");
});

test("Search shows it is busy while its request is pending", async () => {
  const server = workServer();
  let release!: () => void;
  const released = new Promise<void>((resolve) => {
    release = resolve;
  });
  onTestFinished(release);
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.post("*/api/v1/works/:workId/search", async () => {
        await released;
        return HttpResponse.error();
      }),
      ...server.handlers,
    ],
  });
  await userEvent.click(searchButton());
  await expect.element(searchButton()).toHaveAttribute("aria-busy", "true");
});

test("a refused Search says the server's reason in a toast", async () => {
  const server = workServer();
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.post("*/api/v1/works/:workId/search", () =>
        problem(409, "no_indexer", "Add an Indexer before searching"),
      ),
      ...server.handlers,
    ],
  });
  await userEvent.click(searchButton());
  await expect
    .element(notifications().getByRole("dialog"))
    .toHaveTextContent("FailedAdd an Indexer before searching");
  await expect.element(searchButton()).not.toHaveAttribute("aria-busy");
});

const failedImport: GrabsGrab = {
  ...imported,
  stage: "failed",
  history: [...imported.history.slice(0, -1), { stage: "failed", at: "2026-10-12T09:03:02Z" }],
  failure: {
    stage: "importing",
    code: "import_failed",
    message: "place movies/The Quiet Orchard (2021): permission denied",
  },
};

test("a failed Grab shows under Needs you with its stage, cause and a way to search again", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect.element(row()).toHaveTextContent("The Quiet OrchardMissingMissing");

  server.grabMoves(failedImport);

  const needsYou = page.getByRole("region", { name: "Needs you" });
  const card = needsYou.getByRole("article", { name: "The Quiet Orchard" });
  await expect.element(card).toMatchTextContent(/^Failed at Import/u);
  await expect
    .element(card)
    .toMatchTextContent(
      "CauseSlipStream couldn't put the file in the library." +
        "ReleaseThe.Quiet.Orchard.2021.1080p.WEB-DL" +
        "Detailplace movies/The Quiet Orchard (2021): permission denied",
    );
  await expect
    .element(page.getByText("No Grabs yet. A search that finds a release grabs it."))
    .toBeVisible();

  await userEvent.click(card.getByRole("button", { name: "Search again" }));
  await expect.poll(() => server.searched).toEqual([orchardId]);
});
