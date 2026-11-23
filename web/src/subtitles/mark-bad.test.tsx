import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../test/handlers";
import { met, openOrchard as open, subtitleRow, uhdSubtitles } from "../test/subtitles-work";

test("Mark bad on an uploaded subtitle says the Admin's own file is deleted, and shows a refusal in the dialog", async () => {
  open({
    work: { files: [uhdSubtitles([met("en", { source: "upload", provider: null, final: true })])] },
    markBad: () =>
      problem(
        404,
        "subtitle_not_found",
        "The file has no subtitle in that language",
        "This file has no subtitle in that language to mark bad.",
      ),
  });
  await userEvent.click(
    subtitleRow("English").getByRole("button", { name: "More for English subtitles" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Mark bad…" }));
  const dialog = page.getByRole("dialog", {
    name: "Mark The Quiet Orchard (2021) - 2160p.en.srt bad?",
  });
  await expect
    .element(
      dialog.getByText(
        "SlipStream didn't download it, so this deletes your file and can't be undone. SlipStream then searches for another English subtitle.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Mark bad" }));
  await expect
    .element(dialog.getByText("This file has no subtitle in that language to mark bad."))
    .toBeVisible();
});
