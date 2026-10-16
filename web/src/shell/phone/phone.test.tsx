import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksListedWork } from "../../api/gen/model";
import { getGetSystemMockHandler } from "../../api/gen/query/system/system.msw";
import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../../test/library";
import { asPhone } from "../../test/viewport";
import { renderApp } from "../../test/render-app";
import { workServer } from "../../test/work-server";

const mainNav = () => page.getByRole("navigation", { name: "Main" });
const tabs = () => page.getByRole("navigation", { name: "Tabs" });

function counting(activity: number) {
  return http.get("*/api/v1/counts", () => HttpResponse.json({ activity }));
}

const listed = (count: number): WorksListedWork[] =>
  Array.from({ length: count }, (_, index) => ({
    ...orchard,
    id: `${orchardId.slice(0, -2)}${String(index).padStart(2, "0")}`,
    label: `Orchard ${index + 1}`,
  }));

test("the phone shell has a tab bar, with the Activity count as a badge", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: [counting(2)] });
  await expect
    .poll(() =>
      tabs()
        .getByRole("listitem")
        .all()
        .map((tab) => tab.element().textContent),
    )
    .toEqual(["Library", "Activity2", "Search", "More"]);
  const badge = getComputedStyle(tabs().getByText("2", { exact: true }).element());
  expect([badge.fontSize, badge.fontWeight, badge.height]).toEqual(["12px", "600", "18px"]);
  await expect
    .element(tabs().getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(mainNav()).not.toBeInTheDocument();
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
});

const moreButtons = [
  "Films",
  "This Node",
  "Tasks",
  "Your Account",
  "Open the portal",
  "Sign out",
  "Cancel",
];

test("More opens a sheet with the rest of the nav and the version, and Cancel closes it", async () => {
  await asPhone();
  const { router } = renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
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
  expect(
    sheet
      .getByRole("button")
      .all()
      .map((button) => button.element().textContent),
  ).toEqual(moreButtons);
  const startup = sheet.getByRole("switch", { name: "Startup sequence" });
  await expect.element(startup).toBeChecked();
  await userEvent.click(sheet.getByText("Startup sequence"));
  await expect.element(startup).not.toBeChecked();
  await expect.element(sheet).toBeVisible();

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

test("More reaches each Content Type's library under the Library tab", async () => {
  await asPhone();
  const { router } = renderApp({
    path: "/activity",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema] })],
  });
  await userEvent.click(tabs().getByRole("button", { name: "More" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "More" }).getByRole("button", { name: "Films" }),
  );
  await expect.poll(() => router.state.location.href).toBe("/library/cinema");
  await expect.element(page.getByRole("heading", { name: "Films", level: 1 })).toBeVisible();
  await expect
    .element(tabs().getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "true");
});

test("More opens the portal", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/library" });
  await userEvent.click(tabs().getByRole("button", { name: "More" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "More" }).getByRole("button", { name: "Open the portal" }),
  );
  await expect.poll(() => router.state.location.href).toBe("/portal");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
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

test("a pushed page goes back to its library, and a tab returns to where it was", async () => {
  await asPhone();
  const server = workServer();
  const { router } = renderApp({
    path: "/library/cinema",
    handlers: [...server.handlers, getListWorksMockHandler({ works: [{ ...orchard }] })],
  });
  await userEvent.click(page.getByRole("link", { name: "The Quiet Orchard 2021" }));
  const back = page.getByRole("banner").getByRole("link", { name: "Films" });
  await expect.element(back).toHaveAttribute("href", "/library/cinema");
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();

  await userEvent.click(tabs().getByRole("link", { name: "Activity" }));
  await expect.poll(() => router.state.location.href).toBe("/activity");
  await userEvent.click(tabs().getByRole("link", { name: "Library" }));
  await expect.poll(() => router.state.location.href).toBe(`/works/${orchardId}`);

  await userEvent.click(back);
  await expect.poll(() => router.state.location.href).toBe("/library/cinema");
});
