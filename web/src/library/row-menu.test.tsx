import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetWorkSearchingMockHandler } from "../api/gen/query/grabs/grabs.msw";
import { orchardRow } from "../test/library";
import { cinemaLibrary } from "../test/library-page";
import { renderApp } from "../test/render-app";
import { aSearchQueued, aWorkSearching, queuedBehind, queuedGroupOf } from "../test/searching";

function open(view: "grid" | "table") {
  const searched: string[] = [];
  const press = http.post("*/api/v1/works/:workId/search", async ({ params, request }) => {
    searched.push(`${String(params["workId"])} ${await request.text()}`);
    return HttpResponse.json(aSearchQueued([queuedGroupOf(1)]), { status: 202 });
  });
  // The press's group waits behind Search all, so the toast says Queued.
  const held = getGetWorkSearchingMockHandler(aWorkSearching({ groups: [queuedBehind(1)] }));
  renderApp({
    path: `/library?view=${view}`,
    handlers: [press, held, ...cinemaLibrary().handlers],
  });
  return searched;
}

test.each(["grid", "table"] as const)(
  "in the %s a Work's menu holds Search, which searches the whole Work, and Manual search",
  async (view) => {
    const searched = open(view);
    await userEvent.click(
      page.getByRole("button", { name: `More actions for ${orchardRow.label}` }),
    );
    await expect.element(page.getByRole("menuitem", { name: "Manual search" })).toBeVisible();
    await userEvent.click(page.getByRole("menuitem", { name: "Search", exact: true }));
    await expect.element(page.getByText(`Queued ${orchardRow.label} for search.`)).toBeVisible();
    expect(searched).toEqual([`${orchardRow.id} `]);
  },
);
