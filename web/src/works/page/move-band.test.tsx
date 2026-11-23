import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { WorksWorkMove } from "../../api/gen/model";
import { renderKit } from "../../test/render-kit";
import { MoveBand } from "./move-band";

const move = (over: Partial<WorksWorkMove>): WorksWorkMove => ({
  unitId: "0199f0b2-2222-7000-8000-000000000001",
  batchId: "0199f0b1-1111-7000-8000-000000000001",
  phase: "transfer",
  toNodeId: "4a2c6e8f-1b3d-4f5a-9c7e-2d4f6a8b0c1e",
  toNodeName: "den",
  words: "Copying to den",
  ...over,
});

test.each([
  [move({ phase: "queued", words: "Moving to den" }), "Moving to den"],
  [move({}), "Moving to den · Copying to den"],
])("the Work band names the Move's step: %#", async (shown, words) => {
  renderKit(<MoveBand move={shown} />);
  await expect.element(page.getByRole("paragraph")).toHaveTextContent(words);
});

test("a failed Move's band links its failure to Activity", async () => {
  renderKit(
    <MoveBand
      move={move({
        phase: "failed",
        words: "Heat.mkv is gone from seedbox. Scan Movies (D:), then retry.",
      })}
    />,
  );
  await expect.element(page.getByRole("link")).toHaveAttribute("href", "/activity");
});

test("a canceled Move shows no band", async () => {
  renderKit(
    <>
      <MoveBand move={move({ phase: "canceled", words: "Move canceled" })} />
      <p>after</p>
    </>,
  );
  await expect.element(page.getByText("after")).toBeVisible();
  expect(page.getByText("Move canceled").elements()).toEqual([]);
});
