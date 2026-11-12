import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { downloading, failedImport, imported, waitingToFetch } from "../test/grabs";
import { renderKit } from "../test/render-kit";
import { JobsStatus } from "./jobs-status";
import { isFailed } from "./stages";

const grabs = [waitingToFetch, downloading, imported, failedImport];

test("the jobs moving are the Grabs the server says are in flight", async () => {
  renderKit(<JobsStatus grabs={grabs} failed={grabs.filter((grab) => isFailed(grab))} />);
  await expect
    .element(page.getByRole("paragraph"))
    .toHaveTextContent("2 jobs moving · 1 failure · 1 imported");
});
