import { expect, test } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { problem } from "../../../test/handlers";
import { joinCodeServer } from "../../../test/join-codes";
import { ids, manageServer, threeNodes } from "../../../test/landing";
import { renderApp } from "../../../test/render-app";
import { outcome, refusal } from "../../../test/roles";
import { asPhone, resize } from "../../../test/viewport";

const views = () => page.getByRole("navigation", { name: "View" });
const roster = () => page.getByRole("list", { name: "Roster" });
const diagram = () => page.getByRole("figure", { name: "Diagram" });
const rowOf = (name: string) =>
  roster()
    .getByRole("listitem")
    .filter({ has: page.getByRole("link", { name: new RegExp(`^${name}`, "u") }) });

const textOf = (locator: Locator) => () => locator.element().textContent;

async function openMenu(name: string, item: string) {
  await userEvent.click(page.getByRole("button", { name: `${name} actions` }));
  await userEvent.click(page.getByRole("menuitem", { name: item }));
}

test("a desktop browser opens Nodes on the Diagram, which takes drags, and the Roster tab lists the same Nodes", async () => {
  const { router } = renderApp({ path: "/nodes", handlers: threeNodes });
  await expect.element(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  await expect.element(page.getByText("seedbox is offline.")).toBeVisible();
  await expect
    .element(views().getByRole("link", { name: "Diagram" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(diagram()).toHaveAttribute("data-takes-drags", "true");
  await expect.element(page.getByRole("button", { name: "Add a Node" })).toBeVisible();

  await userEvent.click(views().getByRole("link", { name: "Roster" }));
  expect(router.state.location.search).toEqual({ view: "roster" });
  await expect.element(roster()).toBeVisible();
  await expect
    .element(views().getByRole("link", { name: "Roster" }))
    .toHaveAttribute("aria-current", "page");
});

test("below 1000px Nodes opens on the Roster, and the Diagram is there to look at, without drags", async () => {
  await resize(900, 800);
  renderApp({ path: "/nodes", handlers: threeNodes });
  await expect.element(roster()).toBeVisible();
  await expect
    .element(views().getByRole("link", { name: "Roster" }))
    .toHaveAttribute("aria-current", "page");
  await userEvent.click(views().getByRole("link", { name: "Diagram" }));
  await expect.element(diagram()).toHaveAttribute("data-takes-drags", "false");
});

test("on phone Nodes is the top page and opens on the Roster", async () => {
  await asPhone();
  renderApp({ path: "/nodes", handlers: threeNodes });
  await expect.element(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  await expect.element(roster()).toBeVisible();
  await userEvent.click(views().getByRole("link", { name: "Diagram" }));
  await expect.element(diagram()).toHaveAttribute("data-takes-drags", "false");
});

test("each Roster row says the Node's state, when it was last seen, how it is reached, its Roles and what its probes found", async () => {
  renderApp({ path: "/nodes?view=roster", handlers: threeNodes });
  const homeRow = rowOf("home");
  await expect.poll(textOf(homeRow)).toMatch(/^homeOnline · ControllerSearchingDownloading/u);
  const seedboxRow = rowOf("seedbox");
  await expect
    .poll(textOf(seedboxRow))
    .toMatch(/Offline · Last seen .+Downloading · Draining · 3 torrents \(41 GB\) left/u);
  await expect
    .element(seedboxRow.getByRole("list", { name: "What seedbox found" }))
    .toHaveTextContent("Weak: The downloads folder is on a network share.");
  await expect
    .element(rowOf("vps"))
    .toHaveTextContent("vpsOnline · home dials vps · 24 ms round tripSearching");
  expect(
    roster()
      .getByRole("link")
      .all()
      .map((link) => link.element().getAttribute("href")),
  ).toEqual([`/nodes/${ids.home}`, `/nodes/${ids.seedbox}`, `/nodes/${ids.vps}`]);
});

test("a Roster row opens its Node's Overview", async () => {
  const { router } = renderApp({ path: "/nodes?view=roster", handlers: threeNodes });
  await userEvent.click(rowOf("vps").getByRole("link"));
  await expect.poll(() => router.state.location.pathname).toBe(`/nodes/${ids.vps}`);
  await expect.element(page.getByRole("heading", { name: "vps", level: 1 })).toBeVisible();
});

test("with one Node the Roster shows it complete and offers Add a Node in its own row", async () => {
  const codes = joinCodeServer();
  renderApp({ path: "/nodes?view=roster", handlers: codes.handlers });
  await expect.element(rowOf("home")).toBeVisible();
  await userEvent.click(roster().getByRole("button", { name: /^Add a Node/u }));
  await expect.element(page.getByRole("dialog", { name: "Add a Node" })).toBeVisible();
});

test("Rename from the Roster says when the name is taken, then renames", async () => {
  const server = manageServer({
    renames: [() => problem(409, "node_name_taken", "Another Node has this name")],
  });
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("vps", "Rename");
  const dialog = page.getByRole("dialog", { name: "Rename vps" });
  const name = dialog.getByRole("textbox", { name: "Name" });
  await userEvent.fill(name, "seedbox");
  await userEvent.click(dialog.getByRole("button", { name: "Rename" }));
  await expect.element(name).toHaveAccessibleDescription("Another Node has this name");

  await userEvent.fill(name, " edge ");
  await userEvent.click(dialog.getByRole("button", { name: "Rename" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.renamed).toEqual([
    { nodeId: ids.vps, body: { name: "seedbox" } },
    { nodeId: ids.vps, body: { name: "edge" } },
  ]);
});

test("Remove from the Roster says what the Node still holds and writes it off", async () => {
  const server = manageServer({});
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("seedbox", "Remove");
  const dialog = page.getByRole("dialog", { name: "Remove seedbox?" });
  await expect
    .poll(textOf(dialog))
    .toMatch(
      /seedbox still holds 3 torrents \(41 GB\) for Downloading\. Untick its Roles and remove it when they finish, or remove it now and write them off\./u,
    );
  await userEvent.click(dialog.getByRole("button", { name: "Remove and write off" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.removed).toEqual([{ nodeId: ids.seedbox, writeOff: "true" }]);
});

test("a Node that started holding work since the Roster loaded is refused, then removed with a write-off", async () => {
  const held = outcome({
    checks: [
      {
        verdict: "block",
        reason: "node_holds_work",
        node: { id: ids.vps, name: "vps" },
        role: "downloading",
        words:
          "vps still holds 1 torrent for Downloading. Untick its Roles and remove it when they finish, or remove it now and write them off.",
      },
    ],
  });
  const server = manageServer({
    removes: [() => refusal("node_holds_work", "The Node holds work", held)],
  });
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("vps", "Remove");
  const dialog = page.getByRole("dialog", { name: "Remove vps?" });
  await userEvent.click(dialog.getByRole("button", { name: "Remove", exact: true }));
  await expect
    .element(dialog.getByRole("alert"))
    .toHaveTextContent("vps started holding work, so nothing was removed.");
  await expect.poll(textOf(dialog)).toMatch(/vps still holds 1 torrent for Downloading\./u);
  await userEvent.click(dialog.getByRole("button", { name: "Remove and write off" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.removed).toEqual([
    { nodeId: ids.vps, writeOff: "false" },
    { nodeId: ids.vps, writeOff: "true" },
  ]);
});

test("the Controller can't be removed, and Remove says to hand it over", async () => {
  const server = manageServer({});
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("home", "Remove");
  const dialog = page.getByRole("dialog", { name: "home can't be removed" });
  await expect.poll(textOf(dialog)).toMatch(/hand the Controller over to another Node first/u);
  await userEvent.click(dialog.getByRole("button", { name: "Close" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.removed).toEqual([]);
});

test("Write off is on the menu only of a Node that holds work, and opens the write-off", async () => {
  renderApp({ path: "/nodes?view=roster", handlers: threeNodes });
  await userEvent.click(page.getByRole("button", { name: "vps actions" }));
  await expect.element(page.getByRole("menuitem", { name: "Rename" })).toBeVisible();
  await expect.element(page.getByRole("menuitem", { name: "Write off" })).not.toBeInTheDocument();
  await userEvent.keyboard("{Escape}");
  await openMenu("seedbox", "Write off");
  await expect
    .poll(textOf(page.getByRole("dialog", { name: "Write off what seedbox holds" })))
    .toMatch(/seedbox holds 3 torrents \(41 GB\) for Downloading\./u);
});
