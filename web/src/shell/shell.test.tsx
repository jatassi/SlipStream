import { HttpResponse, http } from "msw";
import { expect, onTestFinished, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksContentType, WorksListedWork } from "../api/gen/model";
import { getGetSystemMockHandler } from "../api/gen/query/system/system.msw";
import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../test/library";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
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
const tabs = () => page.getByRole("navigation", { name: "Tabs" });

async function asPhone() {
  await page.viewport(390, 844);
  onTestFinished(async () => {
    await page.viewport(1440, 900);
  });
}

function counting(first: number) {
  let activity = first;
  return {
    handler: http.get("*/api/v1/counts", () => HttpResponse.json({ activity })),
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

test("the sidebar lists Library with each enabled Content Type, Activity and This Node", async () => {
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
  ).toEqual(["Library", "Films", "Activity", "This Node"]);
  await expect
    .element(mainNav().getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(mainNav().getByRole("link", { name: "Films" }))
    .not.toHaveAttribute("aria-current");

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
  await page.viewport(1000, 800);
  onTestFinished(async () => {
    await page.viewport(1440, 900);
  });
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

  await userEvent.click(mainNav().getByRole("link", { name: "This Node" }));
  await expect.poll(() => document.title).toBe("This Node · SlipStream");
  await expect
    .element(page.getByRole("navigation", { name: "Breadcrumbs" }))
    .toHaveTextContent("This Node");
});

test("a Work's page is titled with its label", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect.poll(() => document.title).toBe("The Quiet Orchard · SlipStream");
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
    .element(page.getByRole("navigation", { name: "Breadcrumbs" }))
    .toHaveTextContent("LibraryFilms");
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

test("the phone shell has a tab bar, with the Activity count as a badge", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: [counting(2).handler] });
  await expect
    .poll(() =>
      tabs()
        .getByRole("listitem")
        .all()
        .map((tab) => tab.element().textContent),
    )
    .toEqual(["Library", "Activity2", "Search", "More"]);
  await expect
    .element(tabs().getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(mainNav()).not.toBeInTheDocument();
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
});

test("More opens a sheet with the rest of the nav and the version, and Cancel closes it", async () => {
  await asPhone();
  const { router } = renderApp({
    path: "/library",
    handlers: [
      getGetSystemMockHandler({
        build: { stamped: true, version: "0.4.0" },
        dataDir: "/data/home",
        listeners: {
          http: "127.0.0.2:8080",
          torrent: "127.0.0.2:6881",
          link: "127.0.0.2:8485",
          control: "/data/home/control.sock",
        },
        pid: 4242,
        seed: "home",
        startedAt: "2026-10-12T09:00:00Z",
      }),
    ],
  });
  await userEvent.click(tabs().getByRole("button", { name: "More" }));
  const sheet = page.getByRole("dialog", { name: "More" });
  await expect.element(sheet.getByRole("button", { name: "This Node" })).toBeVisible();
  await expect.element(sheet.getByText("SlipStream 0.4.0")).toBeVisible();
  await expect.element(sheet.getByRole("button", { name: "Sign out" })).toBeVisible();
  await expect
    .element(sheet.getByRole("button", { name: "Startup sequence" }))
    .toHaveAttribute("aria-pressed", "true");

  await userEvent.click(sheet.getByRole("button", { name: "Cancel" }));
  await expect.element(sheet).not.toBeInTheDocument();

  await userEvent.click(tabs().getByRole("button", { name: "More" }));
  await userEvent.click(sheet.getByRole("button", { name: "This Node" }));
  await expect.element(sheet).not.toBeInTheDocument();
  await expect.poll(() => router.state.location.href).toBe("/system");
  await expect
    .element(tabs().getByRole("button", { name: "More" }))
    .not.toHaveAttribute("aria-current");
});

test("on a phone a menu opens as an action sheet", async () => {
  await asPhone();
  renderApp({
    path: "/library",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema] })],
  });
  await userEvent.click(page.getByRole("button", { name: "Add" }));
  const sheet = page.getByRole("dialog", { name: "Add a Work" });
  await expect.element(sheet.getByRole("button", { name: "Add a Film" })).toBeVisible();
  await userEvent.click(sheet.getByRole("button", { name: "Cancel" }));
  await expect.element(sheet).not.toBeInTheDocument();
});

test("the large title hands its words to the nav bar once it scrolls away", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: [getListWorksMockHandler({ works: listed(40) })] });
  const barTitle = page.getByRole("banner").getByText("Library");
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
  await expect.poll(() => getComputedStyle(barTitle.element()).opacity).toBe("0");

  window.scrollTo({ top: 400 });
  await expect.poll(() => getComputedStyle(barTitle.element()).opacity).toBe("1");
  window.scrollTo({ top: 0 });
});

test("a pushed page goes back to its tab, and a tab returns to where it was", async () => {
  await asPhone();
  const server = workServer();
  const { router } = renderApp({
    path: "/library/cinema",
    handlers: [...server.handlers, getListWorksMockHandler({ works: [{ ...orchard }] })],
  });
  await userEvent.click(page.getByRole("link", { name: "The Quiet Orchard 2021" }));
  const back = page.getByRole("banner").getByRole("link", { name: "Library" });
  await expect.element(back).toHaveAttribute("href", "/library");
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();

  await userEvent.click(tabs().getByRole("link", { name: "Activity" }));
  await expect.poll(() => router.state.location.href).toBe("/activity");
  await userEvent.click(tabs().getByRole("link", { name: "Library" }));
  await expect.poll(() => router.state.location.href).toBe(`/works/${orchardId}`);

  await userEvent.click(back);
  await expect.poll(() => router.state.location.href).toBe("/library");
});
