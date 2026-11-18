import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesAttempt } from "../api/gen/model";
import { problem } from "../test/handlers";
import { candidate, manualSearch, openMissing as open } from "../test/subtitle-candidates";
import { attempting, hoursFromNow, started } from "../test/subtitles-work";
import { asPhone } from "../test/viewport";

const title = "English subtitle for The Quiet Orchard (2021) - 2160p.mkv";
const notifications = () => page.getByRole("region", { name: "Messages" });

test("Manual search opens a 940 dialog on desktop and says when no provider found a subtitle", async () => {
  const world = open({});
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await expect.element(dialog.getByText("No subtitle was found for this file.")).toBeVisible();
  await expect.element(dialog).toHaveClass(/w-\[940px\]/u);
  expect(world.calls).toEqual(["candidates"]);
});

test("on phone Manual search is a full sheet", async () => {
  await asPhone();
  open({});
  await userEvent.click(
    page
      .getByRole("list", { name: "Subtitles" })
      .getByRole("button", { name: "More for English subtitles" }),
  );
  await userEvent.click(page.getByRole("button", { name: "Manual search" }));
  const sheet = page.getByRole("dialog", { name: title });
  await expect.element(sheet.getByText("No subtitle was found for this file.")).toBeVisible();
  await expect.element(sheet).not.toHaveClass(/w-\[940px\]/u);
});

test("an expired search says to search again", async () => {
  open({
    start: started,
    candidates: () =>
      problem(
        410,
        "subtitle_search_expired",
        "The subtitle search expired",
        "These results have expired. Search again.",
      ),
  });
  await userEvent.click(manualSearch());
  await expect
    .element(
      page
        .getByRole("dialog", { name: title })
        .getByText("These results have expired. Search again."),
    )
    .toBeVisible();
});

test("candidates rank with the first as Best, each step it meets checked, and tries before folded with what happened", async () => {
  open({
    start: started,
    candidates: () => ({
      state: "done",
      outcomes: [
        { provider: "subdl", kind: "ok", until: null },
        { provider: "subsource", kind: "not_found", until: null },
      ],
      candidates: [
        candidate("best", { reasons: ["made_by_person", "same_group"] }),
        candidate("next", { provider: "subsource", reasons: ["made_by_person"] }),
        candidate("image", {
          disabled: true,
          tried: { outcome: "image", match: null, at: new Date().toISOString() },
        }),
        candidate("failed", {
          tried: { outcome: "failed_timing", match: 41, at: new Date().toISOString() },
        }),
      ],
    }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  const ranked = dialog
    .getByRole("list", { name: "Candidates" })
    .getByRole("listitem")
    .filter({ hasText: "2160p" });
  await expect.element(ranked.nth(0).getByText("Best")).toBeVisible();
  await expect.element(ranked.nth(1).getByText("Best")).not.toBeInTheDocument();
  await expect
    .element(ranked.nth(0).getByRole("list", { name: "Ranking steps" }))
    .toHaveTextContent(
      "Made by a personSame Release groupSame source and resolution, doesn't holdHigher on your provider list, doesn't hold",
    );
  await expect.element(dialog.getByText("SubSource found nothing.")).toBeVisible();

  await userEvent.click(dialog.getByRole("button", { name: "Tried before" }));
  const tried = dialog.getByRole("list", { name: "Tried before" });
  await expect
    .element(tried.getByText("SubDL · SlipStream can't use image subtitles."))
    .toBeVisible();
  await expect
    .element(
      tried.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.image from SubDL" }),
    )
    .toBeDisabled();
  await expect
    .element(tried.getByText("SubDL · Its timing didn't match this file. · Timing match 41%"))
    .toBeVisible();
});

test("a pick shows each step as the attempt moves, then toasts and closes once the subtitle is added", async () => {
  let attempt: SubtitlesAttempt = {
    step: "downloading",
    provider: "subdl",
    outcome: null,
    uploadId: null,
  };
  const world = open({
    start: started,
    pick: attempting,
    attempt: () => attempt,
    candidates: () => ({ state: "done", outcomes: [], candidates: [candidate("best")] }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best from SubDL" }),
  );
  await expect.element(dialog.getByText("Downloading from SubDL…")).toBeVisible();
  expect(world.calls).toEqual(["candidates", "pick best"]);

  attempt = { ...attempt, step: "checking" };
  world.attemptMoves();
  await expect.element(dialog.getByText("Checking timing…")).toBeVisible();

  attempt = {
    ...attempt,
    step: "finished",
    outcome: { kind: "added", match: 82, threshold: 50, answer: null },
  };
  world.attemptMoves();
  await expect.element(notifications().getByText("English subtitle added.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
});

test("a pick whose timing fails stays open with the reason", async () => {
  const world = open({
    start: started,
    pick: attempting,
    attempt: () => ({
      step: "finished",
      provider: "subdl",
      outcome: { kind: "failed_timing", match: 41, threshold: 50, answer: null },
      uploadId: null,
    }),
    candidates: () => ({ state: "done", outcomes: [], candidates: [candidate("best")] }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best from SubDL" }),
  );
  await expect
    .element(
      dialog.getByText(
        "Its timing didn't match this file. Its timing match is 41%, and SlipStream keeps subtitles at 50% or more.",
      ),
    )
    .toBeVisible();
  expect(world.calls).toEqual(["candidates", "pick best"]);
});

test("a pick the provider didn't serve says why and when it serves again, and leaves the others to pick", async () => {
  const world = open({
    attempt: () => ({
      step: "finished",
      provider: "subdl",
      outcome: {
        kind: "unfetched",
        match: null,
        threshold: null,
        answer: { provider: "subdl", kind: "limited", until: hoursFromNow(2) },
      },
      uploadId: null,
    }),
    candidates: () => ({
      state: "done",
      outcomes: [],
      candidates: [candidate("best"), candidate("next", { provider: "subsource" })],
    }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best from SubDL" }),
  );
  await expect
    .element(
      dialog.getByText(
        /^SubDL is out of downloads\. SlipStream searches it again (at|tonight|tomorrow).+\. Pick another subtitle\.$/u,
      ),
    )
    .toBeVisible();
  await expect
    .element(
      dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.next from SubSource" }),
    )
    .toBeEnabled();
  expect(world.calls).toEqual(["candidates", "pick best"]);
});
