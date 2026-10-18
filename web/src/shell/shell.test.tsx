import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksContentType, WorksListedWork } from "../api/gen/model";
import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../test/library";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { asPhone, resize } from "../test/viewport";
import { workServer } from "../test/work-server";

const shows: WorksContentType = {
  ...cinema,
  id: "shows",
  name: "Shows",
  icon: "tv",
  enabled: false,
  work: { one: "Show", many: "Shows" },
};

const mainNav = () => page.getByRole("navigation", { name: "Main" });

const currentLinks = () =>
  page
    .getByRole("link")
    .all()
    .filter((link) => link.element().getAttribute("aria-current") === "page")
    .map((link) => link.element().textContent);

const backLink = () => page.getByRole("banner").getByRole("link");

function counting(first: number) {
  let activity = first;
  return {
    handler: http.get("*/api/v1/counts", () =>
      HttpResponse.json({ activity, alerts: { nodes: null } }),
    ),
    set(next: number) {
      activity = next;
    },
  };
}

const listed = (count: number): WorksListedWork[] =>
  Array.from({ length: count }, (_, index) => ({
    ...orchard,
    id: `${orchardId.slice(0, -2)}${String(index).padStart(2, "0")}`,
    label: `Orchard ${index + 1}`,
  }));

test("the sidebar lists Library with each enabled Content Type, Activity and the system rows", async () => {
  const server = liveServer();
  let contentTypes = [cinema, shows];
  renderApp({
    path: "/library",
    handlers: [...server.handlers, getListContentTypesMockHandler(() => ({ contentTypes }))],
  });
  await expect
    .element(mainNav().getByRole("link", { name: "Films" }))
    .toHaveAttribute("href", "/library/cinema");
  expect(
    mainNav()
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Library", "Films", "Activity", "Nodes", "Accounts", "Settings", "Tasks"]);
  await expect
    .element(mainNav().getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(mainNav().getByRole("link", { name: "Films" }))
    .not.toHaveAttribute("aria-current");
  await expect
    .element(page.getByRole("link", { name: "SlipStream" }))
    .toHaveAttribute("href", "/library");
  expect(currentLinks()).toEqual(["Library"]);

  contentTypes = [cinema, { ...shows, enabled: true }];
  server.clients[0]?.send({
    event: "works.content-types-changed",
    data: { contentType: "shows" },
  });
  await expect
    .element(mainNav().getByRole("link", { name: "Shows" }))
    .toHaveAttribute("href", "/library/shows");
});

test("the sidebar collapses to a 60px rail that a reload keeps", async () => {
  renderApp({ path: "/library" });
  const width = () => mainNav().element().getBoundingClientRect().width;
  await expect.poll(width).toBe(228);

  await userEvent.click(page.getByRole("button", { name: "Collapse sidebar" }));
  await expect.poll(width).toBe(60);
  await expect.element(mainNav().getByRole("link", { name: "Activity" })).toBeVisible();

  renderApp({ path: "/activity" });
  await expect.poll(() => page.getByRole("navigation", { name: "Main" }).all()).toHaveLength(2);
  expect(
    page
      .getByRole("navigation", { name: "Main" })
      .all()
      .map((nav) => nav.element().getBoundingClientRect().width),
  ).toEqual([60, 60]);
});

test("a window below 1100px starts with the rail", async () => {
  await resize(1000, 800);
  renderApp({ path: "/library" });
  await expect.element(page.getByRole("button", { name: "Expand sidebar" })).toBeVisible();
  expect(mainNav().element().getBoundingClientRect().width).toBe(60);
});

test("Activity counts moving jobs and follows grabs.changed", async () => {
  const server = liveServer();
  const counts = counting(2);
  renderApp({ path: "/library", handlers: [...server.handlers, counts.handler] });
  const activity = mainNav().getByRole("link", { name: /^Activity/u });
  await expect.element(activity).toHaveTextContent("Activity2 live");

  counts.set(3);
  server.clients[0]?.send({
    event: "grabs.changed",
    data: { grabId: "7e9a1c3d-5f7b-4d9e-8a1c-5e7f9b1d3a4c", workId: orchardId },
  });
  await expect.element(activity).toHaveTextContent("Activity3 live");
});

test("the tab is titled after the page, and follows navigation", async () => {
  renderApp({ path: "/activity" });
  await expect.poll(() => document.title).toBe("Activity · SlipStream");

  await userEvent.click(mainNav().getByRole("link", { name: "Nodes" }));
  await expect.poll(() => document.title).toBe("home · SlipStream");
  await expect
    .element(mainNav().getByRole("link", { name: "Nodes" }))
    .toHaveAttribute("aria-current", "page");
});

test("a Work's page is titled with its label", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect.poll(() => document.title).toBe("The Quiet Orchard · SlipStream");
});

test("a Work of a disabled Content Type sits under the Library alone", async () => {
  await asPhone();
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] }),
      ...workServer().handlers,
    ],
  });
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect.element(backLink()).toHaveAccessibleName("Library");
  await expect.element(backLink()).toHaveAttribute("href", "/library");
});

test("Add sits under its Content Type's library", async () => {
  await asPhone();
  renderApp({
    path: "/add?contentType=cinema",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema] })],
  });
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  await expect.element(backLink()).toHaveAccessibleName("Films");
  await expect.element(backLink()).toHaveAttribute("href", "/library/cinema");
});

test("the account menu opens the portal", async () => {
  const { router } = renderApp({ path: "/library" });
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Open the portal" }));
  await expect.poll(() => router.state.location.href).toBe("/portal");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
});

test("a Content Type's page lists its Works, is titled with them and adds one", async () => {
  const asked: string[] = [];
  renderApp({
    path: "/library/cinema",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler(({ request }) => {
        asked.push(new URL(request.url).search);
        return { works: listed(1) };
      }),
    ],
  });
  await expect.element(page.getByRole("heading", { name: "Films" })).toBeVisible();
  await expect.poll(() => document.title).toBe("Films · SlipStream");
  expect(asked).toEqual(["?contentType=cinema"]);
  await expect
    .element(page.getByRole("banner").getByRole("button", { name: "Add a Film" }))
    .toHaveAttribute("href", "/add?contentType=cinema");
});

test("a disabled Content Type has no page", async () => {
  renderApp({
    path: "/library/shows",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema, shows] })],
  });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
  await expect.element(mainNav().getByRole("link", { name: "Films" })).toBeVisible();
});
