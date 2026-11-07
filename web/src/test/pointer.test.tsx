import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderApp } from "./render-app";

const library = () => page.getByRole("navigation").getByRole("link", { name: "Library" }).first();

test("one test leaves the pointer on a link", async () => {
  renderApp({ path: "/nodes" });
  await userEvent.hover(library());
  await expect.poll(() => library().element().matches(":hover")).toBe(true);
});

test("the next test draws that link without hovering it", async () => {
  const { router } = renderApp({ path: "/nodes" });
  await expect.element(library()).toBeVisible();
  await expect.poll(() => router.state.status).toBe("idle");
  await new Promise((resolve) => {
    requestAnimationFrame(resolve);
  });
  expect(library().element().matches(":hover")).toBe(false);
});
