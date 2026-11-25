import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { joinCodeServer } from "../../../test/join-codes";
import { ids, manageServer, threeNodes } from "../../../test/landing";
import { nothingSetUp } from "../../../test/remote-access";
import { renderApp } from "../../../test/render-app";
import { outcome, refusal } from "../../../test/roles";
import { resize } from "../../../test/viewport";

const views = () => page.getByRole("navigation", { name: "View" });
const roster = () => page.getByRole("list", { name: "Nodes list" });
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

test("below 1000px Nodes opens on the Roster, and the Diagram is there to look at, without drags", async () => {
  await resize(900, 800);
  renderApp({ path: "/nodes", handlers: threeNodes });
  await expect.element(roster()).toBeVisible();
  await expect
    .element(views().getByRole("link", { name: "List" }))
    .toHaveAttribute("aria-current", "page");
  await userEvent.click(views().getByRole("link", { name: "Diagram" }));
  await expect.element(diagram()).toHaveAttribute("data-takes-drags", "false");
});

test("Nodes shows its Roster and the Remote access link while Remote access hasn't answered", async () => {
  let answer: (() => void) | undefined;
  const answered = new Promise<void>((resolve) => {
    answer = resolve;
  });
  renderApp({
    path: "/nodes?view=roster",
    handlers: [
      ...threeNodes,
      http.get("*/api/v1/remote-access", async () => {
        await answered;
        return HttpResponse.json(nothingSetUp);
      }),
    ],
  });
  await expect.element(rowOf("home")).toBeVisible();
  await expect
    .element(page.getByRole("link", { name: "Remote access" }))
    .toHaveAttribute("href", "/nodes/remote-access");
  answer?.();
  await expect
    .element(page.getByRole("link", { name: "Remote access · Nothing set up" }))
    .toBeVisible();
});

test("each Roster row says the Node's state, when it was last seen, how it is reached, its Roles and what its probes found", async () => {
  renderApp({ path: "/nodes?view=roster", handlers: threeNodes });
  const homeRow = rowOf("home");
  await expect.poll(textOf(homeRow)).toMatch(/^homeOnline · ControllerSearchingDownloading/u);
  const seedboxRow = rowOf("seedbox");
  await expect
    .poll(textOf(seedboxRow))
    .toMatch(/Offline · Last seen .+Downloading · Finishing its work, 3 torrents \(41 GB\) stay/u);
  await expect
    .element(seedboxRow.getByRole("list", { name: "What seedbox found" }))
    .toMatchTextContent(/^Weak.+network share/u);
  await expect
    .element(rowOf("vps"))
    .toHaveTextContent("vpsOnline · home connects to vps · 24 ms round tripSearching");
  expect(
    roster()
      .getByRole("link")
      .all()
      .map((link) => link.element().getAttribute("href")),
  ).toEqual([`/nodes/${ids.home}`, `/nodes/${ids.seedbox}`, `/nodes/${ids.vps}`]);
});

test("with one Node the Roster shows it complete and offers Add a Node in its own row", async () => {
  const codes = joinCodeServer();
  renderApp({ path: "/nodes?view=roster", handlers: codes.handlers });
  await expect.element(rowOf("home")).toBeVisible();
  await userEvent.click(roster().getByRole("button", { name: /^Add a Node/u }));
  await expect.element(page.getByRole("dialog", { name: "Add a Node" })).toBeVisible();
});

test("Remove from the Roster says what the Node still holds and writes it off", async () => {
  const server = manageServer({});
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("seedbox", "Remove");
  const dialog = page.getByRole("dialog", { name: "Remove seedbox?" });
  await expect
    .poll(textOf(dialog))
    .toMatch(
      /seedbox still has 3 torrents \(41 GB\) for Downloading\. Untick its Roles and remove seedbox once that work finishes, or remove it now and give up that work\./u,
    );
  await userEvent.click(dialog.getByRole("button", { name: "Remove and give up its work" }));
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
          "vps still has 1 torrent for Downloading. Untick its Roles and remove vps once that work finishes, or remove it now and give up that work.",
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
  await expect.element(dialog.getByRole("alert")).toMatchTextContent(/^vps started holding work/u);
  await expect.poll(textOf(dialog)).toMatch(/vps still has 1 torrent for Downloading\./u);
  await userEvent.click(dialog.getByRole("button", { name: "Remove and give up its work" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.removed).toEqual([
    { nodeId: ids.vps, writeOff: "false" },
    { nodeId: ids.vps, writeOff: "true" },
  ]);
});

test("the Controller can't be removed, and Remove says why", async () => {
  const server = manageServer({});
  renderApp({ path: "/nodes?view=roster", handlers: [...threeNodes, ...server.handlers] });
  await openMenu("home", "Remove");
  const dialog = page.getByRole("dialog", { name: "home can't be removed" });
  await expect
    .poll(textOf(dialog))
    .toMatch(/home is the Controller and keeps the library, so it can't be removed\./u);
  await expect.poll(textOf(dialog)).not.toMatch(/hand/iu);
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
    .poll(textOf(page.getByRole("dialog", { name: "Give up seedbox's work" })))
    .toMatch(/seedbox has 3 torrents \(41 GB\) for Downloading\./u);
});
