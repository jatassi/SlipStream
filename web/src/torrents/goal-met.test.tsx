import { http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { seedpoolId, trackersServer } from "../test/trackers";

const wouldAct = () => page.getByRole("list", { name: "Would act now" });
const waiting = () => page.getByRole("list", { name: "Goal met, waiting" });
const rowOf = (list: ReturnType<typeof wouldAct>, name: string) =>
  list.getByRole("listitem").filter({ hasText: name });

function previewAsked() {
  const asked: string[] = [];
  return {
    asked,
    handler: http.get("*/api/v1/torrents/goals/preview", ({ request }) => {
      asked.push(new URL(request.url).pathname);
    }),
  };
}

test("the Goal met tab leads with what the goals would do now, each kept copy and what waits", async () => {
  const preview = previewAsked();
  renderApp({
    path: "/activity/torrents?tab=goal-met",
    handlers: [preview.handler, ...liveServer().handlers, ...trackersServer().handlers],
  });
  await expect
    .element(rowOf(wouldAct(), "Low.Tide.2023.1080p.WEB-DL"))
    .toMatchTextContent("Low.Tide.2023.1080p.WEB-DLseedpool · Pause");
  await expect
    .element(rowOf(wouldAct(), "Elsewhere.2021.1080p.WEB-DL"))
    .toMatchTextContent("Elsewhere.2021.1080p.WEB-DLTorrentLeech · Remove");
  await expect
    .element(
      wouldAct().getByText(
        "SlipStream removes Hold.Fast.2024.1080p.WEB-DL and keeps its download copy, because its Grab didn't import.",
      ),
    )
    .toBeVisible();
  await expect
    .element(rowOf(waiting(), "Salt.Line.2022.1080p.WEB-DL"))
    .toMatchTextContent("Its goal waits until its Grab finishes importing.");
  expect(preview.asked).toEqual(["/api/v1/torrents/goals/preview"]);
});

test("a Tracker's Goal met tab shows only what the goals would do to its own torrents", async () => {
  renderApp({
    path: `/settings/trackers/${seedpoolId}?tab=goal-met`,
    handlers: [...liveServer().handlers, ...trackersServer().handlers],
  });
  await expect.element(rowOf(wouldAct(), "Low.Tide.2023.1080p.WEB-DL")).toBeVisible();
  expect(wouldAct().getByText("Elsewhere.2021.1080p.WEB-DL").query()).toBeNull();
});

test("with nothing for the goals to do the tab says so, and no other tab asks for the preview", async () => {
  const preview = previewAsked();
  const { router } = renderApp({
    path: "/activity/torrents?tab=all",
    handlers: [
      preview.handler,
      ...liveServer().handlers,
      ...trackersServer({ preview: { acts: [], kept: [], waits: [] } }).handlers,
    ],
  });
  await expect.element(page.getByRole("table", { name: "Torrents" })).toBeVisible();
  expect(preview.asked).toEqual([]);
  await router.navigate({ to: "/activity/torrents", search: { tab: "goal-met" } });
  await expect.element(page.getByText("No goal would act if Seeding goals ran now.")).toBeVisible();
  expect(page.getByRole("list", { name: "Goal met, waiting" }).query()).toBeNull();
});
