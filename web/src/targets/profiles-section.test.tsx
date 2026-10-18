import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsProfile, WorksContentType } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { changeOp } from "../test/impact";
import { cinema } from "../test/library";
import { films, profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { targetHandlers } from "../test/targets";

const zines: WorksContentType = {
  ...cinema,
  id: "zine",
  name: "Zines",
  work: { one: "Zine", many: "Zines" },
  item: { one: "Issue", many: "Issues" },
};

const zineProfile: TargetsProfile = {
  ...films,
  id: "0199e1f2-7a3b-7c4d-8e5f-000000000003",
  name: "Zines",
  contentType: "zine",
  targets: [],
  works: 0,
};

const handlers = (more: Parameters<typeof renderApp>[0] = {}) => [
  ...(more.handlers ?? []),
  ...targetHandlers(),
  ...profileHandlers([films, ultra, zineProfile]),
  getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
];

const textsOf = (table: string) =>
  page
    .getByRole("table", { name: table })
    .getByRole("row")
    .all()
    .slice(1)
    .map((row) => row.element().textContent);

test("Profiles lists each Content Type's Profiles with the default marked, their Targets and Works", async () => {
  renderApp({ path: "/settings/profiles", handlers: handlers() });
  await expect.element(page.getByRole("heading", { name: "Films 2", level: 2 })).toBeVisible();
  expect(textsOf("Films Profiles")).toEqual([
    "FilmsDefault1080p12 Works",
    "Ultra2160p HDR, 1080p1 Work",
  ]);
  await expect.element(page.getByRole("heading", { name: "Zines 1", level: 2 })).toBeVisible();
  expect(textsOf("Zines Profiles")).toEqual(["ZinesDefaultNone0 Works"]);
  await expect
    .element(page.getByRole("button", { name: "Actions for Films" }))
    .not.toBeInTheDocument();

  await userEvent.click(page.getByRole("link", { name: "Ultra" }));
  await expect.element(page.getByRole("heading", { name: "Ultra", level: 1 })).toBeVisible();
});

test("Make default makes another Profile the type's default and says what new Works take", async () => {
  const made: string[] = [];
  renderApp({
    path: "/settings/profiles",
    handlers: handlers({
      handlers: [
        http.post("*/api/v1/profiles/:profileId/default", ({ params }) => {
          made.push(String(params["profileId"]));
          return HttpResponse.json({ ...ultra, default: true });
        }),
      ],
    }),
  });
  await userEvent.click(page.getByRole("button", { name: "Actions for Ultra" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Make default" }));
  await expect.element(page.getByText("New Films now take Ultra.")).toBeVisible();
  expect(made).toEqual([ultra.id]);
});

test("New Profile opens an empty editor on the first enabled Content Type", async () => {
  const { router } = renderApp({ path: "/settings/profiles", handlers: handlers() });
  await userEvent.click(page.getByRole("button", { name: "New Profile" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/profiles/new");
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("");
  await expect
    .element(page.getByRole("combobox", { name: "Content Type" }))
    .toHaveTextContent("Films");
  await expect
    .element(page.getByText("This Profile wants nothing.", { exact: false }))
    .toBeVisible();
});

async function pickReplacement(name: string) {
  await userEvent.click(page.getByRole("button", { name: "Actions for Ultra" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  const picking = page.getByRole("dialog", { name: "Delete Ultra?" });
  await expect
    .element(picking.getByText("Its 1 Work moves to the Profile you pick."))
    .toBeVisible();
  const picker = picking.getByRole("combobox", { name: "Move its Works to" });
  await expect.element(picker).toHaveTextContent("Films");
  await userEvent.click(picker);
  expect(
    page
      .getByRole("option")
      .all()
      .map((option) => option.element().textContent),
  ).toEqual(["Films", "Other"]);
  await userEvent.click(page.getByRole("option", { name }));
  await userEvent.click(picking.getByRole("button", { name: "Delete Ultra" }));
}

test("Delete moves a Profile's Works to the one picked, through the Impact", async () => {
  const other: TargetsProfile = {
    ...ultra,
    id: "0199e1f2-7a3b-7c4d-8e5f-000000000004",
    name: "Other",
  };
  const op = changeOp(
    "delete",
    `*/api/v1/profiles/${ultra.id}`,
    () => ({
      works: 1,
      ownList: 0,
      unwanted: { files: 2, bytes: 2 * 1024 ** 3 },
      renamed: 0,
      newlyWanted: 1,
      downloading: 0,
    }),
    { profile: other },
  );
  renderApp({
    path: "/settings/profiles",
    handlers: [
      op.handler,
      ...targetHandlers(),
      ...profileHandlers([films, ultra, other, zineProfile]),
      getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
    ],
  });
  await pickReplacement("Other");

  const impact = page.getByRole("dialog", { name: "Delete Ultra", exact: true });
  await expect
    .element(impact.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("1 Work changes.2 files would fill nothing (2.0 GB).1 cell becomes wanted.");
  await userEvent.click(impact.getByRole("button", { name: "Delete Ultra and delete 2 files" }));

  await expect.element(page.getByText("Deleted Ultra. Its Works now take Other.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { moveTo: other.id, preview: "true" },
    { moveTo: other.id, unwanted: "delete", confirm: "2" },
  ]);
});
