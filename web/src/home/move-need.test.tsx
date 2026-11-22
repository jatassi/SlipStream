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

test("Switch now on a long switch wait asks first, naming what it gives up", async () => {
  const sent: string[] = [];
  await renderAction(<MoveNeeds needs={[waiting]} />, needServer(sent));

  await expect.element(page.getByText(waiting.words)).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Switch now" }));
  const confirm = page.getByRole("dialog", { name: "Switch The Quiet Orchard now?" });
  await expect
    .element(
      confirm.getByText(
        "Someone may still be playing it. Their playback may stop when the old files are deleted. A Media Server that hasn't saved it yet loses who watched The Quiet Orchard, their resume positions and ratings, and its collections.",
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm.getByRole("button", { name: "Switch now" }));
  await expect.element(confirm).not.toBeInTheDocument();
  expect(sent).toEqual([`/api/v1/moves/units/${waiting.unitId}/switch-now`]);
});

test("a failed Move offers Retry", async () => {
  const sent: string[] = [];
  await renderAction(<MoveNeeds needs={[failed]} />, needServer(sent));

  await expect.element(page.getByText(failed.words)).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Switch now" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Retry" }));
  await expect.poll(() => sent).toEqual([`/api/v1/moves/units/${failed.unitId}/retry`]);
});
