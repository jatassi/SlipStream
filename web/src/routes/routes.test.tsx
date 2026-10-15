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
const pages = Object.values(routesById).filter(
  (route) => route.id.startsWith("/_app/") && route.options.component !== undefined,
);

test("every signed-in page declares its title", () => {
  expect(pages.length).toBeGreaterThan(0);
  expect(
    pages.filter((route) => route.options.staticData?.title === undefined).map((route) => route.id),
  ).toEqual([]);
});
