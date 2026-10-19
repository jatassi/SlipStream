import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListGrabsMockHandler } from "../../../api/gen/query/grabs/grabs.msw";
import { downloading } from "../../../test/grabs";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const handlers = [getListGrabsMockHandler({ grabs: [downloading] })];
const sentence = () => page.getByRole("paragraph").filter({ hasText: "moving" });

test("on a desktop Jobs and Torrents are page tabs under the Activity title", async () => {
  const { router } = renderApp({ path: "/activity", handlers });
  const views = page.getByRole("main").getByRole("navigation", { name: "Activity" });
  await expect
    .element(views.getByRole("link", { name: "Jobs" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(sentence()).toBeVisible();
  expect(page.getByRole("radiogroup").all()).toHaveLength(0);

  await userEvent.click(views.getByRole("link", { name: "Torrents" }));
  await expect.poll(() => router.state.location.pathname).toBe("/activity/torrents");
  await expect
    .element(views.getByRole("link", { name: "Torrents" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(views.getByRole("link", { name: "Jobs" }))
    .not.toHaveAttribute("aria-current");
  await expect.element(page.getByRole("heading", { name: "Activity", level: 1 })).toBeVisible();
  await expect
    .element(
      page.getByRole("navigation", { name: "Main" }).getByRole("link", { name: /^Activity/u }),
    )
    .toHaveAttribute("aria-current", "page");
  await expect.element(sentence()).not.toBeInTheDocument();
});

test("on a phone Jobs and Torrents are a segmented control under the large title", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/activity", handlers });
  const views = page.getByRole("radiogroup", { name: "View" });
  await expect.element(views.getByRole("radio", { name: "Jobs" })).toBeChecked();
  await expect.element(sentence()).toBeVisible();
  await expect
    .element(page.getByRole("main").getByRole("navigation", { name: "Activity" }))
    .not.toBeInTheDocument();

  await userEvent.click(views.getByRole("radio", { name: "Torrents" }));
  await expect.poll(() => router.state.location.pathname).toBe("/activity/torrents");
  await expect.element(views.getByRole("radio", { name: "Torrents" })).toBeChecked();
  await expect.element(page.getByRole("heading", { name: "Activity", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("banner").getByRole("link")).not.toBeInTheDocument();
  await expect.element(sentence()).not.toBeInTheDocument();

  await userEvent.click(views.getByRole("radio", { name: "Jobs" }));
  await expect.poll(() => router.state.location.pathname).toBe("/activity");
  await expect.element(sentence()).toBeVisible();
});
