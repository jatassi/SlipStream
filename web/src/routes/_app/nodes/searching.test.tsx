import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IndexersChallengeSolver, NodesNode } from "../../../api/gen/model";
import { SetChallengeSolverBody } from "../../../api/gen/zod/indexers/indexers.zod";
import { prowlarr, searchingSeedbox } from "../../../test/indexers";
import { indexersServer } from "../../../test/indexers-server";
import { home, homeId, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";

const at = "http://localhost:8191";
const unreachable = "http://127.0.0.1:9999";

const cantReach = (address: string) =>
  HttpResponse.json(
    {
      status: 422,
      code: "challenge_solver_unreachable",
      title: "The Node can't reach the Challenge solver",
      detail: `home can't reach a Challenge solver at ${address}. Check the address and that the solver is running beside home.`,
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

/** Each Node's solver, kept across PUTs, which refuse the unreachable address. */
function solvers(start: Record<string, IndexersChallengeSolver>) {
  const state = { ...start };
  const puts: { id: string; body: unknown }[] = [];
  const handlers = [
    http.get("*/api/v1/nodes/:id/challenge-solver", ({ params }) =>
      HttpResponse.json(state[String(params["id"])] ?? { address: null, timeout: 60 }),
    ),
    http.put("*/api/v1/nodes/:id/challenge-solver", async ({ params, request }) => {
      const id = String(params["id"]);
      const body = SetChallengeSolverBody.parse(await request.json());
      puts.push({ id, body });
      if (body.address === unreachable) {
        return cantReach(body.address);
      }
      state[id] = {
        address: body.address === "" ? null : body.address,
        timeout: body.timeout ?? 60,
      };
      return HttpResponse.json(state[id]);
    }),
  ];
  return { handlers, puts };
}

function serve({
  nodes = [home, searchingSeedbox],
  start = {},
  indexers,
}: {
  nodes?: NodesNode[];
  start?: Record<string, IndexersChallengeSolver>;
  indexers?: Parameters<typeof indexersServer>[0];
} = {}) {
  const solver = solvers(start);
  return {
    ...solver,
    handlers: [...indexersServer({ ...indexers, nodes }).handlers, ...solver.handlers],
  };
}

const address = () => page.getByRole("textbox", { name: "Address" });
const timeout = () => page.getByRole("textbox", { name: "Timeout" });
const save = () => page.getByRole("button", { name: "Save" });
const homed = () => page.getByRole("list", { name: "Indexers" });

test("a Node with no solver shows the default timeout and the Indexers homed on it", async () => {
  renderApp({ path: `/nodes/${homeId}/searching`, handlers: serve().handlers });
  await expect.element(page.getByRole("heading", { name: "Challenge solver" })).toBeVisible();
  await expect.element(address()).toHaveValue("");
  await expect.element(timeout()).toHaveValue("60");
  await expect.element(page.getByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  await expect.element(homed().getByRole("link", { name: /Prowlarr/u })).toBeVisible();
  await expect.element(homed().getByRole("link", { name: /Larkspur/u })).toBeVisible();
  await expect
    .element(homed().getByRole("link", { name: /TorrentLeech/u }))
    .not.toBeInTheDocument();
  await userEvent.click(save());
  await expect.element(page.getByText("Enter the solver's address")).toBeVisible();
});

test("a solver saves only once the Node reaches it", async () => {
  const { handlers, puts } = serve();
  renderApp({ path: `/nodes/${homeId}/searching`, handlers });
  await userEvent.fill(address(), unreachable);
  await userEvent.click(save());
  await expect
    .element(
      page.getByText(
        `home can't reach a Challenge solver at ${unreachable}. Check the address and that the solver is running beside home.`,
      ),
    )
    .toBeVisible();
  await expect.element(address()).toHaveAttribute("aria-invalid", "true");

  await userEvent.fill(address(), at);
  await userEvent.fill(timeout(), "90");
  await userEvent.click(save());
  await expect.element(page.getByText("Saved home's Challenge solver")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Remove" })).toBeVisible();
  expect(puts).toEqual([
    { id: homeId, body: { address: unreachable, timeout: 60 } },
    { id: homeId, body: { address: at, timeout: 90 } },
  ]);
});

test("Remove clears the Node's solver", async () => {
  const { handlers, puts } = serve({ start: { [seedboxId]: { address: at, timeout: 45 } } });
  renderApp({ path: `/nodes/${seedboxId}/searching`, handlers });
  await expect.element(address()).toHaveValue(at);
  await expect.element(timeout()).toHaveValue("45");
  await expect.element(homed().getByRole("link", { name: /TorrentLeech/u })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Remove" }));
  await expect.element(page.getByText("Removed seedbox's Challenge solver")).toBeVisible();
  await expect.element(address()).toHaveValue("");
  expect(puts).toEqual([{ id: seedboxId, body: { address: "" } }]);
});

test("an offline Node's solver reads but can't change", async () => {
  const away: NodesNode = { ...seedbox, roles: [{ role: "searching", state: "away" }] };
  const { handlers } = serve({
    nodes: [home, away],
    start: { [seedboxId]: { address: at, timeout: 60 } },
    indexers: { indexers: [prowlarr] },
  });
  renderApp({ path: `/nodes/${seedboxId}/searching`, handlers });
  await expect
    .element(
      page.getByText(
        "seedbox is offline. Saving tests the solver from seedbox, so its Challenge solver can change once it's back.",
      ),
    )
    .toBeVisible();
  await expect.element(address()).toHaveAttribute("readonly");
  await expect.element(save()).not.toBeInTheDocument();
  await expect.element(page.getByText("No Indexer has seedbox as its home Node.")).toBeVisible();
});
