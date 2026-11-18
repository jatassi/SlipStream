import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../test/handlers";
import { orchardId } from "../test/library";
import { renderApp } from "../test/render-app";
import { scannedFiles } from "../test/scanned-files";
import { attempting, missing, subtitlesServer, uhdSubtitles } from "../test/subtitles-work";
import { workServer } from "../test/work-server";

const title = "English subtitle for The Quiet Orchard (2021) - 2160p.mkv";
const notifications = () => page.getByRole("region", { name: "Messages" });
const cue = "1\n00:00:01,000 --> 00:00:02,000\nHello\n";

function open(world: Omit<Parameters<typeof subtitlesServer>[0], "work">) {
  const subtitles = subtitlesServer({
    work: { files: [uhdSubtitles([missing("en", "none_passed")])] },
    ...world,
  });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [...subtitles.handlers, ...workServer({ files: scannedFiles }).handlers],
  });
  return subtitles;
}

async function openUpload() {
  await userEvent.click(
    page
      .getByRole("list", { name: "Subtitles" })
      .getByRole("listitem")
      .filter({ hasText: "English" })
      .getByRole("button", { name: "Upload" }),
  );
  const dialog = page.getByRole("dialog", { name: title });
  await expect.element(dialog).toBeVisible();
  return dialog;
}

test("a file over 5 MB is refused before anything is sent", async () => {
  const world = open({});
  const dialog = await openUpload();
  await userEvent.upload(
    dialog.getByLabelText("Subtitle file"),
    new File([new Uint8Array((5 << 20) + 1)], "big.srt"),
  );
  await expect.element(dialog.getByText("Use a subtitle file of 5 MB or smaller.")).toBeVisible();
  expect(world.calls).toEqual([]);
});

test("a dropped file is sent in base64, and a refusal shows in the dialog", async () => {
  const world = open({
    upload: () =>
      problem(
        409,
        "audio_not_scanned",
        "SlipStream hasn't scanned the file's audio yet",
        "SlipStream checks a subtitle's timing against the file's audio and hasn't scanned it yet. Try again after the scan finishes.",
      ),
  });
  const dialog = await openUpload();
  const drop = new DataTransfer();
  drop.items.add(new File([cue], "Orchard.en.srt"));
  dialog
    .getByRole("group", { name: "Subtitle file" })
    .element()
    .dispatchEvent(new DragEvent("drop", { dataTransfer: drop, bubbles: true }));
  await expect
    .element(
      dialog.getByText(
        "SlipStream checks a subtitle's timing against the file's audio and hasn't scanned it yet. Try again after the scan finishes.",
      ),
    )
    .toBeVisible();
  expect(world.uploads).toEqual([{ name: "Orchard.en.srt", bytes: btoa(cue) }]);
});

test("an upload whose timing fails offers Keep anyway under Details, which keeps it", async () => {
  const world = open({
    upload: attempting,
    attempt: () => ({
      step: "finished",
      provider: null,
      outcome: { kind: "failed_timing", match: 38, threshold: 50, answer: null },
      uploadId: "2e4a6c8e-0a2c-4e6a-8c0e-2a4c6e8a0c2f",
    }),
  });
  const dialog = await openUpload();
  await userEvent.upload(dialog.getByLabelText("Subtitle file"), new File([cue], "Orchard.en.srt"));
  await expect
    .element(
      dialog.getByText(
        "This subtitle's timing doesn't match the file, so SlipStream didn't keep it. Pick another file, or open Details to keep it anyway.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Details" }));
  await expect
    .element(
      dialog.getByText(
        "Its timing match is 38%, and SlipStream keeps subtitles at 50% or more. If you keep it, its lines may show early or late.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Keep anyway" }));
  await expect.element(notifications().getByText("English subtitle added.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(world.calls).toEqual(["upload", "keep"]);
});
