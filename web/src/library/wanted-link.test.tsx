import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { orchardRow } from "../test/library";
import { cinemaLibrary, ember, harvest } from "../test/library-page";
import { renderApp } from "../test/render-app";
import { wantedServer } from "../test/wanted";

const link = () => page.getByRole("list", { name: "Wanted" }).getByRole("link");

test("the Library counts what Wanted lists, its failed Works apart, and opens Wanted", async () => {
  const rows = [
    { ...orchardRow, wanted: "missing" as const },
    { ...harvest, wanted: "gone" as const },
    { ...ember, wanted: "failed" as const },
  ];
  const { router } = renderApp({
    path: "/library",
    handlers: [...cinemaLibrary(rows).handlers, ...wantedServer().handlers],
  });
  await expect
    .element(link())
    .toMatchTextContent(/^2 films need files or upgrades\. 1 has a failed Grab\./u);
  await userEvent.click(link());
  await expect.poll(() => router.state.location.pathname).toBe("/library/wanted/cinema");
});

test("with nothing wanted the Library has no Wanted row", async () => {
  renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await expect.element(page.getByRole("link", { name: "The Quiet Orchard" })).toBeVisible();
  expect(page.getByRole("list", { name: "Wanted" }).query()).toBeNull();
});
