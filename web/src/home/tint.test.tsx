import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloading } from "../test/grabs";
import {
  bareWork,
  harvestWork,
  homeWork,
  job,
  jobRow,
  orchardWork,
  renderHome,
  summaryOf,
} from "../test/home";

const backdropNow = () =>
  document.querySelector<HTMLElement>('[data-ambient-backdrop="current"]')?.style.backgroundImage ??
  "none";

/** Runs the page's timers ms on, then lets them run on their own again. */
async function after(ms: number) {
  await vi.advanceTimersByTimeAsync(ms);
  vi.useRealTimers();
}

test("the top Job whose Work hasn't failed tints Home, and a hovered row tints after a pause", async () => {
  const fallen = homeWork("Fallen Orchard", { backdrop: "white", failed: true });
  renderHome(
    summaryOf({
      jobs: [
        job(downloading, fallen),
        job(downloading, orchardWork),
        job(downloading, harvestWork),
      ],
    }),
  );
  await expect.element(jobRow("Orchard Harvest")).toBeVisible();
  await expect.poll(backdropNow).toContain("/artwork/red.png?size=large");
  onTestFinished(() => {
    vi.useRealTimers();
  });
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  await userEvent.hover(jobRow("Orchard Harvest"));
  await vi.advanceTimersByTimeAsync(89);
  expect(backdropNow()).toContain("/artwork/red.png");
  await after(1);
  await expect.poll(backdropNow).toContain("/artwork/two-tone.png?size=large");
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  await userEvent.hover(page.getByRole("heading", { name: "Home", level: 1 }));
  await vi.advanceTimersByTimeAsync(449);
  expect(backdropNow()).toContain("/artwork/two-tone.png");
  await after(1);
  await expect.poll(backdropNow).toContain("/artwork/red.png?size=large");
});

test("Home stays neutral while the top Work has no backdrop, and with no Jobs", async () => {
  const { server } = renderHome(
    summaryOf({ jobs: [job(downloading, bareWork), job(downloading, orchardWork)] }),
  );
  await expect.element(jobRow("Bare Branches")).toBeVisible();
  expect(backdropNow()).toBe("none");
  server.become(summaryOf());
  server.clients[0]?.send({ event: "imports.changed", data: { rootIds: [] } });
  await expect.element(page.getByText(/^No jobs are moving/u)).toBeVisible();
  expect(backdropNow()).toBe("none");
});
