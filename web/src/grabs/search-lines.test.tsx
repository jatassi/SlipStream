import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { waitingToFetch } from "../test/grabs";
import { orchardId } from "../test/library";
import { renderApp } from "../test/render-app";
import { everyOutcome, workServer } from "../test/work-server";

test("a search shows each Indexer's line in the server's words", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.searchRuns(everyOutcome);
  const searches = page.getByRole("list", { name: "Searches" });
  await expect.element(searches).toMatchTextContent(/^Found nothing acceptable1 Release found/u);
  expect(
    searches
      .getByRole("list", { name: "Indexers" })
      .getByRole("listitem")
      .all()
      .map((item) => item.element().textContent),
  ).toEqual([
    "AlderFound 1 Release",
    "BirchNo Releases",
    "CedarLimit reached until 14:05",
    "DuneLimit reached until 15:00",
    "ElmWaiting for seedbox",
    "FirFir isn't responding. SlipStream retries at 14:20.",
    "GaleStopped because Gale rejected the username or password.",
    "HazelSkipped because it has no movie categories",
    "IvyDidn't finish in time",
    "JuniperDeleted since this search",
  ]);
});

test("a Grab whose fetch waits on its Indexer says why in the server's words", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves(waitingToFetch);
  await expect
    .element(page.getByRole("table", { name: "Grabs" }))
    .toMatchTextContent(
      "Waiting to fetch the .torrent from Fixture Site. SlipStream retries at 14:20.",
    );
});
