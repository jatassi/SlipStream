import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloading } from "../test/grabs";
import { job, orchardWork, renderHome, summaryOf } from "../test/home";
import { actionsServer, torrent } from "../test/torrents";

const unknownA = "3b000000-0000-4000-8000-000000000001";
const unknownB = "3b000000-0000-4000-8000-000000000002";

const orchardJob = job(
  downloading,
  orchardWork,
  torrent({
    state: "downloading",
    tabs: ["downloading", "all"],
    workId: orchardWork.id,
    name: "The.Quiet.Orchard.2021.1080p.WEB-DL",
  }),
);

async function removeFromJobs() {
  await userEvent.click(page.getByRole("button", { name: "The Quiet Orchard actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Remove" }));
  return page.getByRole("dialog", { name: "Remove The Quiet Orchard?" });
}

test("a remove from Jobs calls Works Home doesn't show Another Work, once in a sentence", async () => {
  const actions = actionsServer({
    preview: {
      cancels: [{ grabId: downloading.id, workId: unknownA, release: "Other.Release.1080p" }],
      stopSeeding: [unknownA, orchardWork.id, unknownB],
      frees: 0,
      linked: false,
      hitAndRun: [],
    },
  });
  renderHome(summaryOf({ jobs: [orchardJob] }), [actions.handler]);
  const dialog = await removeFromJobs();
  await expect
    .element(dialog)
    .toMatchTextContent(
      /Another WorkOther\.Release\.1080pAnother Work and The Quiet Orchard stop seeding\./u,
    );
});

test("a remove from Jobs refused for an import names the torrent's Work", async () => {
  const actions = actionsServer({ refuse: () => ({ code: "importing", workId: unknownA }) });
  renderHome(summaryOf({ jobs: [orchardJob] }), [actions.handler]);
  const dialog = await removeFromJobs();
  await expect
    .element(dialog)
    .toMatchTextContent(
      "SlipStream is finishing the import of The Quiet Orchard. Try again once it's done.",
    );
});
