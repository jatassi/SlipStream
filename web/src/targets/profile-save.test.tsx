import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { UpdateProfileBody } from "../api/gen/zod/targets/targets.zod";
import { changeOp, noImpact } from "../test/impact";
import { cinema } from "../test/library";
import { profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { starter, targetHandlers } from "../test/targets";

const handlers = () => [
  ...targetHandlers(),
  ...profileHandlers(),
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
];

test("saving a Profile's Targets shows the Works that keep their own Edition list, and sends the edit", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/profiles/${ultra.id}`,
    () => ({ ...noImpact, works: 3, ownList: 1, renamed: 2 }),
    { profile: ultra },
  );
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [op.handler, ...handlers()],
  });
  await userEvent.click(page.getByRole("button", { name: "Remove 2160p HDR" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  const sheet = page.getByRole("dialog", { name: "Save Ultra" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toMatchTextContent(/3 Works change.+1 Work keeps its own Edition list.+2 files are renamed/u);
  expect(sheet.getByRole("radiogroup").query()).toBeNull();
  await userEvent.click(sheet.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
  expect(op.sent[1]?.body).toEqual({
    name: "Ultra",
    contentType: "cinema",
    targets: [{ targetId: starter.id, sourcing: "download" }],
    editions: { kind: "any" },
    languages: [],
  });
});

test("saving a Profile's chosen cuts previews them through the Impact, then sends them", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/profiles/${ultra.id}`,
    () => ({ ...noImpact, works: 1, newlyWanted: 2 }),
    { profile: ultra },
  );
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [op.handler, ...handlers()],
  });
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Superfan" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  const sheet = page.getByRole("dialog", { name: "Save Ultra" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toMatchTextContent(/1 Work changes.+2 more files/u);
  await userEvent.click(sheet.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  const editions = { kind: "cuts", cuts: ["standard", "superfan"] };
  expect(op.sent.map((call) => [call.query, UpdateProfileBody.parse(call.body).editions])).toEqual([
    [{ preview: "true" }, editions],
    [{ unwanted: "delete", confirm: "0" }, editions],
  ]);
});

const english = {
  language: "en",
  forced: false,
  hearingImpaired: "allow",
  audio: "always",
} as const;

test("saving a Profile's Targets keeps the subtitle languages it lists", async () => {
  const subtitled = { ...ultra, languages: [english] };
  const op = changeOp("put", `*/api/v1/profiles/${ultra.id}`, () => ({ ...noImpact, works: 3 }), {
    profile: subtitled,
  });
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [
      op.handler,
      ...targetHandlers(),
      ...profileHandlers([subtitled]),
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Remove 2160p HDR" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await userEvent.click(
    page
      .getByRole("dialog", { name: "Save Ultra" })
      .getByRole("button", { name: "Save", exact: true }),
  );

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  expect(op.sent.map((call) => UpdateProfileBody.parse(call.body).languages)).toEqual([
    [english],
    [english],
  ]);
});

const french = {
  language: "fr",
  forced: false,
  hearingImpaired: "allow",
  audio: "always",
} as const;

const languageImpact = {
  ...noImpact,
  subtitles: {
    gained: [{ language: "en", forced: false, searches: 1204, needScan: 312 }],
    dropped: [{ language: "fr", forced: false, deleted: 88 }],
    days: 12,
  },
};

test("a subtitle language change shows its searches, scans, days and deletes, and the save confirms the deletes", async () => {
  const subtitled = { ...ultra, languages: [french] };
  const op = changeOp("put", `*/api/v1/profiles/${ultra.id}`, () => languageImpact, {
    profile: subtitled,
  });
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [
      op.handler,
      ...targetHandlers(),
      ...profileHandlers([subtitled]),
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
    ],
  });
  await userEvent.click(
    page.getByRole("button", { name: "Remove French, full, hearing impaired allowed" }),
  );
  await userEvent.click(page.getByRole("button", { name: "Add language" }));
  await userEvent.click(page.getByRole("menuitem", { name: "English", exact: true }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));

  const sheet = page.getByRole("dialog", { name: "Save Ultra" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toHaveTextContent(
      [
        "Saving searches English subtitles for 1,204 files. 312 of them need their audio scanned first.",
        "At today's downloads, the last of these arrives in about 12 days.",
        "Saving deletes 88 French subtitles SlipStream downloaded. Uploads and subtitles found beside the files stay.",
      ].join(""),
    );
  await userEvent.click(sheet.getByRole("button", { name: "Save and delete 88 subtitles" }));

  await expect.element(page.getByText("Saved Ultra.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0", confirmSubtitles: "88" },
  ]);
  expect(op.sent.map((call) => UpdateProfileBody.parse(call.body).languages)).toEqual([
    [english],
    [english],
  ]);
});
