import { afterEach, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloading, waitingForSeedbox } from "../test/grabs";
import { fillingVolume, healthServer, reportOf } from "../test/health";
import {
  bareWork,
  countsAre,
  failedGrab,
  harvestWork,
  homeSentence,
  job,
  need,
  orchardWork,
  renderHome,
  summaryOf,
  waitOf,
} from "../test/home";
import { resize, restoreViewport } from "../test/viewport";

afterEach(restoreViewport);

test("the status sentence counts what moves, fails and waits, with Health's worst, each linked", async () => {
  const failed = need(failedGrab("2026-10-12T09:02:00Z"), harvestWork, null);
  renderHome(
    summaryOf({
      jobs: [job(downloading, orchardWork)],
      needs: [failed, need(waitingForSeedbox, bareWork, waitOf("node_away"))],
      today: { imported: 5, upgraded: 1 },
    }),
    [
      countsAre({ pendingImports: 1 }),
      ...healthServer(reportOf({ storage: [fillingVolume] })).handlers,
    ],
  );
  await expect
    .element(homeSentence())
    .toHaveTextContent("1 job running · 1 failure · 2 waiting for you · Storage warning");
  await expect
    .element(homeSentence().getByRole("link", { name: "1 failure" }))
    .toHaveAttribute("href", `#failed-${failed.grab.id}`);
  await expect
    .element(page.getByRole("link", { name: "2 waiting for you" }))
    .toHaveAttribute("href", "#needs-you");
  await expect
    .element(page.getByRole("link", { name: "Storage warning" }))
    .toHaveAttribute("href", "#health");
  await expect.element(page.getByText("Today: 5 imported · 1 upgraded")).toBeVisible();
  await resize(1360, 900);
  await expect.element(page.getByText("Today: 5 imported · 1 upgraded")).not.toBeInTheDocument();
});

test("a quiet Home reads Nothing running with no totals line, and 100 failures read 100+", async () => {
  const { server } = renderHome(summaryOf());
  await expect.element(homeSentence()).toMatchTextContent(/^Nothing running$/u);
  await expect.element(page.getByText(/^Today:/u)).not.toBeInTheDocument();
  await expect.element(page.getByRole("heading", { name: "Needs you" })).not.toBeInTheDocument();
  server.become(
    summaryOf({
      needs: [need(failedGrab("2026-10-12T09:02:00Z"), harvestWork, null)],
      failed: 100,
    }),
  );
  server.clients[0]?.send({
    event: "grabs.changed",
    data: { grabId: downloading.id, workId: harvestWork.id, progress: false },
  });
  await expect.element(homeSentence()).toHaveTextContent("Nothing running · 100+ failures");
  await expect.element(page.getByRole("link", { name: "See all 100+ in Activity" })).toBeVisible();
});

test("while Reconciling, the jobs part says they are paused for it, with its definition", async () => {
  renderHome(summaryOf({ jobs: [job(downloading, orchardWork), job(downloading, harvestWork)] }), [
    countsAre({ reconciling: { nodes: [], rehome: false, roots: 0 } }),
  ]);
  await expect.element(homeSentence()).toMatchTextContent(/^2 jobs paused while Reconciling$/u);
  await userEvent.hover(homeSentence().getByRole("button", { name: "Reconciling" }));
  await expect
    .element(page.getByText(/^After a restore, SlipStream matches the library/u))
    .toBeVisible();
});
