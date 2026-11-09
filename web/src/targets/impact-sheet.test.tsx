import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { z } from "zod";

import type { TargetsImpact, TargetsTarget } from "../api/gen/model";
import { changeOp, noImpact } from "../test/impact";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

const oneUnwanted: TargetsImpact = {
  works: 2,
  ownList: 0,
  unwanted: { files: 1, bytes: 734_003_200 },
  renamed: 3,
  renames: [],
  newlyWanted: 4,
  downloading: 1,
};

const saving = (answer: Parameters<typeof changeOp>[2]) =>
  changeOp("put", `*/api/v1/targets/${starter.id}`, answer, { target: starter });

const sheet = () => page.getByRole("dialog", { name: "Save 1080p" });
const lines = () =>
  sheet()
    .getByRole("list", { name: "What changes" })
    .getByRole("listitem")
    .all()
    .map((line) => line.element().textContent);

async function editAndSave() {
  await userEvent.click(page.getByRole("checkbox", { name: "WEBDL-1080p" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
}

test("saving a Target shows its Impact in plain counts, with Delete chosen and the count on the button", async () => {
  const op = saving(() => oneUnwanted);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();

  await expect.element(sheet()).toBeVisible();
  expect(lines()).toEqual([
    "2 Works change.",
    "1 file would fill nothing (700 MB).",
    "3 files are renamed.",
    "4 cells become wanted.",
    "1 download in progress is for a cell that is no longer wanted. Its file is deleted or kept, as chosen below.",
  ]);
  await expect.element(sheet().getByRole("radio", { name: "Delete" })).toBeChecked();
  await expect
    .element(sheet().getByRole("button", { name: "Save and delete 1 file" }))
    .toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([{ preview: "true" }]);

  await userEvent.click(sheet().getByRole("radio", { name: "Keep as Kept files" }));
  await userEvent.click(sheet().getByRole("button", { name: "Save and keep 1 file" }));

  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
  await expect.element(page.getByRole("heading", { name: "Settings", level: 1 })).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "keep", confirm: "1" },
  ]);
  expect(op.sent[1]?.body).toEqual(op.sent[0]?.body);
  expect(op.sent[1]?.body).toMatchObject({ name: "1080p", cutoff: "bluray-1080p" });
});

test("the Impact lists the files it renames under Details, Kept files included, and counts the rest", async () => {
  const op = saving(() => ({
    ...noImpact,
    works: 2,
    renamed: 3,
    renames: [
      {
        workId: "0199e1f2-7a3b-7c4d-8e5f-0000000000a1",
        title: "Heat",
        files: [
          {
            fileId: "0199e1f2-7a3b-7c4d-8e5f-0000000000b1",
            from: "Heat (1995)/Heat (1995) - 1080p.mkv",
            to: "Heat (1995)/Heat (1995) - 1080p web.mkv",
          },
          {
            fileId: "0199e1f2-7a3b-7c4d-8e5f-0000000000b2",
            from: "Heat (1995)/Heat (1995) - 1080p (extra 1).mkv",
            to: "Heat (1995)/Heat (1995) - 1080p web (extra 1).mkv",
          },
        ],
      },
    ],
  }));
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();

  await expect.element(sheet()).toBeVisible();
  expect(lines()).toEqual(["2 Works change.", "3 files are renamed."]);
  await userEvent.click(sheet().getByRole("button", { name: "Details" }));
  const heat = sheet().getByRole("list", { name: "Heat" });
  await expect.element(heat).toBeVisible();
  expect(
    heat
      .getByRole("listitem")
      .elements()
      .map((row) => row.textContent),
  ).toEqual([
    "Heat (1995)/Heat (1995) - 1080p.mkvHeat (1995)/Heat (1995) - 1080p web.mkv",
    "Heat (1995)/Heat (1995) - 1080p (extra 1).mkvHeat (1995)/Heat (1995) - 1080p web (extra 1).mkv",
  ]);
  await expect.element(sheet().getByText("1 more file is renamed.")).toBeVisible();
});

test("an Impact that shows nothing saves without the sheet", async () => {
  const op = saving(() => noImpact);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "1080p web");
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved 1080p web.")).toBeVisible();
  expect(sheet().query()).toBeNull();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
});

test("a save refused as impact_changed previews again and shows the fresh counts, saying they changed", async () => {
  const fresh: TargetsImpact = { ...oneUnwanted, unwanted: { files: 3, bytes: 3 * 734_003_200 } };
  const op = saving((query, calls) => {
    if (query.get("preview") === "true") {
      return calls === 0 ? oneUnwanted : fresh;
    }
    return calls === 1 ? "changed" : fresh;
  });
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();
  await userEvent.click(sheet().getByRole("button", { name: "Save and delete 1 file" }));

  await expect
    .element(
      sheet().getByText("These counts changed since the last preview. Check them before you save."),
    )
    .toBeVisible();
  expect(lines()).toContain("3 files would fill nothing (2.1 GB).");
  await userEvent.click(sheet().getByRole("button", { name: "Save and delete 3 files" }));

  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "1" },
    { preview: "true" },
    { unwanted: "delete", confirm: "3" },
  ]);
});

test("Cancel on the Impact saves nothing and keeps the edit", async () => {
  const op = saving(() => oneUnwanted);
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [op.handler, ...targetHandlers()],
  });
  await editAndSave();
  await userEvent.click(sheet().getByRole("button", { name: "Cancel" }));

  await expect.element(sheet()).not.toBeInTheDocument();
  await expect.element(page.getByRole("checkbox", { name: "WEBDL-1080p" })).not.toBeChecked();
  expect(op.sent.map((call) => call.query)).toEqual([{ preview: "true" }]);
});

async function editScores() {
  const scores = page.getByRole("region", { name: "Scores" });
  await expect.element(scores.getByText(/^Scores 1 of 3 Custom Formats\./u)).toBeVisible();
  await userEvent.click(scores.getByRole("button", { name: "Details" }));
  await userEvent.fill(scores.getByRole("searchbox", { name: "Filter Custom Formats" }), "tier");
  await expect
    .poll(() =>
      scores
        .getByRole("list", { name: "Custom Format scores" })
        .getByRole("listitem")
        .elements()
        .map((row) => row.textContent),
    )
    .toEqual(["HD Bluray Tier 01Score"]);
  await userEvent.fill(scores.getByRole("textbox", { name: "HD Bluray Tier 01 score" }), "60");
  await userEvent.fill(scores.getByRole("textbox", { name: "Increment" }), "0");
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await expect.element(page.getByText("Fix the scores marked below.")).toBeVisible();
  await userEvent.fill(scores.getByRole("textbox", { name: "Increment" }), "10");
}

test("saving a Target sends the scores and propers setting as edited", async () => {
  const scored: TargetsTarget = {
    ...keeper,
    scores: {
      formats: [{ formatId: "0199e1f2-7a3b-7c4d-8e5f-0000000000f1", score: -100 }],
      minimum: 10,
      upgradeUntil: 50,
      increment: 5,
    },
    propers: "dont_prefer",
  };
  const op = changeOp("put", `*/api/v1/targets/${scored.id}`, () => noImpact, { target: scored });
  renderApp({
    path: `/settings/targets/${scored.id}`,
    handlers: [op.handler, ...targetHandlers([scored])],
  });
  await editScores();
  await userEvent.click(page.getByRole("radio", { name: "Prefer, don't upgrade" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved 2160p HDR.")).toBeVisible();
  const sent = {
    scores: {
      formats: [
        { formatId: "0199e1f2-7a3b-7c4d-8e5f-0000000000f1", score: -100 },
        { formatId: "0199e1f2-7a3b-7c4d-8e5f-0000000000f3", score: 60 },
      ],
      minimum: 10,
      upgradeUntil: 50,
      increment: 10,
    },
    propers: "prefer_no_upgrade",
  };
  expect(op.sent.map((call) => call.body)).toEqual([
    expect.objectContaining(sent),
    expect.objectContaining(sent),
  ]);
});

test("a Target reopened right after its save opens as saved", async () => {
  let saved = starter;
  const renamed = z.object({ name: z.string() });
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [
      http.put(`*/api/v1/targets/${starter.id}`, async ({ request }) => {
        if (new URL(request.url).searchParams.get("preview") !== "true") {
          saved = { ...starter, name: renamed.parse(await request.json()).name };
        }
        return HttpResponse.json({ impact: noImpact, target: saved });
      }),
      http.get(`*/api/v1/targets/${starter.id}`, () => HttpResponse.json(saved)),
      ...targetHandlers(),
    ],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "1080p web");
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await expect.element(page.getByText("Saved 1080p web.")).toBeVisible();

  await userEvent.click(
    page.getByRole("table", { name: "Targets" }).getByRole("link", { name: "1080p", exact: true }),
  );
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("1080p web");
});
