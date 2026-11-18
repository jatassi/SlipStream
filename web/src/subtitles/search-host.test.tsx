import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesAttempt, SubtitlesCandidate } from "../api/gen/model";
import { problem } from "../test/handlers";
import { orchardId } from "../test/library";
import { renderApp } from "../test/render-app";
import { scannedFiles } from "../test/scanned-files";
import {
  attempting,
  missing,
  started,
  subtitlesServer,
  uhdSubtitles,
} from "../test/subtitles-work";
import { asPhone } from "../test/viewport";
import { workServer } from "../test/work-server";

const title = "English subtitle for The Quiet Orchard (2021) - 2160p.mkv";
const notifications = () => page.getByRole("region", { name: "Messages" });

const candidate = (key: string, change: Partial<SubtitlesCandidate> = {}): SubtitlesCandidate => ({
  key,
  provider: "subdl",
  release: `The.Quiet.Orchard.2021.2160p.${key}`,
  reasons: [],
  disabled: false,
  tried: null,
  ...change,
});

function open(world: Omit<Parameters<typeof subtitlesServer>[0], "work">) {
  const subtitles = subtitlesServer({
    work: { files: [uhdSubtitles([missing("en", "nothing_found")])] },
    ...world,
  });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [...subtitles.handlers, ...workServer({ files: scannedFiles }).handlers],
  });
  return subtitles;
}

const manualSearch = () =>
  page
    .getByRole("list", { name: "Subtitles" })
    .getByRole("listitem")
    .filter({ hasText: "English" })
    .getByRole("button", { name: "Manual search" });

test("Manual search opens a 940 dialog on desktop and says when this build can't search yet", async () => {
  const world = open({});
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await expect
    .element(
      dialog.getByText(
        "This build of SlipStream can't search for, download or upload subtitles yet.",
      ),
    )
    .toBeVisible();
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
  await expect
    .element(
      sheet.getByText(
        "This build of SlipStream can't search for, download or upload subtitles yet.",
      ),
    )
    .toBeVisible();
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
    .element(tried.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.image" }))
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
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best" }),
  );
  await expect.element(dialog.getByText("Downloading from SubDL…")).toBeVisible();
  expect(world.calls).toEqual(["candidates", "pick best"]);

  attempt = { ...attempt, step: "checking" };
  world.attemptMoves();
  await expect.element(dialog.getByText("Checking timing…")).toBeVisible();

  attempt = { ...attempt, step: "finished", outcome: { kind: "added", match: 82, threshold: 50 } };
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
      outcome: { kind: "failed_timing", match: 41, threshold: 50 },
      uploadId: null,
    }),
    candidates: () => ({ state: "done", outcomes: [], candidates: [candidate("best")] }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best" }),
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
