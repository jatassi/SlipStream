import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../test/handlers";
import { candidate, manualSearch, openMissing as open } from "../test/subtitle-candidates";
import { attempting, started } from "../test/subtitles-work";
import { asPhone } from "../test/viewport";

const title = "English subtitle for The Quiet Orchard (2021) - 2160p.mkv";

test("two candidates of one release are told apart by their provider", async () => {
  const release = "The.Quiet.Orchard.2021.2160p.WEB-DL-GRP";
  const world = open({
    start: started,
    pick: attempting,
    candidates: () => ({
      state: "done",
      outcomes: [],
      candidates: [
        candidate("subdl:1", { release }),
        candidate("subsource:1", { provider: "subsource", release }),
      ],
    }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(dialog.getByRole("button", { name: `Pick ${release} from SubSource` }));
  expect(world.calls).toEqual(["candidates", "pick subsource:1"]);
});

test("a pick whose attempt can't be read says why", async () => {
  open({
    start: started,
    pick: attempting,
    attempt: () =>
      problem(
        410,
        "subtitle_attempt_expired",
        "The subtitle download or upload expired",
        "This download has expired. Search again.",
      ),
    candidates: () => ({ state: "done", outcomes: [], candidates: [candidate("best")] }),
  });
  await userEvent.click(manualSearch());
  const dialog = page.getByRole("dialog", { name: title });
  await userEvent.click(
    dialog.getByRole("button", { name: "Pick The.Quiet.Orchard.2021.2160p.best from SubDL" }),
  );
  await expect.element(dialog.getByText("This download has expired. Search again.")).toBeVisible();
});

test("on phone a long release tried before fits the sheet with its Pick in view", async () => {
  await asPhone();
  const release =
    "The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX.HDR.DV.HEVC.TrueHD.Atmos.7.1-GROUP";
  open({
    start: started,
    candidates: () => ({
      state: "done",
      outcomes: [],
      candidates: [
        candidate("failed", {
          release,
          reasons: ["made_by_person"],
          tried: { outcome: "failed_timing", match: 24, at: new Date().toISOString() },
        }),
      ],
    }),
  });
  await userEvent.click(
    page
      .getByRole("list", { name: "Subtitles" })
      .getByRole("button", { name: "More for English subtitles" }),
  );
  await userEvent.click(page.getByRole("button", { name: "Manual search" }));
  const sheet = page.getByRole("dialog", { name: title });
  await userEvent.click(sheet.getByRole("button", { name: "Tried before" }));
  const pick = sheet.getByRole("button", { name: `Pick ${release} from SubDL` });
  await expect.element(pick).toBeVisible();
  const edge = sheet.element().getBoundingClientRect().right;
  expect(pick.element().getBoundingClientRect().right).toBeLessThanOrEqual(edge);
  expect(sheet.getByText(release).element().getBoundingClientRect().right).toBeLessThanOrEqual(
    edge,
  );
});
