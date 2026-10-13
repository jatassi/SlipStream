import { expect, test } from "vitest";
import { page } from "vitest/browser";

import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";

test("the Library lists each Work with its year and links to it", async () => {
  renderApp({
    path: "/",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler({
        works: [
          {
            id: orchardId,
            contentType: "cinema",
            key: "9000001",
            label: "The Quiet Orchard",
            sortLabel: "Quiet Orchard",
            addedAt: "2026-10-12T09:00:00Z",
            refreshedAt: "2026-10-12T09:00:00Z",
            fields: orchard.fields,
          },
        ],
      }),
    ],
  });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  const work = page.getByRole("main").getByRole("listitem");
  await expect.element(work).toHaveTextContent("The Quiet Orchard 2021");
  await expect
    .element(work.getByRole("link", { name: "The Quiet Orchard" }))
    .toHaveAttribute("href", `/works/${orchardId}`);
  await expect
    .element(page.getByRole("link", { name: "Add film" }))
    .toHaveAttribute("href", "/add");
});

test("an empty Library with every Content Type off offers nothing to add", async () => {
  renderApp({
    path: "/",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] }),
      getListWorksMockHandler({ works: [] }),
    ],
  });
  await expect.element(page.getByText("Nothing in the library yet.")).toBeVisible();
  await expect.element(page.getByRole("link", { name: /^Add /u })).not.toBeInTheDocument();
});
