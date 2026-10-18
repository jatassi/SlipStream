import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsTarget } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

const table = () => page.getByRole("table", { name: "Targets" });

test("Targets lists each Target's quality span, cutoff and the Profiles using it", async () => {
  renderApp({ path: "/settings/targets", handlers: targetHandlers() });
  const tabs = page.getByRole("navigation", { name: "Settings" });
  await expect
    .element(tabs.getByRole("link", { name: "Targets" }))
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
