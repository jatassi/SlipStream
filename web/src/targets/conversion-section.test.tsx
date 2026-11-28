import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsTarget } from "../api/gen/model";
import { changeOp, noImpact } from "../test/impact";
import { renderApp } from "../test/render-app";
import { keeper, targetHandlers } from "../test/targets";

const refusals = [
  {
    problem: { code: "codec_filtered", field: "Codec", facts: { codec: "h264" } },
    words:
      "These settings make H.264 files, and this Target's video codec filter refuses them. Change the codec or the Target's filters.",
  },
  {
    problem: { code: "codec_filtered", field: "Audio", facts: { audio: "stereo_aac" } },
    words:
      "These settings make stereo AAC audio, and this Target's audio filters refuse it. Change the audio or the Target's filters.",
  },
  {
    problem: { code: "quality_unranked", field: "MaxHeight", facts: { height: "1080" } },
    words:
      "This Target accepts no 1080p quality, so it refuses every file these settings make. Change the largest resolution, or accept a 1080p quality.",
  },
  {
    problem: {
      code: "outside_size",
      field: "Kbps",
      facts: { kbps: "900", minKbps: "1398", maxKbps: "4474" },
    },
    words:
      "At 900 kbps, files fall outside this Target's 1080p sizes, which take 1,398 kbps to 4,474 kbps. Change the bitrate or the sizes.",
  },
  {
    problem: { code: "outside_size", field: "Kbps", facts: { kbps: "900", minKbps: "1398" } },
    words:
      "At 900 kbps, files fall outside this Target's 1080p sizes, which take at least 1,398 kbps. Change the bitrate or the sizes.",
  },
  {
    problem: { code: "range_from_sdr", field: "Range", facts: { range: "sdr" } },
    words:
      "An SDR source always makes an SDR file, and this Target's HDR filter refuses SDR. Let the HDR filter allow SDR.",
  },
  {
    problem: { code: "container_audio", field: "Audio", facts: { container: "mp4" } },
    words: "MP4 can't hold TrueHD or DTS audio. Choose stereo AAC or 5.1 AC-3, or choose MKV.",
  },
] as const;

test.each(refusals)(
  "Conversion settings the server refuses with $problem.code on $problem.field read in words",
  async ({ problem, words }) => {
    const sent: unknown[] = [];
    const refusing = http.post("*/api/v1/targets", async ({ request }) => {
      sent.push(await request.json());
      return HttpResponse.json(
        {
          status: 422,
          code: "conversion_refused",
          title: "These Conversion settings can't make a file this Target accepts",
          problems: [problem],
        },
        { status: 422, headers: { "Content-Type": "application/problem+json" } },
      );
    });
    renderApp({ path: "/settings/targets/new", handlers: [...targetHandlers(), refusing] });
    await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Phone");
    await userEvent.click(
      page.getByRole("switch", { name: "Make this Target's files from another Target's" }),
    );
    await userEvent.fill(page.getByRole("textbox", { name: "Video bitrate in kbps" }), "900");
    await userEvent.click(page.getByRole("button", { name: "Create Target" }));
    await expect.element(page.getByRole("alert")).toHaveTextContent(words);
    expect(sent).toEqual([
      expect.objectContaining({
        conversion: {
          codec: "h264",
          maxHeight: 1080,
          kbps: 900,
          depth: 8,
          range: "sdr",
          keepDolbyVision: false,
          curve: "bt2390",
          audio: "copy",
          dropCommentary: true,
          container: "mkv",
          runOnNodeId: null,
        },
      }),
    ]);
  },
);

const phone: TargetsTarget = {
  ...keeper,
  conversion: {
    codec: "hevc",
    maxHeight: 720,
    kbps: 2000,
    depth: 10,
    range: "sdr",
    keepDolbyVision: false,
    curve: "bt2390",
    audio: "stereo_aac",
    dropCommentary: true,
    container: "mp4",
    runOnNodeId: null,
    version: 2,
    older: 3,
    causes: [],
    defaultKbps: 2000,
  },
};
const row = (workId: string, title: string, files: number) => ({
  workId,
  title,
  files,
  seconds: 3600,
  bytes: 1_000_000_000,
  addedAt: "2026-11-01T00:00:00Z",
});
const remake = {
  older: 3,
  rows: [
    row("0199e1f2-7a3b-7c4d-8e5f-0000000000b1", "Heat", 1),
    row("0199e1f2-7a3b-7c4d-8e5f-0000000000b2", "Ronin", 2),
  ],
};
test("a save while files made with older settings remain remakes only the picked Works", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/targets/${phone.id}`,
    (query) => (query.get("preview") === "true" ? { ...noImpact, remake } : noImpact),
    { target: phone },
  );
  renderApp({
    path: `/settings/targets/${phone.id}`,
    handlers: [...targetHandlers([phone]), op.handler],
  });
  await expect.element(page.getByText("3 files were made with older settings.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  const dialog = page.getByRole("dialog");
  await userEvent.click(dialog.getByRole("checkbox", { name: "Ronin" }));
  await userEvent.click(dialog.getByRole("button", { name: "Save and remake 2 files" }));
  await expect
    .poll(() => op.sent.at(-1)?.query)
    .toEqual({
      unwanted: "delete",
      confirm: "0",
      confirmRemake: "2",
    });
  expect(op.sent.at(-1)?.body).toEqual(
    expect.objectContaining({ remake: ["0199e1f2-7a3b-7c4d-8e5f-0000000000b2"] }),
  );
});
