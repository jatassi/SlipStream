import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { ConversionsConversionList, ConversionsConversionRow } from "../api/gen/model";
import {
  getListConversionsMockHandler,
  getMoveConversionUpMockHandler,
} from "../api/gen/query/conversions/conversions.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "../test/anthology";
import { renderApp } from "../test/render-app";

const waiting = (id: string, title: string): ConversionsConversionRow => ({
  kind: "conversion",
  id,
  stage: "queued",
  lane: "new",
  work: { id: `${id}-work`, title, type: anthologies.id },
  items: [],
  target: { id: "720p-mobile", name: "720p mobile" },
  source: { fileId: `${id}-file`, name: `${title}.mkv`, targetId: "1080p", target: "1080p", edition: "any" },
  node: { id: "home", name: "home" },
  progress: null,
  pausedBy: null,
  wait: { code: "copying", node: { id: "home", name: "home" } },
  start: null,
  finish: null,
  failure: null,
  attempts: [],
});

const list: ConversionsConversionList = {
  counts: { converting: 0, paused: 0, waiting: 2, failed: 0, clearsAt: null },
  attention: [],
  converting: [],
  next: [{ lane: "new", rows: [waiting("heat", "Heat"), waiting("orchard", "The Quiet Orchard")] }],
  cursor: null,
};

function conversionsServer() {
  const bands: (string | null)[] = [];
  const movedUp: string[] = [];
  return {
    bands,
    movedUp,
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [] }),
      getListConversionsMockHandler(({ request }) => {
        bands.push(new URL(request.url).searchParams.get("band"));
        return list;
      }),
      getMoveConversionUpMockHandler(({ params }) => {
        movedUp.push(String(params["id"]));
        return waiting(String(params["id"]), "The Quiet Orchard");
      }),
    ],
  };
}

test("the band filter asks for one band, and Move up moves that Conversion", async () => {
  const server = conversionsServer();
  const { router } = renderApp({ path: "/activity/conversions", handlers: server.handlers });
  const lane = page.getByRole("list", { name: "New" });
  await expect.element(lane.getByRole("link", { name: "The Quiet Orchard" })).toBeVisible();
  expect(server.bands).toEqual([null]);

  await userEvent.click(
    page.getByRole("radiogroup", { name: "Show" }).getByRole("radio", { name: /Up next/u }),
  );
  await expect.poll(() => router.state.location.search).toMatchObject({ band: "next" });
  await expect.poll(() => server.bands).toEqual([null, "next"]);

  await userEvent.click(page.getByRole("button", { name: "Move up The Quiet Orchard" }));
  await expect.poll(() => server.movedUp).toEqual(["orchard"]);
});
