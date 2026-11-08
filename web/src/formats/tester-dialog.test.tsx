import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FormatsParsed } from "../api/gen/model";
import { renderKit } from "../test/render-kit";
import { worker } from "../test/worker";
import { useTester } from "./tester-dialog";

const name = "Heat.1995.1080p.BluRay.x264-AMIABLE";

const read: FormatsParsed = {
  name,
  sentence: "Reads as Bluray-1080p from AMIABLE. It matches Heat (1995) and would fill 1080p.",
  words: "Bluray-1080p · AMIABLE",
  facts: {
    quality: "bluray-1080p",
    source: "bluray",
    resolution: "1080p",
    modifier: null,
    revision: { version: 1, real: 0, repack: false },
    group: "AMIABLE",
    cut: "standard",
    edition: null,
    hdr: [],
    video: "x264",
    audio: [],
    channels: [],
    languages: [],
    ids: {},
    titles: ["Heat"],
    year: 1995,
    marks: [],
    span: "",
  },
  works: [{ id: "work-heat", label: "Heat (1995)" }],
  covers: [],
  formats: [],
  targets: [],
};

function Release() {
  const tester = useTester();
  return (
    <>
      <button
        type="button"
        onClick={() => {
          tester.open({ names: [name], work: { id: "work-heat", label: "Heat (1995)" } });
        }}
      >
        Open in tester
      </button>
      {tester.dialog}
    </>
  );
}

test("Open in tester tests a Release's name against its Work at once, and can match it by title instead", async () => {
  const sent: unknown[] = [];
  worker.use(
    http.post("*/api/v1/releases/parse", async ({ request }) => {
      sent.push(await request.json());
      return HttpResponse.json({ results: [read] });
    }),
  );
  renderKit(<Release />);
  await userEvent.click(page.getByRole("button", { name: "Open in tester" }));

  const dialog = page.getByRole("dialog", { name: "Tester" });
  await expect.element(dialog.getByText(read.sentence)).toBeVisible();
  await expect.element(dialog.getByRole("textbox", { name: "Release names" })).toHaveValue(name);
  await expect.element(dialog.getByText("Each name is judged against Heat (1995).")).toBeVisible();
  expect(sent).toEqual([{ names: [name], kind: "release", workId: "work-heat" }]);

  await userEvent.click(dialog.getByRole("button", { name: "Match by title instead" }));
  await userEvent.click(dialog.getByRole("button", { name: "Test", exact: true }));
  await expect.poll(() => sent).toHaveLength(2);
  expect(sent[1]).toEqual({ names: [name], kind: "release" });
});
