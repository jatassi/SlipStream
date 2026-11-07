import { expect, test } from "./fixtures.ts";
import {
  addSeries,
  episodeRow,
  expectMonitoring,
  lanternKeepers,
  levelHeader,
  openLevel,
  openMonitoring,
  seasons,
  setEpisodeMonitored,
  treeOf,
} from "./series.ts";

test.use({ scenario: "series" });
test.describe.configure({ timeout: 120_000 });

test("a season's switch monitors all its episodes, an episode's its own, and a mixed season says so", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "first" });
  await page.goto(`/works/${workId}`);
  const season2 = seasons(page).getByRole("switch", { name: "Monitor Season 2" });
  await expect(season2).toHaveAttribute("aria-checked", "false");
  await season2.click();
  await expect(season2).toHaveAttribute("aria-checked", "true");
  await expectMonitoring(page, world, workId, [6, 11]);
  await expect(levelHeader(page, "Season 2")).not.toContainText("Not monitored");
  await openLevel(page, "Season 2");
  await expect(episodeRow(page, "S02E03")).not.toContainText("Not monitored");

  await setEpisodeMonitored(page, "S01E02", false);
  await expectMonitoring(page, world, workId, [5, 11]);
  const season1 = seasons(page).getByRole("switch", { name: "Monitor Season 1" });
  await expect(season1).toHaveAttribute("aria-checked", "mixed");
  await expect(episodeRow(page, "S01E02")).toContainText("Not monitored");
  await expect(levelHeader(page, "Season 1")).toContainText("0 of 2 met");

  await page.reload();
  await expect(season1).toHaveAttribute("aria-checked", "mixed");
  await expectMonitoring(page, world, workId, [5, 11]);
});

test("the Monitoring dialog previews what starts and stops being monitored, then applies it", async ({
  adminPage: page,
  world,
}) => {
  const workId = await addSeries(world, lanternKeepers, { preset: "first" });
  await page.goto(`/works/${workId}`);
  const dialog = await openMonitoring(page, "The Lantern Keepers");
  const presets = [
    "All · 9 episodes",
    "Future · 0 episodes",
    "Missing · 9 episodes",
    "Existing · 0 episodes",
    "First season · 3 episodes",
    "Latest season · 3 episodes",
    "None · 0 episodes",
  ];
  await expect(dialog.getByRole("radio")).toHaveCount(presets.length);
  for (const [at, name] of presets.entries()) {
    // oxlint-disable-next-line no-await-in-loop -- one radio after another, in order
    await expect(dialog.getByRole("radio").nth(at)).toHaveAccessibleName(name);
  }
  await dialog.getByRole("radio", { name: "Latest season · 3 episodes" }).check();
  const change = dialog.getByText(/being monitored/u);
  await expect(change).toHaveText("3 episodes start being monitored and 3 stop.");
  await dialog.getByRole("switch", { name: "Include specials" }).click();
  await expect(change).toHaveText("5 episodes start being monitored and 3 stop.");
  await expect(dialog.getByRole("switch", { name: "Monitor new episodes" })).toBeChecked();
  await dialog.getByRole("button", { name: "Apply" }).click();
  await expect(dialog).toBeHidden();

  await expectMonitoring(page, world, workId, [5, 11]);
  await expect(levelHeader(page, "Season 1")).toContainText("Not monitored");
  await expect(seasons(page).getByRole("switch", { name: "Monitor Specials" })).toHaveAttribute(
    "aria-checked",
    "true",
  );
  expect(await treeOf(world, workId)).toMatchObject({ monitorNew: true, monitorAsides: true });
});
