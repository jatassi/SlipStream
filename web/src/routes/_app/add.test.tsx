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
            fields: [{ key: "year", kind: "number", number: 2021 }],
            workId: null,
          },
          {
            key: "9000002",
            label: "Orchard Harvest",
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
  ];
  return { handlers, searched, added };
}

test("searching puts the term in the URL and lists each candidate with its year", async () => {
  const server = catalogServer();
  const { router } = renderApp({ path: "/add", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Add film" })).toBeVisible();

  await userEvent.fill(page.getByRole("searchbox", { name: "Title" }), "  quiet orchard ");
  await userEvent.click(page.getByRole("button", { name: "Search" }));

  const results = page.getByRole("list", { name: "Results" }).getByRole("listitem");
  await expect.element(results.first()).toHaveTextContent("The Quiet Orchard 2021Add");
  await expect.element(results.last()).toHaveTextContent("Orchard Harvest 2019Open");
  expect(router.state.location.href).toBe("/add?query=quiet+orchard");
  expect(server.searched).toEqual(["/api/v1/content-types/cinema/catalog?query=quiet+orchard"]);
  await expect
    .element(page.getByRole("link", { name: "Open Orchard Harvest" }))
    .toHaveAttribute("href", `/works/${harvestId}`);
});

test("Add adds the Work monitored and opens its page", async () => {
  const server = catalogServer();
  const { router } = renderApp({ path: "/add?query=quiet", handlers: server.handlers });

  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));

  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  expect(server.added).toEqual([{ contentType: "cinema", key: "9000001", monitor: true }]);
  expect(router.state.location.href).toBe(`/works/${orchardId}`);
});

test("a failed Add shows the server's reason", async () => {
  const server = catalogServer();
  renderApp({
    path: "/add?query=quiet",
    handlers: [
      http.post("*/api/v1/works", () =>
        problem(502, "metadata_unavailable", "The metadata source didn't answer. Try again later."),
      ),
      ...server.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("The metadata source didn't answer. Try again later.");
});
