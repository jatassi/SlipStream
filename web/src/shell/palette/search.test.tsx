import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { harvestId } from "../../test/adding";
import {
  capsule,
  group,
  groupLabels,
  library,
  openFromCapsule,
  optionTexts,
  orchards,
  palette,
  wait,
} from "../../test/palette";
import { renderApp } from "../../test/render-app";

test("typing groups the library's Works, then the catalog's results, then commands", async () => {
  const server = library();
  renderApp({ path: "/library", handlers: server.handlers });
  await openFromCapsule();
  await userEvent.keyboard("or");

  await expect.poll(groupLabels).toEqual(["Works", "Add a Film", "Commands"]);
  expect(optionTexts("Works")).toEqual(["The Quiet Orchard2021 · Film · Missing"]);
  expect(optionTexts("Add a Film")).toEqual([
    "The Quiet Orchard2021",
    "Orchard Harvest2019 · In the library",
    "See all resultsFilms",
  ]);
});

test("Works ask the library for its first 8 matches", async () => {
  const server = library(orchards(10));
  renderApp({ path: "/library", handlers: server.handlers });
  await openFromCapsule();
  await userEvent.keyboard("orch");
  await expect.poll(() => optionTexts("Works")).toHaveLength(8);
  expect(optionTexts("Works").at(-1)).toBe("Orchard 82021 · Film · Missing");
  expect(server.asked.at(-1)).toBe("orch 8");
});

test("the catalog is asked once typing pauses, and never for one character", async () => {
  const server = library();
  renderApp({ path: "/library", handlers: server.handlers });
  await openFromCapsule();
  await userEvent.keyboard("o");
  await wait(600);
  expect(server.searched).toEqual([]);

  await userEvent.keyboard("rch");
  await expect.poll(() => optionTexts("Add a Film")).toHaveLength(3);
  await wait(600);
  expect(server.searched).toEqual(["orch"]);
});

test("typing replaces the query in place and pushes no history entry", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library().handlers });
  await expect.element(capsule()).toBeVisible();
  const before = router.history.length;
  await openFromCapsule();
  await userEvent.keyboard("orchard");
  await expect.poll(() => optionTexts("Add a Film")).toHaveLength(3);
  expect(router.history.length).toBe(before);
  expect(router.state.location.href).toBe("/activity");
});

test("a catalog result already in the library opens its Work page", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library().handlers });
  await openFromCapsule();
  await userEvent.keyboard("orch");
  await userEvent.click(group("Add a Film").getByRole("option", { name: /^Orchard Harvest/u }));
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${harvestId}`);
  await expect.element(palette()).not.toBeInTheDocument();
});

test("any other catalog result opens the add sheet over the page", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library().handlers });
  await openFromCapsule();
  await userEvent.keyboard("orch");
  await userEvent.click(group("Add a Film").getByRole("option", { name: /^The Quiet Orchard/u }));
  const sheet = page.getByRole("dialog", { name: "Add The Quiet Orchard" });
  await expect.element(sheet.getByText(/^Adds The Quiet Orchard \(2021\)/u)).toBeVisible();
  await expect.element(palette()).not.toBeInTheDocument();
  expect(router.state.location.href).toBe("/activity");
});

test("See all results opens the add page with the query", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library().handlers });
  await openFromCapsule();
  await userEvent.keyboard("orch");
  await userEvent.click(group("Add a Film").getByRole("option", { name: /^See all results/u }));
  await expect.poll(() => router.state.location.href).toBe("/add?contentType=cinema&query=orch");
});
