import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderAction } from "../test/actions";
import { batch, moveUnit } from "../test/moves";
import { BatchRow } from "./batch-row";

const failed = moveUnit(2, {
  phase: "failed",
  words: "Film 2.mkv is gone from home. Scan the Films library root, then retry.",
  failure: {
    code: "source_gone",
    words: "Film 2.mkv is gone from home. Scan the Films library root, then retry.",
  },
});
const copying = moveUnit(1, { notes: ["1 file on home changed after this Move was planned."] });

function batchServer(sent: string[]) {
  return [
    http.get("*/api/v1/moves/:batchId", () =>
      HttpResponse.json({ batch: batch(), units: [copying, failed] }),
    ),
    http.post("*/api/v1/moves/:batchId/cancel", ({ request }) => {
      sent.push(new URL(request.url).pathname);
      return HttpResponse.json({
        canceled: 2,
        switched: 1,
        words: "Canceled 2 Moves. 1 film already switched stays.",
      });
    }),
    http.post("*/api/v1/moves/units/:unitId/retry", ({ request }) => {
      sent.push(new URL(request.url).pathname);
      return HttpResponse.json({ ...failed, phase: "queued", failure: null });
    }),
  ];
}

test("a batch row cancels the units that haven't switched", async () => {
  const sent: string[] = [];
  await renderAction(<BatchRow batch={batch()} />, batchServer(sent));

  await expect.element(page.getByText("1 of 3 films switched")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Move back…" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Cancel the remaining 2" }));
  await expect.poll(() => sent).toEqual([`/api/v1/moves/${batch().id}/cancel`]);
});

test("a batch row expands to each unit, its failures first with Retry", async () => {
  const sent: string[] = [];
  await renderAction(<BatchRow batch={batch()} />, batchServer(sent));

  await userEvent.click(page.getByRole("button", { name: "Details" }));
  const band = page.getByRole("region", { name: "Needs you" });
  await expect.element(band.getByText("Film 2", { exact: true })).toBeVisible();
  await expect.element(page.getByRole("listitem").getByText("The Quiet Orchard")).toBeVisible();

  await userEvent.click(band.getByRole("button", { name: "Retry" }));
  await expect.poll(() => sent).toEqual([`/api/v1/moves/units/${failed.id}/retry`]);
});

test("a finished batch offers Move back and no cancel", async () => {
  await renderAction(
    <BatchRow
      batch={batch({
        counts: { done: 3 },
        words: "3 films moved",
        canCancel: false,
        canMoveBack: true,
        finishedAt: "2026-11-01T10:00:00Z",
      })}
    />,
    batchServer([]),
  );

  await expect.element(page.getByRole("button", { name: "Move back…" })).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: /^Cancel the remaining/u }))
    .not.toBeInTheDocument();
});
