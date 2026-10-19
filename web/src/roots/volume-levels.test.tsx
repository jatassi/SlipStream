import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { fillingVolume, healthServer, reportOf } from "../test/health";
import { gib, goneLevels, mediaLevels } from "../test/health-levels";
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

async function openLevels() {
  await userEvent.click(volumes().getByRole("button", { name: "Actions for /srv/media" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Set warning levels" }));
  return page.getByRole("dialog", { name: "Warning levels for /srv/media" });
}

test("a volume with a Storage warning says Warning and fills its bar in the warning colour", async () => {
  openWith("warn");
  const chip = volumes().getByText("Warning", { exact: true });
  await expect.element(chip).toBeVisible();
  const bar = volumes().getByRole("meter", { name: "/srv/media used" }).element();
  const fill = bar.querySelector("[class*=rounded-capsule] > *");
  expect(fill && getComputedStyle(fill).backgroundColor).toBe(
    getComputedStyle(chip.element()).backgroundColor,
  );
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

test("Set warning levels gives a volume its own warning level, and it says so", async () => {
  const health = openWith();
  const dialog = await openLevels();
  await expect
    .element(
      dialog.getByText("/srv/media on home follows the global levels until you change one here."),
    )
    .toBeVisible();
  await userEvent.fill(
    dialog.getByRole("textbox", { name: "Warn when free space is below" }),
    "300",
  );
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));
  await expect.element(dialog.getByText("/srv/media on home has its own levels.")).toBeVisible();
  expect(health.sent.volumes).toEqual([
    { nodeId: homeId, mount: "/srv/media", warnBelow: { amount: 300 * gib, unit: "bytes" } },
  ]);
  await userEvent.click(dialog.getByRole("button", { name: "Done" }));
  await expect.element(volumes().getByText("Own levels")).toBeVisible();
});

test("Follow the global levels resets both of a volume's own levels", async () => {
  const health = openWith();
  health.state.volumes = [
    {
      ...mediaLevels,
      warnBelow: {
        ...mediaLevels.warnBelow,
        isDefault: false,
        value: { amount: 10, unit: "percent" },
      },
    },
  ];
  const dialog = await openLevels();
  await expect.element(dialog.getByText("/srv/media on home has its own levels.")).toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Follow the global levels" }));
  await expect
    .element(
      dialog.getByText("/srv/media on home follows the global levels until you change one here."),
    )
    .toBeVisible();
  expect(health.sent.volumes).toEqual([
    { nodeId: homeId, mount: "/srv/media", warnBelow: null, errorBelow: null },
  ]);
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
