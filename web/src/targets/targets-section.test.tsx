import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsTarget } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

const table = () => page.getByRole("table", { name: "Targets" });

test("Targets lists each Target's quality span, cutoff and the Profiles using it", async () => {
  renderApp({ path: "/settings/targets", handlers: targetHandlers() });
  const nav = page.getByRole("navigation", { name: "Settings" });
  await expect
    .element(nav.getByRole("link", { name: "Targets" }))
    .toHaveAttribute("aria-current", "page");
  const rows = table().getByRole("row");
  await expect
    .element(rows.nth(1))
    .toHaveTextContent("1080pHDTV-720p to Remux-1080pBluray-1080pMovies");
  await expect
    .element(rows.nth(2))
    .toHaveTextContent("2160p HDRWEBRip-2160p to Remux-2160pNo upgradesNone");
  await userEvent.click(table().getByRole("link", { name: "1080p" }));
  await expect.element(page.getByRole("heading", { name: "1080p", level: 1 })).toBeVisible();
});

test("New Target opens an empty editor", async () => {
  const { router } = renderApp({ path: "/settings/targets", handlers: targetHandlers() });
  await userEvent.click(page.getByRole("button", { name: "New Target" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/targets/new");
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("");
});

test("Duplicate copies a Target and says what it made", async () => {
  const copy: TargetsTarget = {
    ...keeper,
    id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e4c",
    name: "2160p HDR copy",
  };
  renderApp({
    path: "/settings/targets",
    handlers: [
      ...targetHandlers(),
      http.post(`*/api/v1/targets/${keeper.id}/duplicate`, () =>
        HttpResponse.json(copy, { status: 201 }),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Actions for 2160p HDR" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Duplicate" }));
  await expect.element(page.getByText("Made 2160p HDR copy from 2160p HDR.")).toBeVisible();
});

test("deleting a Target a Profile holds is refused with the Profiles named", async () => {
  const deleted: string[] = [];
  renderApp({
    path: "/settings/targets",
    handlers: [
      ...targetHandlers(),
      http.delete("*/api/v1/targets/:targetId", ({ params }) => {
        if (params["targetId"] === starter.id) {
          return HttpResponse.json(
            {
              status: 409,
              code: "target_in_use",
              title: "Profiles hold the Target",
              detail: "Movies uses it",
            },
            { status: 409, headers: { "Content-Type": "application/problem+json" } },
          );
        }
        deleted.push(String(params["targetId"]));
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Actions for 1080p" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  const dialog = page.getByRole("dialog", { name: "Delete 1080p?" });
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect
    .element(dialog.getByRole("alert"))
    .toHaveTextContent("1080p can't be deleted. Movies uses it.");
  await userEvent.click(dialog.getByRole("button", { name: "Cancel" }));

  await userEvent.click(page.getByRole("button", { name: "Actions for 2160p HDR" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "Delete 2160p HDR?" }).getByRole("button", { name: "Delete" }),
  );
  await expect.element(page.getByText("Deleted 2160p HDR.")).toBeVisible();
  expect(deleted).toEqual([keeper.id]);
});

test("a saved Target renamed to another Target's name in other case shows the refusal under Save", async () => {
  const refusal = "Another Target is named 4K. Use a name that differs by more than case.";
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [
      ...targetHandlers(),
      http.put("*/api/v1/targets/:targetId", () =>
        HttpResponse.json(
          {
            status: 409,
            code: "target_name_taken",
            title: "A Target already has that name",
            detail: refusal,
          },
          { status: 409, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
    ],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "4k");
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await expect.element(page.getByRole("alert")).toHaveTextContent(refusal);
});
