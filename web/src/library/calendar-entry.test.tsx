import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { calendarServer } from "../test/calendar";
import { cinemaLibrary, twoTypeLibrary } from "../test/library-page";
import { label } from "../test/palette";
import { renderApp } from "../test/render-app";

const entry = () => page.getByRole("list", { name: "Calendar" }).getByRole("link");

test("the Library of one Content Type opens that type's Calendar", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [...cinemaLibrary().handlers, ...calendarServer().handlers.slice(1)],
  });
  await expect.element(entry()).toMatchTextContent(/Release dates in your library, day by day\./u);
  await userEvent.click(entry());
  await expect.poll(() => router.state.location.pathname).toBe("/library/calendar/cinema");
  await expect.element(page.getByRole("heading", { name: "Calendar", level: 1 })).toBeVisible();
});

test("a Library of every Content Type opens the whole Calendar, as Open Calendar does", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [...twoTypeLibrary(), ...calendarServer().handlers.slice(1)],
  });
  await userEvent.click(entry());
  await expect.poll(() => router.state.location.pathname).toBe("/library/calendar");
  await router.navigate({ to: "/library" });
  await userEvent.click(page.getByRole("button", { name: label }));
  await userEvent.keyboard("Open Calendar");
  await userEvent.click(page.getByRole("option", { name: "Open Calendar" }));
  await expect.poll(() => router.state.location.pathname).toBe("/library/calendar");
});
