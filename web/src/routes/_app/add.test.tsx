import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksNewWork } from "../../api/gen/model";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
  getSearchCatalogMockHandler,
} from "../../api/gen/query/works/works.msw";
import { AddWorkBody } from "../../api/gen/zod/works/works.zod";
import { problem } from "../../test/handlers";
import { cinema, orchard, orchardId } from "../../test/library";
import { films, profileHandlers, ultra } from "../../test/profiles";
import { renderApp } from "../../test/render-app";

const harvestId = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

function catalogServer() {
  const searched: string[] = [];
  const added: WorksNewWork[] = [];
  const handlers = [
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    getSearchCatalogMockHandler(({ request }) => {
      const url = new URL(request.url);
      searched.push(`${url.pathname}?${url.search.slice(1)}`);
      return {
        candidates: [
          {
            key: "9000001",
            label: "The Quiet Orchard",
            original: "",
            overview: "",
            fields: [{ key: "year", kind: "number", number: 2021 }],
            workId: null,
          },
          {
            key: "9000002",
            label: "Orchard Harvest",
            original: "",
            overview: "",
            fields: [{ key: "year", kind: "number", number: 2019 }],
            workId: harvestId,
          },
        ],
      };
    }),
    http.post("*/api/v1/works", async ({ request }) => {
      added.push(AddWorkBody.parse(await request.json()));
      return HttpResponse.json({ created: true, work: orchard });
    }),
    getGetWorkMockHandler(orchard),
    ...profileHandlers([
      films,
      ultra,
      {
        ...ultra,
        id: "0199e1f2-7a3b-7c4d-8e5f-000000000009",
        name: "Zines",
        contentType: "zine",
      },
    ]),
  ];
  return { handlers, searched, added };
}

test("searching puts the term in the URL and lists each candidate with its year", async () => {
  const server = catalogServer();
  const { router } = renderApp({ path: "/add?contentType=cinema", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();

  await userEvent.fill(page.getByRole("searchbox", { name: "Title" }), "  quiet orchard ");
  await userEvent.click(page.getByRole("button", { name: "Search" }));

  const results = page.getByRole("list", { name: "Results" }).getByRole("listitem");
  await expect.element(results.first()).toHaveTextContent("The Quiet Orchard2021Add");
  await expect.element(results.last()).toHaveTextContent("Orchard Harvest2019In the library");
  expect(router.state.location.href).toBe("/add?contentType=cinema&query=quiet+orchard");
  expect(server.searched).toEqual(["/api/v1/content-types/cinema/catalog?query=quiet+orchard"]);
  await expect
    .element(page.getByRole("link", { name: "Orchard Harvest 2019 In the library" }))
    .toHaveAttribute("href", `/works/${harvestId}`);
});

const notifications = () => page.getByRole("region", { name: "Notifications" });

async function openSheet() {
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  const sheet = page.getByRole("dialog", { name: "Add The Quiet Orchard" });
  await expect.element(sheet).toBeVisible();
  return sheet;
}

test("Add opens a sheet on the type's default Profile, adds the Work monitored and opens its page", async () => {
  const server = catalogServer();
  const { router } = renderApp({
    path: "/add?contentType=cinema&query=quiet",
    handlers: server.handlers,
  });

  const sheet = await openSheet();
  await expect
    .element(sheet.getByRole("combobox", { name: "Profile" }))
    .toHaveTextContent("Films (default)");
  await expect.element(sheet.getByRole("switch", { name: "Monitor" })).toBeChecked();
  await userEvent.click(sheet.getByRole("button", { name: "Add Film" }));

  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect.element(notifications().getByText("Added The Quiet Orchard")).toBeVisible();
  expect(server.added).toEqual([
    { contentType: "cinema", key: "9000001", monitor: true, profileId: films.id },
  ]);
  expect(router.state.location.href).toBe(`/works/${orchardId}`);
});

test("the add sheet offers only the type's Profiles, and adds unmonitored on another", async () => {
  const server = catalogServer();
  renderApp({ path: "/add?contentType=cinema&query=quiet", handlers: server.handlers });

  const sheet = await openSheet();
  await userEvent.click(sheet.getByRole("combobox", { name: "Profile" }));
  await expect.element(page.getByRole("option", { name: "Ultra" })).toBeVisible();
  expect(
    page
      .getByRole("option")
      .all()
      .map((option) => option.element().textContent),
  ).toEqual(["Films (default)", "Ultra"]);
  await userEvent.click(page.getByRole("option", { name: "Ultra" }));
  await userEvent.click(sheet.getByRole("switch", { name: "Monitor" }));
  await userEvent.click(sheet.getByRole("button", { name: "Add Film" }));

  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  expect(server.added).toEqual([
    { contentType: "cinema", key: "9000001", monitor: false, profileId: ultra.id },
  ]);
});

test("a failed Add says the server's reason in a toast", async () => {
  const server = catalogServer();
  renderApp({
    path: "/add?contentType=cinema&query=quiet",
    handlers: [
      http.post("*/api/v1/works", () =>
        problem(502, "metadata_unavailable", "The metadata source didn't answer. Try again later."),
      ),
      ...server.handlers,
    ],
  });
  const sheet = await openSheet();
  await userEvent.click(sheet.getByRole("button", { name: "Add Film" }));
  await expect
    .element(notifications().getByRole("dialog"))
    .toHaveTextContent("FailedThe metadata source didn't answer. Try again later.");
  await expect.element(sheet.getByRole("button", { name: "Add Film" })).toBeEnabled();
});

test.each([
  ["a disabled", "/add?contentType=cinema"],
  ["an unknown", "/add?contentType=nothing"],
])("%s Content Type has no Add page", async (_kind, path) => {
  renderApp({
    path,
    handlers: [getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] })],
  });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
  await expect.element(page.getByRole("heading", { name: /^Add/u })).not.toBeInTheDocument();
});
