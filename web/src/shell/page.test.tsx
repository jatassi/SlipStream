import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router";
import type { ReactNode } from "react";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "../ui/shell-context";
import { type Heading, Page, ShellSlotsContext } from "./page";

function renderPage(heading: Heading, shell: ShellKind, body: ReactNode) {
  const router = createRouter({
    routeTree: createRootRoute({ component: () => body }),
    history: createMemoryHistory(),
  });
  renderKit(
    <ShellSlotsContext value={{ action: null, inspector: null, heading }}>
      <RouterProvider router={router} />
    </ShellSlotsContext>,
    { shell },
  );
}

const views = (
  <nav aria-label="View">
    <a href="?view=diagram">Diagram</a>
    <a href="?view=roster">Roster</a>
  </nav>
);

const middle = (element: Element) => {
  const box = element.getBoundingClientRect();
  return box.top + box.height / 2;
};

test("in the desktop shell what sits beside the title shares its line", async () => {
  renderPage(
    { kind: "content" },
    "desktop",
    <Page title="Nodes" beside={views}>
      body
    </Page>,
  );
  const title = page.getByRole("heading", { name: "Nodes", level: 1 });
  const beside = page.getByRole("navigation", { name: "View" });
  await expect.element(beside).toBeVisible();
  expect(Math.abs(middle(beside.element()) - middle(title.element()))).toBeLessThan(4);
  expect(beside.element().getBoundingClientRect().left).toBeGreaterThan(
    title.element().getBoundingClientRect().right,
  );
});

test("on a phone's top-level page it sits under the large title", async () => {
  renderPage(
    { kind: "large", onScrolledOut: () => {} },
    "phone",
    <Page title="Nodes" beside={views}>
      body
    </Page>,
  );
  const title = page.getByRole("heading", { name: "Nodes", level: 1 });
  const beside = page.getByRole("navigation", { name: "View" });
  await expect.element(beside).toBeVisible();
  expect(beside.element().getBoundingClientRect().top).toBeGreaterThanOrEqual(
    title.element().getBoundingClientRect().bottom,
  );
});
