import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { fillingVolume, healthServer, reportOf } from "../test/health";
import { goneLevels, mediaLevels } from "../test/health-levels";
import { homeId } from "../test/nodes";
import { serveRoots } from "../test/roots";
import { openRoots } from "../test/roots-page";

const volumes = () => page.getByRole("list", { name: "Volumes on home" });

function openWith(level: "warn" | "fail" | null = null) {
  serveRoots();
  const health = healthServer(reportOf(level === null ? {} : { storage: [fillingVolume] }));
  health.state.volumes = [{ ...mediaLevels, level }];
  openRoots({ handlers: health.handlers });
  return health;
}

test("a volume with a Storage warning says Warning", async () => {
  openWith("warn");
  await expect.element(volumes().getByText("Warning", { exact: true })).toBeVisible();
});

test("a volume with nothing wrong carries no level", async () => {
  openWith();
  await expect.element(volumes().getByText("1.2 TB free of 8.0 TB")).toBeVisible();
  expect(
    volumes()
      .getByText(/^(?:Warning|Error)$/u)
      .query(),
  ).toBeNull();
});

test("own levels whose volume no Node reports stay listed until removed", async () => {
  serveRoots();
  const health = healthServer(reportOf());
  health.state.volumes = [mediaLevels];
  health.state.notReporting = [goneLevels];
  openRoots({ handlers: health.handlers });
  const gone = page.getByRole("list", { name: "Volumes not reporting" });
  await expect
    .element(gone.getByRole("listitem"))
    .toHaveTextContent("D:\\ on homeWarning below 500 GB, error below the global levelRemove");
  await userEvent.click(gone.getByRole("button", { name: "Remove the levels for D:\\ on home" }));
  await expect.element(gone).not.toBeInTheDocument();
  expect(health.sent.removed).toEqual([`${homeId} D:\\`]);
});
