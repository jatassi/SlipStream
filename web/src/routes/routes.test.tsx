import { createMemoryHistory } from "@tanstack/react-router";
import { expect, test } from "vitest";

import { createApp } from "../app";

const { router, dispose } = createApp(createMemoryHistory());
dispose();

type Walked = {
  id: string;
  options: { component?: unknown; staticData?: { title?: unknown } };
};

const routesById: Record<string, Walked> = { ...router.routesById };
const pagesUnder = (layout: string) =>
  Object.values(routesById).filter(
    (route) => route.id.startsWith(`/${layout}/`) && route.options.component !== undefined,
  );

test.each(["_app", "portal"])("every page under %s declares its title", (layout) => {
  const pages = pagesUnder(layout);
  expect(pages.length).toBeGreaterThan(0);
  expect(
    pages.filter((route) => route.options.staticData?.title === undefined).map((route) => route.id),
  ).toEqual([]);
});
