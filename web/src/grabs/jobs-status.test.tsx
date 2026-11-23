import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { downloading, failedImport, imported, waitingToFetch } from "../test/grabs";
import { renderKit } from "../test/render-kit";
import { JobsStatus } from "./jobs-status";
import { isFailed } from "./stages";

const grabs = [waitingToFetch, downloading, imported, failedImport];

test.each([
  { moves: 0, text: "2 in progress · 1 failure · 1 imported" },
  { moves: 1, text: "3 in progress · 1 failure · 1 imported" },
])("the jobs in progress are the Grabs in flight and the $moves Moves running", async ({ moves, text }) => {
  renderKit(
    <JobsStatus grabs={grabs} failed={grabs.filter((grab) => isFailed(grab))} moves={moves} />,
  );
  await expect.element(page.getByRole("paragraph")).toHaveTextContent(text);
});
