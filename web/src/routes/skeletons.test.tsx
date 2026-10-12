import { createMemoryHistory } from "@tanstack/react-router";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { createApp } from "../app";
import { pendingForever } from "../test/handlers";
import { renderApp } from "../test/render-app";

const zeroUuid = "00000000-0000-0000-0000-000000000000";
const { router, dispose } = createApp(createMemoryHistory());
dispose();
type Walked = {
  id: string;
  fullPath: string;
  options: { loader?: unknown; pendingComponent?: unknown };
};

const routesById: Record<string, Walked> = { ...router.routesById };
const signedInPages = Object.values(routesById).filter(
  (route) => route.id.startsWith("/_app/") && route.options.loader !== undefined,
);

test("every signed-in page that loads data has a skeleton", () => {
  expect(signedInPages.length).toBeGreaterThan(0);
  expect(
    signedInPages
      .filter((route) => route.options.pendingComponent === undefined)
      .map((route) => route.id),
  ).toEqual([]);
});

test.each(signedInPages.map((route) => [route.id, route.fullPath.replaceAll(/\$\w+/gu, zeroUuid)]))(
  "%s shows its skeleton while its data loads",
  async (_id, path) => {
    renderApp({ path, handlers: pendingForever() });
    await expect
      .element(page.getByRole("status").and(page.getByLabelText(/^Loading/u)))
      .toBeVisible();
  },
);
