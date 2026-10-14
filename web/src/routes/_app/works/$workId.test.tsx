import { HttpResponse, http } from "msw";
import { expect, onTestFinished, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloading, imported } from "../../../test/grabs";
import { problem } from "../../../test/handlers";
import { orchardId } from "../../../test/library";
import { renderApp } from "../../../test/render-app";
import { filled, grabbed, searching, workServer } from "../../../test/work-server";

const row = () => page.getByRole("row", { name: /^The Quiet Orchard /u });

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
  await expect.element(page.getByText("No files yet.")).toBeVisible();
  await expect.element(page.getByText("No searches yet.")).toBeVisible();
  await expect.element(page.getByText("No Grabs yet.")).toBeVisible();
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
      "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv 1920×1080",
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

test("Search starts a search, and the run and its Grab show without a reload", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });

  await userEvent.click(page.getByRole("button", { name: "Search" }));
  await expect.poll(() => server.searched).toEqual([orchardId]);

  server.searchRuns(searching);
  const searches = page.getByRole("list", { name: "Searches" });
  await expect.element(searches).toHaveTextContent("Searching");

  server.searchRuns(grabbed);
  await expect.element(searches).toHaveTextContent("Grabbed · 2 releases");

  server.grabMoves(downloading);
  const grabs = page.getByRole("list", { name: "Grabs" });
  await expect.element(grabs).toHaveTextContent("The.Quiet.Orchard.2021.1080p.WEB-DLDownloading");
  await expect
    .element(
      page.getByRole("progressbar", { name: "Downloading The.Quiet.Orchard.2021.1080p.WEB-DL" }),
    )
    .toHaveAttribute("value", "183500800");

  server.grabMoves(imported, filled);
  await expect.element(grabs).toHaveTextContent("The.Quiet.Orchard.2021.1080p.WEB-DLImported");
  await expect
    .element(row())
    .toHaveTextContent("The Quiet OrchardMetThe Quiet Orchard (2021) - 1080p.mkvDownloading");
});

test("Search is disabled while its request is pending", async () => {
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
  const search = page.getByRole("button", { name: "Search" });
  await userEvent.click(search);
  await expect.element(search).toBeDisabled();
});

test("a refused Search shows the server's reason", async () => {
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
  await userEvent.click(page.getByRole("button", { name: "Search" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("Add an Indexer before searching");
  await expect.element(page.getByRole("button", { name: "Search" })).toBeEnabled();
});
