import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  getSearchCatalogMockHandler,
  getSearchWorksMockHandler,
} from "../../api/gen/query/works/works.msw";
import { nodeSystems } from "../../test/node-system";
import { homeId } from "../../test/nodes";
import {
  capsule,
  duneId,
  group,
  highlighted,
  label,
  library,
  openFromCapsule,
  optionTexts,
  orchards,
  palette,
  groupLabels,
  pressShortcut,
  wait,
} from "../../test/palette";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";

const noCatalog = () => [getSearchCatalogMockHandler({ candidates: [] }), ...library().handlers];

test("the shortcut opens the palette, and Enter on a typed Work opens its page", async () => {
  const { router } = renderApp({ path: "/library", handlers: library().handlers });
  await expect.element(capsule()).toBeVisible();

  await pressShortcut();
  await expect.element(palette().getByRole("combobox", { name: label })).toHaveFocus();
  await userEvent.keyboard("dun");
  await expect.poll(() => optionTexts("Works")).toEqual(["Dune: Part Two2024 · Film · Missing"]);

  await userEvent.keyboard("{Enter}");
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${duneId}`);
  await expect.element(page.getByRole("heading", { name: "Dune: Part Two" })).toBeVisible();
  await expect.element(palette()).not.toBeInTheDocument();
});

test("at rest the palette lists the commands, and Esc returns focus to the capsule", async () => {
  renderApp({ path: "/library", handlers: library().handlers });
  await userEvent.click(capsule());

  await expect.poll(groupLabels).toEqual(["Commands"]);
  await expect.poll(highlighted).toBe("Go to Library");
  await expect
    .poll(() => optionTexts("Commands"))
    .toEqual([
      "Go to Library",
      "Go to FilmsLibrary",
      "Go to Activity",
      "Go to Nodes",
      "Go to Health",
      "Go to Accounts",
      "Go to Settings",
      "Go to Tasks",
      "Go to Overviewhome",
      "Go to Roleshome",
      "Go to Web and APIhome",
      "Go to Torrent clienthome",
      "Go to Searchinghome",
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
  renderApp({ path: "/library", handlers: library().handlers });
  await expect.element(capsule()).toBeVisible();
  const style = getComputedStyle(capsule().element());
  expect([style.height, style.borderRadius]).toEqual(["32px", "999px"]);
});

test("the palette's group labels are Label type, never uppercase", async () => {
  renderApp({ path: "/library", handlers: library().handlers });
  await userEvent.click(capsule());
  const heading = palette().getByText("Commands", { exact: true });
  await expect.element(heading).toBeVisible();
  const style = getComputedStyle(heading.element());
  expect([style.fontSize, style.fontWeight, style.textTransform]).toEqual(["11px", "500", "none"]);
});

test("the shortcut toggles the palette closed", async () => {
  renderApp({ path: "/library", handlers: library().handlers });
  await expect.element(capsule()).toBeVisible();
  await pressShortcut();
  await expect.element(palette()).toBeVisible();
  await pressShortcut();
  await expect.element(palette()).not.toBeInTheDocument();
});

test("Toggle sidebar keeps the highlight when the catalog answers, and collapses the sidebar", async () => {
  renderApp({ path: "/library", handlers: library().handlers });
  const nav = page.getByRole("navigation", { name: "Main" });
  await expect.poll(() => nav.element().getBoundingClientRect().width).toBe(228);

  await openFromCapsule();
  await userEvent.keyboard("toggle");
  await expect.poll(highlighted).toBe("Toggle sidebar");
  await expect.poll(groupLabels).toEqual(["Add a Film", "Commands"]);
  expect(highlighted()).toBe("Toggle sidebar");
  await userEvent.keyboard("{Enter}");
  await expect.poll(() => nav.element().getBoundingClientRect().width).toBe(60);
});

test("a changed query highlights the top result", async () => {
  renderApp({ path: "/library", handlers: library().handlers });
  await openFromCapsule();
  await userEvent.keyboard("toggle");
  await expect.poll(groupLabels).toEqual(["Add a Film", "Commands"]);
  await expect.poll(highlighted).toBe("Toggle sidebar");
  await userEvent.keyboard("{Backspace}");
  await expect.poll(highlighted).toBe("The Quiet Orchard2021");
});

test("a highlighted Work the answer drops gives the highlight to the top result", async () => {
  let release!: () => void;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  const rows = orchards(3);
  renderApp({
    path: "/library",
    handlers: [
      getSearchWorksMockHandler(async ({ request }) => {
        if (new URL(request.url).searchParams.get("q") !== "orch") {
          return { workIds: [], rows: rows.slice(0, 2) };
        }
        await held;
        return { workIds: [], rows: rows.filter((_, index) => index !== 1) };
      }),
      ...noCatalog(),
    ],
  });
  await openFromCapsule();
  await userEvent.keyboard("orc");
  await expect.poll(() => optionTexts("Works")).toHaveLength(2);
  await userEvent.keyboard("h{ArrowDown}");
  await expect.poll(highlighted).toBe("Orchard 22021 · Film · Missing");

  release();
  await expect
    .poll(() => optionTexts("Works"))
    .toEqual(["Orchard 12021 · Film · Missing", "Orchard 32021 · Film · Missing"]);
  expect(highlighted()).toBe("Orchard 12021 · Film · Missing");
});

test("Add a Film opens the Add page for its Content Type", async () => {
  const { router } = renderApp({ path: "/activity", handlers: library().handlers });
  await userEvent.click(capsule());
  await userEvent.click(group("Commands").getByRole("option", { name: "Add a Film" }));
  await expect.poll(() => router.state.location.href).toBe("/add?contentType=cinema");
});

test("Go to Log settings opens that section of the browser's own Node", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [...library().handlers, ...nodeSystems().handlers],
  });
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
  renderApp({
    path: "/library",
    handlers: noCatalog(),
  });
  await openFromCapsule();
  await userEvent.keyboard("zzz");
  await wait(600);
  await expect
    .element(palette().getByText(/^Nothing matches/u))
    .toHaveTextContent(
      `Nothing matches "zzz". Try a Work's name, or a command such as Add a Film.`,
    );
  expect(palette().getByRole("option").all()).toHaveLength(0);
});

test("on a phone, Search opens the palette full screen with Cancel", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: library().handlers });
  await userEvent.click(
    page.getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "Search" }),
  );

  await expect.element(palette()).toBeVisible();
  const { width, height } = palette().element().getBoundingClientRect();
  expect([width, height]).toEqual([390, 844]);
  await expect.element(palette().getByText("Esc")).not.toBeInTheDocument();
  expect(
    group("Commands").getByRole("option").first().element().getBoundingClientRect().height,
  ).toBe(56);
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
  renderApp({ path: "/library", handlers: library().handlers });
  await userEvent.click(
    page.getByRole("navigation", { name: "Tabs" }).getByRole("button", { name: "More" }),
  );
  await expect.element(page.getByRole("dialog", { name: "More" })).toBeVisible();

  await pressShortcut();
  await expect.element(page.getByRole("dialog", { name: "More" })).toBeVisible();
  expect(palette().query()).toBeNull();
});
