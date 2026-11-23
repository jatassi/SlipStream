import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SubtitlesWantView } from "../../api/gen/model";
import {
  hdSubtitles,
  hoursFromNow,
  met,
  missing,
  notifications,
  openOrchard,
  subtitleRow,
  tomorrowAtNoon,
  uhdSubtitles,
} from "../../test/subtitles-work";
import { asPhone } from "../../test/viewport";

const files = () => page.getByRole("list", { name: "Files", exact: true });
const fileRow = (name: string) => files().getByRole("listitem").filter({ hasText: name }).first();
const uhdRow = () => fileRow("2160p.mkv");
const hdRow = () => fileRow("1080p.mkv");

const notNeeded = (code: string, why: "audio_in_language" | "full_covers"): SubtitlesWantView => ({
  ...missing(code, "never_searched"),
  state: { kind: "not_needed", why, missing: null, next: null, sidecar: null, track: null },
});

const embedded = (code: string, track: number): SubtitlesWantView => ({
  ...missing(code, "never_searched"),
  state: { kind: "embedded", track, why: null, missing: null, next: null, sidecar: null },
});

const timed = met("en", {
  final: true,
  fit: { offsetMs: 2400, ratio: 25 / 23.976, segments: 3, match: 78 },
});

const forced = {
  ...notNeeded("en-forced", "full_covers"),
  wanted: { ...timed.wanted, forced: true },
};

const open = openOrchard;

const everyState = {
  work: {
    files: [
      uhdSubtitles([
        timed,
        forced,
        notNeeded("fr", "audio_in_language"),
        missing("de", "nothing_found", {
          last: { at: hoursFromNow(-3), found: 0, bestMatch: null },
          next: tomorrowAtNoon(),
        }),
        missing("es", "none_passed", {
          last: { at: hoursFromNow(-3), found: 3, bestMatch: 41 },
          next: tomorrowAtNoon(),
        }),
        missing("it", "map_due"),
      ]),
      hdSubtitles([
        embedded("en", 0),
        met("nl", {
          name: "nl.hi.srt",
          source: "found",
          provider: null,
          release: null,
          final: true,
          hindiWarning: true,
        }),
        missing("pt-BR", "no_holder"),
      ]),
    ],
  },
};

test("each file shows a row per Wanted Language with its state and one sentence for it", async () => {
  open(everyState);

  const lines = {
    "English, Met":
      "From SubDL. A person made it for this file's Release group, so Subtitle search doesn't look for a better one.",
    "English, forced, Not needed":
      "Not needed. The full English subtitle includes the forced lines.",
    "French, Not needed": "Not needed. The file has French audio.",
    "German, Missing":
      /^No subtitle was found 3 hours ago\. SlipStream searches again tomorrow at \d{1,2}:\d\d [AP]M\.$/u,
    "Spanish, Missing":
      /^3 subtitles were found, and none matched this file's timing\. SlipStream searches again tomorrow at \d{1,2}:\d\d [AP]M, or you can upload one\.$/u,
    "Italian, Waiting": "Searching starts once SlipStream has scanned this file's audio.",
  };
  const rows = uhdRow().getByRole("list", { name: "Subtitles" }).getByRole("listitem");
  for (const [index, [chip, line]] of Object.entries(lines).entries()) {
    // oxlint-disable-next-line no-await-in-loop -- each row is checked on the same page
    await expect.element(rows.nth(index).getByText(chip, { exact: true })).toBeInTheDocument();
    // oxlint-disable-next-line no-await-in-loop -- each row is checked on the same page
    await expect.element(rows.nth(index).getByText(line)).toBeVisible();
  }

  await expect.element(hdRow().getByText("Embedded in the file.")).toBeVisible();
  await expect
    .element(
      hdRow().getByText(
        "Found beside the file. SlipStream hasn't checked its timing. Jellyfin reads .hi as Hindi. Rename the file to .nl.sdh.srt to fix that.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      hdRow().getByText(
        "No Node has the Subtitles Role, so SlipStream can't download or time subtitles. Assign it on the Nodes page.",
      ),
    )
    .toBeVisible();
  const portuguese = hdRow().getByRole("listitem").filter({ hasText: "Brazilian Portuguese" });
  await expect.element(portuguese.getByRole("button", { name: "Search" })).toBeDisabled();
  await expect.element(portuguese.getByRole("button", { name: "Manual search" })).toBeDisabled();
  await expect.element(portuguese.getByRole("button", { name: "Upload" })).toBeDisabled();
});

test("a timed subtitle's Details name it and give its Timing sync numbers", async () => {
  open({ work: { files: [uhdSubtitles([timed])] } });
  const row = subtitleRow("English");
  await userEvent.click(row.getByRole("button", { name: "Details" }));
  for (const fact of [
    "The Quiet Orchard (2021) - 2160p.en.srt",
    "The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX-FLUX",
    "+2.4 s",
    "25 to 23.976 fps",
    "2",
    "78%",
  ]) {
    // oxlint-disable-next-line no-await-in-loop -- each fact is checked on the same open Details
    await expect.element(row.getByText(fact, { exact: true })).toBeVisible();
  }
  await expect
    .element(
      row.getByRole("button", { name: /^Splits found: Places where the subtitle's timing jumps/u }),
    )
    .toBeVisible();
});

test("the file's Details list its embedded tracks and the subtitle files beside it", async () => {
  open({
    work: {
      files: [
        uhdSubtitles([embedded("en", 0), met("de", { name: "de.srt" })], {
          embedded: [
            {
              stream: 0,
              codec: "subrip",
              language: "en",
              forced: false,
              image: false,
              ignored: false,
              meets: "en",
            },
            {
              stream: 1,
              codec: "hdmv_pgs_subtitle",
              language: "fr",
              forced: true,
              image: true,
              ignored: false,
              meets: null,
            },
          ],
          companions: [{ name: "sv.srt", path: "movies/x.sv.srt", size: 1200 }],
        }),
      ],
    },
  });
  await userEvent.click(uhdRow().getByRole("button", { name: "Details" }).first());
  const tracks = uhdRow().getByRole("list", { name: "Embedded subtitles" });
  await expect.element(tracks.getByText("English · subrip")).toBeVisible();
  await expect.element(tracks.getByText("Counts for English.")).toBeVisible();
  await expect.element(tracks.getByText(/^An image subtitle/u)).toBeVisible();
  const beside = uhdRow().getByRole("list", { name: "Subtitle files beside this video" });
  await expect.element(beside.getByText("The Quiet Orchard (2021) - 2160p.de.srt")).toBeVisible();
  await expect.element(beside.getByText(/SubDL.+German/u)).toBeVisible();
  await expect.element(beside.getByText("The Quiet Orchard (2021) - 2160p.sv.srt")).toBeVisible();
  await expect.element(beside.getByText(/^Not used/u)).toBeVisible();
});

test("Don't count this track stops an embedded track from counting, and the want then reads missing", async () => {
  const world = open({ work: { files: [uhdSubtitles([embedded("en", 0)])] } });
  await userEvent.click(
    subtitleRow("English").getByRole("button", { name: "More for English subtitles" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Don't count this track" }));
  await expect.poll(() => world.calls).toEqual(["ignore 0"]);

  world.change({ files: [uhdSubtitles([missing("en", "never_searched")])] });
  await expect.element(subtitleRow("English").getByText(/next Subtitle search/u)).toBeVisible();
});

test("Search starts a search for the subtitle and says so in a toast", async () => {
  const world = open({ work: { files: [uhdSubtitles([missing("en", "never_searched")])] } });
  await userEvent.click(subtitleRow("English").getByRole("button", { name: "Search" }));
  await expect.element(notifications().getByText(/English subtitle/u)).toBeVisible();
  expect(world.calls).toEqual(["search"]);
});

test("on phone a subtitle row's actions open as an action sheet", async () => {
  await asPhone();
  open({ work: { files: [uhdSubtitles([met("en")])] } });
  await userEvent.click(
    subtitleRow("English").getByRole("button", { name: "More for English subtitles" }),
  );
  const sheet = page.getByRole("dialog", { name: "More for English subtitles" });
  for (const action of ["Manual search", "Upload", "Mark bad…"]) {
    // oxlint-disable-next-line no-await-in-loop -- each action is checked on the same open sheet
    await expect.element(sheet.getByRole("button", { name: action })).toBeVisible();
  }
});
