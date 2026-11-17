import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { calendarServer } from "../test/calendar";
import { cinemaLibrary, twoTypeLibrary } from "../test/library-page";
import { label } from "../test/palette";
import { renderApp } from "../test/render-app";
import { wantedServer } from "../test/wanted";

const views = () => page.getByRole("navigation", { name: "Library views" });
const view = (name: string) => views().getByRole("link", { name, exact: true });

const hrefs = () =>
  views()
    .getByRole("link")
    .elements()
    .map((link) => [link.textContent, link.getAttribute("href")]);

const current = () =>
  views()
    .getByRole("link")
    .elements()
    .filter((link) => link.getAttribute("aria-current") === "page")
    .map((link) => link.textContent);

async function runCommand(command: string) {
  await userEvent.click(page.getByRole("button", { name: label }));
  await userEvent.keyboard(command);
  await userEvent.click(page.getByRole("option", { name: command }));
}

const pages = () => [
  ...cinemaLibrary().handlers,
  ...wantedServer().handlers,
  ...calendarServer().handlers.slice(1),
];

test("a Content Type's Library, Wanted and Calendar link to each other, keeping the type", async () => {
  const { router } = renderApp({ path: "/library/cinema", handlers: pages() });
  await expect.element(view("Library")).toHaveAttribute("aria-current", "page");
  expect(hrefs()).toEqual([
    ["Library", "/library/cinema"],
    ["Wanted", "/library/wanted/cinema"],
    ["Calendar", "/library/calendar/cinema"],
  ]);

  await userEvent.click(view("Wanted"));
  await expect.element(page.getByRole("heading", { name: "Wanted", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/library/wanted/cinema");
  expect(current()).toEqual(["Wanted"]);

  await userEvent.click(view("Calendar"));
  await expect.element(page.getByRole("heading", { name: "Calendar", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/library/calendar/cinema");
  expect(current()).toEqual(["Calendar"]);
  expect(hrefs()).toEqual([
    ["Library", "/library/cinema"],
    ["Wanted", "/library/wanted/cinema"],
    ["Calendar", "/library/calendar/cinema"],
  ]);

  await userEvent.click(view("Library"));
  await expect.poll(() => router.state.location.pathname).toBe("/library/cinema");
  await expect.element(view("Library")).toHaveAttribute("aria-current", "page");
});

test("on All the views cover every Content Type, and the palette still opens Wanted and the Calendar", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [
      ...twoTypeLibrary(),
      ...wantedServer().handlers,
      ...calendarServer().handlers.slice(1),
    ],
  });
  await expect.element(view("Library")).toHaveAttribute("aria-current", "page");
  expect(hrefs()).toEqual([
    ["Library", "/library"],
    ["Wanted", "/library/wanted"],
    ["Calendar", "/library/calendar"],
  ]);

  await runCommand("Open Wanted");
  await expect.element(view("Wanted")).toHaveAttribute("aria-current", "page");
  await runCommand("Open Calendar");
  await expect.element(view("Calendar")).toHaveAttribute("aria-current", "page");
  expect(router.state.location.pathname).toBe("/library/calendar");
});
