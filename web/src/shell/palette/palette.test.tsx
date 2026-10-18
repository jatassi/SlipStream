import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksListedWork, WorksTree } from "../../api/gen/model";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../../test/library";
import { homeId } from "../../test/nodes";
import { asPhone } from "../../test/viewport";
import { paletteShortcut } from "../palette-state";
import { renderApp } from "../../test/render-app";

const label = "Search Works or run a command";
const duneId = "6f1d3b5a-9c7e-4b2d-8f0a-1c3e5a7b9d2f";
const dune: WorksTree = {
  ...orchard,
  id: duneId,
  key: "9000002",
  label: "Dune: Part Two",
  sortLabel: "Dune: Part Two",
};

const orchards = (count: number): WorksListedWork[] =>
  Array.from({ length: count }, (_, index) => ({
    ...orchard,
    id: `${orchardId.slice(0, -2)}${String(index).padStart(2, "0")}`,
    label: `Orchard ${index + 1}`,
    sortLabel: `Orchard ${index + 1}`,
  }));

const library = (works: WorksListedWork[] = [orchard, dune]) => [
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
  getListWorksMockHandler({ works }),
  getGetWorkMockHandler(dune),
];

const palette = () => page.getByRole("dialog", { name: label });
const capsule = () => page.getByRole("banner").getByRole("button", { name: label });
const group = (name: string) => palette().getByRole("group", { name });
const optionTexts = (name: string) =>
  group(name)
    .getByRole("option")
    .all()
    .map((option) => option.element().textContent);

const [modifier] = paletteShortcut.keys.split("+");
const pressShortcut = () => userEvent.keyboard(`{${modifier}>}k{/${modifier}}`);
const openFromCapsule = async () => {
  await userEvent.click(capsule());
  await expect.element(palette().getByRole("combobox", { name: label })).toHaveFocus();
};

test("the shortcut opens the palette, and Enter on a typed Work opens its page", async () => {
  const { router } = renderApp({ path: "/library", handlers: library() });
  await expect.element(capsule()).toBeVisible();

  await pressShortcut();
  await expect.element(palette().getByRole("combobox", { name: label })).toHaveFocus();
  await userEvent.keyboard("dun");
  await expect.poll(() => optionTexts("Works")).toEqual(["Dune: Part TwoFilm"]);

  await userEvent.keyboard("{Enter}");
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${duneId}`);
  await expect.element(page.getByRole("heading", { name: "Dune: Part Two" })).toBeVisible();
  await expect.element(palette()).not.toBeInTheDocument();
});

test("the palette groups Works then Commands, and Esc returns focus to the capsule", async () => {
  renderApp({ path: "/library", handlers: library() });
  await userEvent.click(capsule());

  await expect
    .poll(() =>
      palette()
        .getByRole("group")
        .all()
        .map((element) => element.element().firstElementChild?.textContent),
    )
    .toEqual(["Works", "Commands"]);
  expect(optionTexts("Works")).toEqual(["The Quiet OrchardFilm", "Dune: Part TwoFilm"]);
  await expect
    .poll(() => optionTexts("Commands"))
    .toEqual([
      "Go to Library",
      "Go to FilmsLibrary",
      "Go to Activity",
      "Go to Nodes",
      "Go to Accounts",
      "Go to Settings",
      "Go to Tasks",
      "Go to Overviewhome",
      "Go to Roleshome",
      "Go to Web and APIhome",
      "Go to Logshome",
      "Go to Log settingshome",
      "Add a Film",
      "Toggle sidebar",
      "New Signup link",
      "Your Account",
      "Add a passkey",
      "Restart SlipStream…",
      "Sign out",
    ]);
  await expect.element(palette().getByText("Esc")).toBeVisible();

  await userEvent.keyboard("{Escape}");
  await expect.element(palette()).not.toBeInTheDocument();
  await expect.element(capsule()).toHaveFocus();
});

test("the top bar's search capsule is a 32px Console capsule", async () => {
  renderApp({ path: "/library", handlers: library() });
  await expect.element(capsule()).toBeVisible();
  const style = getComputedStyle(capsule().element());
  expect([style.height, style.borderRadius]).toEqual(["32px", "999px"]);
});

test("the palette's group labels are Label type, never uppercase", async () => {
  renderApp({ path: "/library", handlers: library() });
  await userEvent.click(capsule());
  const heading = palette().getByText("Commands", { exact: true });
  await expect.element(heading).toBeVisible();
  const style = getComputedStyle(heading.element());
  expect([style.fontSize, style.fontWeight, style.textTransform]).toEqual(["11px", "500", "none"]);
});

test("the shortcut toggles the palette closed", async () => {
  renderApp({ path: "/library", handlers: library() });
  await expect.element(capsule()).toBeVisible();
  await pressShortcut();
  await expect.element(palette()).toBeVisible();
  await pressShortcut();
  await expect.element(palette()).not.toBeInTheDocument();
});

test("Works show 5 at rest and 8 while typing", async () => {
  renderApp({ path: "/library", handlers: library(orchards(10)) });
  await userEvent.click(capsule());
  await expect
    .poll(() => optionTexts("Works"))
    .toEqual(["Orchard 1Film", "Orchard 2Film", "Orchard 3Film", "Orchard 4Film", "Orchard 5Film"]);

  await userEvent.keyboard("orch");
  await expect.poll(() => optionTexts("Works")).toHaveLength(8);
  expect(optionTexts("Works").at(-1)).toBe("Orchard 8Film");
});

test("Toggle sidebar collapses the sidebar", async () => {
  renderApp({ path: "/library", handlers: library() });
  const nav = page.getByRole("navigation", { name: "Main" });
  await expect.poll(() => nav.element().getBoundingClientRect().width).toBe(228);

  await openFromCapsule();
  await userEvent.keyboard("toggle");
  await expect.poll(() => optionTexts("Commands")).toEqual(["Toggle sidebar"]);
  await userEvent.keyboard("{Enter}");
  await expect.poll(() => nav.element().getBoundingClientRect().width).toBe(60);
});

test("Add a Film opens the Add page for its Content Type", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library() });
  await userEvent.click(capsule());
  await userEvent.click(group("Commands").getByRole("option", { name: "Add a Film" }));
  await expect.poll(() => router.state.location.href).toBe("/add?contentType=cinema");
});

test("Go to Log settings opens that section of the browser's own Node", async () => {
  const { router } = renderApp({ path: "/library", handlers: library() });
  await openFromCapsule();
  await userEvent.keyboard("log set");
  await expect.poll(() => optionTexts("Commands")).toEqual(["Go to Log settingshome"]);
  await userEvent.keyboard("{Enter}");
  await expect.poll(() => router.state.location.href).toBe(`/nodes/${homeId}/log-settings`);
  await expect
    .element(
      page.getByRole("navigation", { name: "home" }).getByRole("link", { name: "Log settings" }),
    )
    .toHaveAttribute("aria-current", "page");
});

test("an unmatched query suggests what to try", async () => {
  renderApp({ path: "/library", handlers: library() });
  await openFromCapsule();
  await userEvent.keyboard("zzz");
  await expect
    .element(palette().getByText(/^Nothing matches/u))
    .toHaveTextContent(
      `Nothing matches "zzz". Try a Work's name, or a command such as Add a Film.`,
    );
  expect(palette().getByRole("option").all()).toHaveLength(0);
});

test("on a phone, Search opens the palette full screen with Cancel", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: library() });
  await userEvent.click(
    page.getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "Search" }),
  );

  await expect.element(palette()).toBeVisible();
  const { width, height } = palette().element().getBoundingClientRect();
  expect([width, height]).toEqual([390, 844]);
  await expect.element(palette().getByText("Esc")).not.toBeInTheDocument();
  expect(group("Works").getByRole("option").first().element().getBoundingClientRect().height).toBe(
    56,
  );
  expect(
    palette()
      .getByRole("group", { name: "Commands" })
      .getByRole("option", { name: "Toggle sidebar" })
      .query(),
  ).toBeNull();

  await userEvent.click(palette().getByRole("button", { name: "Cancel" }));
  await expect.element(palette()).not.toBeInTheDocument();
});

test("the shortcut is inert while another modal is open", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: library() });
  await userEvent.click(
    page.getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" }),
  );
  await expect.element(page.getByRole("dialog", { name: "More" })).toBeVisible();

  await pressShortcut();
  await expect.element(page.getByRole("dialog", { name: "More" })).toBeVisible();
  expect(palette().query()).toBeNull();
});
