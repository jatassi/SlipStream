import { createMemoryHistory } from "@tanstack/react-router";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { createApp } from "../app";
import { settingsGroups } from "../settings/sections";
import { pendingForever } from "../test/handlers";
import { renderApp } from "../test/render-app";

const zeroUuid = "00000000-0000-0000-0000-000000000000";
const paramSamples: Readonly<Record<string, string>> = {
  section: settingsGroups[0].sections[0].id,
};
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

test.each(
  signedInPages.map((route) => [
    route.id,
    route.fullPath.replaceAll(
      /\$(\w+)/gu,
      (_param, name: string) => paramSamples[name] ?? zeroUuid,
    ),
  ]),
)("%s shows its skeleton while its data loads", async (_id, path) => {
  renderApp({ path, handlers: pendingForever() });
  await expect
    .element(page.getByRole("status").and(page.getByLabelText(/^Loading/u)))
    .toBeVisible();
});

test("a page whose route names its title shows the title while its data loads", async () => {
  renderApp({ path: "/health", handlers: pendingForever() });
  await expect.element(page.getByRole("status", { name: "Loading Health" })).toBeVisible();
  await expect.element(page.getByRole("heading", { level: 1, name: "Health" })).toBeVisible();
});

const posterFrames = (status: Element) =>
  [...status.querySelectorAll("span")].filter(
    (span) => getComputedStyle(span).aspectRatio === "2 / 3",
  );

test("the Library's skeleton takes the layout and poster size its address names", async () => {
  renderApp({ path: "/library?view=table", handlers: pendingForever() });
  const status = page.getByRole("status", { name: "Loading the Library" });
  await expect.element(status).toBeVisible();
  expect(posterFrames(status.element())).toHaveLength(0);
});

test("the Library's skeleton draws large posters when the address names them", async () => {
  renderApp({ path: "/library?size=large", handlers: pendingForever() });
  const status = page.getByRole("status", { name: "Loading the Library" });
  await expect.element(status).toBeVisible();
  const [frame] = posterFrames(status.element());
  expect(frame?.getBoundingClientRect().width).toBeGreaterThanOrEqual(240);
});

test("the Work page's skeleton draws its poster beside its title", async () => {
  renderApp({ path: `/works/${zeroUuid}`, handlers: pendingForever() });
  const status = page.getByRole("status", { name: "Loading the Work" });
  await expect.element(status).toBeVisible();
  expect(posterFrames(status.element())).toHaveLength(1);
});
