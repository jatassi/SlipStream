import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesWantView } from "../../api/gen/model";
import { scannedFiles } from "../../test/scanned-files";
import { met, missing, openOrchard as open, uhdSubtitles } from "../../test/subtitles-work";

const episode = (id: string, season: number, number: number, wants: SubtitlesWantView[]) => ({
  ...uhdSubtitles(wants),
  fileId: id,
  season,
  episodes: [number],
});

test("a series folds its files per season with the first season missing a subtitle open, and chips open into rows", async () => {
  const [first, second, third] = scannedFiles.files.map(({ id }) => id);
  open({
    work: {
      files: [
        episode(first ?? "", 1, 1, [met("en")]),
        episode(second ?? "", 2, 1, [missing("en", "nothing_found")]),
        episode(third ?? "", 2, 2, [met("en")]),
      ],
    },
  });
  const seasonOne = page.getByRole("region", { name: "Season 1" });
  const seasonTwo = page.getByRole("region", { name: "Season 2" });
  await expect
    .element(seasonTwo.getByRole("button", { name: /^Season 2/u }))
    .toHaveAttribute("aria-expanded", "true");
  await expect
    .element(seasonOne.getByRole("button", { name: /^Season 1/u }))
    .toHaveAttribute("aria-expanded", "false");
  await expect.element(seasonTwo.getByText("1 file · 1 missing subtitle")).toBeVisible();

  const chips = seasonTwo.getByRole("button", { name: "Subtitles: English, Missing" });
  await expect.element(chips).toHaveAttribute("aria-expanded", "false");
  await userEvent.click(chips);
  await expect.element(seasonTwo.getByText("No subtitle was found.")).toBeVisible();

  await userEvent.click(seasonOne.getByRole("button", { name: /^Season 1/u }));
  await expect
    .element(seasonOne.getByRole("button", { name: "Subtitles: English, Met" }))
    .toBeVisible();
});
