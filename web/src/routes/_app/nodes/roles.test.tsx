import type { RequestHandler } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { NodesBoardNode, NodesNode } from "../../../api/gen/model";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import {
  board,
  boardIs,
  homeSeat,
  outcome,
  refusal,
  rolesServer,
  seedboxSeat,
  writeOffServer,
} from "../../../test/roles";
import { asPhone } from "../../../test/viewport";

const onlineSeedbox: NodesNode = {
  ...seedbox,
  standing: "in-step",
  link: { state: "up", since: "2026-10-12T10:00:00Z" },
};

const homeNamed = { id: homeId, name: "home" };
const seedboxNamed = { id: seedboxId, name: "seedbox" };
const torrents = { units: 3, noun: "torrents", bytes: 41 * 1024 ** 3, endless: true };

const drained: NodesBoardNode = {
  ...seedboxSeat,
  assigned: [],
  roles: [{ role: "downloading", state: "draining", held: torrents }],
};

const dialog = () => page.getByRole("dialog", { name: "Review the change" });
const role = (name: string) => page.getByRole("checkbox", { name, exact: true });
const titled = (name: string) => page.getByRole("button", { name, exact: true });

function rolesOf(seat: NodesBoardNode, { handlers = [] }: { handlers?: RequestHandler[] } = {}) {
  const others = seat.id === homeId ? [seedboxSeat] : [homeSeat];
  return renderApp({
    path: `/nodes/${seat.id}/roles`,
    handlers: [nodesAre(home, onlineSeedbox), boardIs(board(...others, seat)), ...handlers],
  });
}

test("the Roles section shows each Role Group's switch and every Role's state", async () => {
  rolesOf(drained);
  await Promise.all(
    ["Acquire", "Store", "Process", "Connect", "Reach"].map((group) =>
      expect.element(page.getByRole("switch", { name: group })).toBeVisible(),
    ),
  );
  await expect.element(page.getByText("0 of 2").first()).toBeVisible();
  await expect.element(titled("Acquire")).toHaveAttribute("aria-expanded", "true");
  await expect.element(role("Downloading")).not.toBeChecked();
  await expect.element(page.getByText("Draining, 3 torrents (41 GB) stay")).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Write off what seedbox holds" }))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Review" })).toBeDisabled();
});

test("a Role Group opens to its Roles only when one is ticked, hinted, draining or just switched", async () => {
  rolesOf(seedboxSeat);
  await expect.element(titled("Acquire")).toHaveAttribute("aria-expanded", "true");
  await expect.element(titled("Process")).toHaveAttribute("aria-expanded", "false");
  await expect.element(role("Subtitles")).not.toBeInTheDocument();
  await expect.element(page.getByText("0 of 2").first()).toBeVisible();

  await userEvent.click(page.getByRole("switch", { name: "Process" }));
  await expect.element(role("Subtitles")).toBeChecked();

  await userEvent.click(titled("Connect"));
  await expect.element(role("Notifying")).not.toBeChecked();
});

test("the Controller keeps Serving ticked and says why", async () => {
  rolesOf(homeSeat);
  await expect.element(role("Serving")).toBeChecked();
  await expect.element(role("Serving")).toBeDisabled();
  await expect.element(page.getByText("The Controller always holds Serving.")).toBeVisible();
});

test("a group switch ticks its Roles together, and Apply sends the Reviewed change", async () => {
  const roles = rolesServer({
    plans: [
      outcome({
        effects: [
          {
            kind: "gains",
            node: seedboxNamed,
            role: "searching",
            words: "seedbox starts Searching.",
          },
        ],
      }),
    ],
    sets: [outcome({ basis: "basis-2" })],
  });
  rolesOf(seedboxSeat, { handlers: roles.handlers });
  await expect.element(role("Searching")).not.toBeChecked();
  await userEvent.click(page.getByRole("switch", { name: "Acquire" }));
  await expect.element(role("Searching")).toBeChecked();
  await expect.element(page.getByText("2 of 2")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Review" }));
  await expect.element(dialog().getByText("seedbox starts Searching.")).toBeVisible();
  expect(roles.planned).toEqual([
    {
      basis: "basis-1",
      set: [{ nodeId: seedboxId, roles: ["downloading", "searching"] }],
      test: true,
    },
  ]);

  await userEvent.click(dialog().getByRole("button", { name: "Apply" }));
  await expect.element(dialog()).not.toBeInTheDocument();
  expect(roles.applied).toEqual([
    { basis: "basis-1", set: [{ nodeId: seedboxId, roles: ["downloading", "searching"] }] },
  ]);
});

const unticking = outcome({
  effects: [
    {
      kind: "drains",
      node: seedboxNamed,
      role: "downloading",
      held: torrents,
      words:
        "seedbox takes no new Downloading work. It keeps its 3 torrents (41 GB) until you move or write them off.",
    },
    {
      kind: "handed-back",
      role: "downloading",
      from: [seedboxNamed],
      to: homeNamed,
      words: "home takes over Downloading, since SlipStream always needs a Node that downloads.",
    },
    {
      kind: "moves",
      role: "searching",
      from: seedboxNamed,
      to: homeNamed,
      held: { units: 3, noun: "Indexers", bytes: 0, endless: false },
      words: "3 Indexers move to home.",
    },
    {
      kind: "pinned",
      node: homeNamed,
      role: "serving",
      words: "home keeps Serving, because the Controller always serves SlipStream.",
    },
  ],
  checks: [
    {
      node: seedboxNamed,
      role: "relaying",
      verdict: "block",
      reason: "role_taken",
      words: "relay-fra already relays. Drag Relaying to move it.",
    },
    {
      node: seedboxNamed,
      role: "storage",
      verdict: "warn",
      reason: "offline_unchecked",
      words: "Can't check Storage while seedbox is offline.",
    },
  ],
});

test("the Review says what unticking does, and a check that blocks keeps Apply off", async () => {
  const roles = rolesServer({ plans: [unticking], sets: [] });
  rolesOf(seedboxSeat, { handlers: roles.handlers });
  await userEvent.click(role("Downloading"));
  await userEvent.click(page.getByRole("button", { name: "Review" }));
  const changes = dialog().getByRole("list", { name: "What changes" });
  await expect.element(changes.getByText(/^seedbox takes no new Downloading work/u)).toBeVisible();
  await expect.element(changes.getByText(/^home takes over Downloading/u)).toBeVisible();
  await expect.element(changes.getByText("3 Indexers move to home.")).toBeVisible();
  await expect.element(changes.getByText(/^home keeps Serving/u)).toBeVisible();
  const checks = dialog().getByRole("list", { name: "Checks" });
  await expect.element(checks.getByText("Blocked")).toBeVisible();
  await expect.element(checks.getByText(/^relay-fra already relays/u)).toBeVisible();
  await expect.element(checks.getByText("Warning")).toBeVisible();
  await expect.element(dialog().getByRole("button", { name: "Apply" })).toBeDisabled();
});

test("a change planned on an old basis comes back as a fresh Review, and a failed reach test blocks", async () => {
  const fresh = outcome({
    basis: "basis-2",
    effects: [
      { kind: "gains", node: seedboxNamed, role: "notifying", words: "seedbox starts Notifying." },
      { kind: "lets-go", node: homeNamed, role: "notifying", words: "home stops Notifying." },
    ],
  });
  const unreachable = outcome({
    basis: "basis-2",
    checks: [
      {
        node: seedboxNamed,
        role: "notifying",
        verdict: "block",
        reason: "reach_failed",
        words: "seedbox can't reach smtp.example.test:587 (connection refused).",
      },
    ],
  });
  const roles = rolesServer({
    plans: [outcome({})],
    sets: [
      () => refusal("roles_stale", "The Roles changed since this was planned", fresh),
      () => refusal("roles_blocked", "A guardrail blocks this change", unreachable),
    ],
  });
  rolesOf(seedboxSeat, { handlers: roles.handlers });
  await userEvent.click(titled("Connect"));
  await userEvent.click(role("Notifying"));
  await userEvent.click(page.getByRole("button", { name: "Review" }));
  await userEvent.click(dialog().getByRole("button", { name: "Apply" }));
  await expect
    .element(
      dialog().getByText("Roles changed since you opened this. Here is the change as it is now."),
    )
    .toBeVisible();
  await expect.element(dialog().getByText("home stops Notifying.")).toBeVisible();

  await userEvent.click(dialog().getByRole("button", { name: "Apply" }));
  await expect
    .element(dialog().getByText("seedbox can't reach smtp.example.test:587 (connection refused)."))
    .toBeVisible();
  await expect.element(dialog().getByRole("button", { name: "Apply" })).toBeDisabled();
  expect(roles.applied.map(({ basis }) => basis)).toEqual(["basis-1", "basis-2"]);
});

test("Write off shows the count and size first, then gives up what the Node holds", async () => {
  const writing = writeOffServer(outcome({}));
  rolesOf(seedboxSeat, { handlers: [writing.handler] });
  await userEvent.click(page.getByRole("button", { name: "Write off what seedbox holds" }));
  const confirm = page.getByRole("dialog", { name: "Write off what seedbox holds" });
  await expect
    .element(confirm.getByText("seedbox holds 3 torrents (41 GB) for Downloading."))
    .toBeVisible();
  await expect
    .element(confirm.getByText("This also unticks Downloading on seedbox."))
    .toBeVisible();
  await userEvent.click(confirm.getByRole("button", { name: "Write off" }));
  await expect.element(confirm).not.toBeInTheDocument();
  expect(writing.sent).toEqual([{ nodeId: seedboxId, body: { roles: ["downloading"] } }]);
});

test("a write-off a guardrail refuses says why", async () => {
  const writing = writeOffServer(() =>
    refusal(
      "keeps_role",
      "The Node is the last that can hold a Role it would write off",
      outcome({
        checks: [
          {
            node: seedboxNamed,
            role: "downloading",
            verdict: "block",
            reason: "keeps_role",
            words:
              "seedbox is the last Node that downloads, so it keeps Downloading and can't write off its 3 torrents (41 GB). Tick Downloading on another Node first.",
          },
        ],
      }),
    ),
  );
  rolesOf(seedboxSeat, { handlers: [writing.handler] });
  await userEvent.click(page.getByRole("button", { name: "Write off what seedbox holds" }));
  const confirm = page.getByRole("dialog", { name: "Write off what seedbox holds" });
  await userEvent.click(confirm.getByRole("button", { name: "Write off" }));
  await expect
    .element(confirm.getByRole("alert"))
    .toHaveTextContent(
      "seedbox is the last Node that downloads, so it keeps Downloading and can't write off its 3 torrents (41 GB). Tick Downloading on another Node first.",
    );
});

test("on phone the Review is a sheet", async () => {
  await asPhone();
  const roles = rolesServer({ plans: [outcome({})], sets: [] });
  rolesOf(seedboxSeat, { handlers: roles.handlers });
  await userEvent.click(role("Searching"));
  await userEvent.click(page.getByRole("button", { name: "Review" }));
  await expect.element(page.getByRole("dialog").getByText("Nothing changes.")).toBeVisible();
});
