import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { HomeMoveNeed } from "../api/gen/model";
import { film, renderAction } from "../test/actions";
import { moveUnit } from "../test/moves";
import { MoveNeeds } from "./move-need";

const work = (n: number) => ({
  id: film(n).id,
  label: film(n).label,
  contentType: "cinema",
  failed: false,
  posterUrl: null,
  backdropUrl: null,
});

const waiting: HomeMoveNeed = {
  unitId: moveUnit(1).id,
  batchId: moveUnit(1).batchId,
  kind: "waiting",
  since: "2026-11-01T09:00:00Z",
  switchNow: true,
  words:
    "The Quiet Orchard has waited 1 hour to switch to den. Someone is playing it on a Media Server.",
  work: work(1),
};

const failed: HomeMoveNeed = {
  unitId: moveUnit(2).id,
  batchId: moveUnit(2).batchId,
  kind: "failed",
  switchNow: false,
  words: "Film 2 couldn't move to den. den didn't respond in time.",
  work: work(2),
};

function needServer(sent: string[]) {
  const answer = ({ request }: { request: Request }) => {
    sent.push(new URL(request.url).pathname);
    return HttpResponse.json(moveUnit(1));
  };
  return [
    http.post("*/api/v1/moves/units/:unitId/switch-now", answer),
    http.post("*/api/v1/moves/units/:unitId/retry", answer),
  ];
}

test("Switch now on a long switch wait asks first", async () => {
  const sent: string[] = [];
  await renderAction(<MoveNeeds needs={[waiting]} />, needServer(sent));

  await userEvent.click(page.getByRole("button", { name: "Switch now" }));
  expect(sent).toEqual([]);
  const confirm = page.getByRole("dialog", { name: /The Quiet Orchard/u });
  await userEvent.click(confirm.getByRole("button", { name: "Switch now" }));
  await expect.element(confirm).not.toBeInTheDocument();
  expect(sent).toEqual([`/api/v1/moves/units/${waiting.unitId}/switch-now`]);
});

test("a failed Move offers Retry", async () => {
  const sent: string[] = [];
  await renderAction(<MoveNeeds needs={[failed]} />, needServer(sent));

  await expect.element(page.getByRole("button", { name: "Retry" })).toBeVisible();
  expect(page.getByRole("button", { name: "Switch now" }).query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Retry" }));
  await expect.poll(() => sent).toEqual([`/api/v1/moves/units/${failed.unitId}/retry`]);
});
