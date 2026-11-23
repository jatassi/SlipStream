import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SupersededView } from "../api/gen/model";
import { getGetSupersededMockHandler } from "../api/gen/query/superseded/superseded.msw";
import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { noPasskeys, problem, signedInAsGrace, signedOut, startedAt } from "../test/handlers";
import { liveServer, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { worker } from "../test/worker";
// oxlint-disable-next-line import/max-dependencies -- the page reads status, the Superseded view, the live socket and the clock
import { whenWords } from "../ui/time-words";

const fenced = getGetStatusMockHandler({
  claim: "controller",
  passkeys: noPasskeys,
  startedAt,
  superseded: true,
});

const fencedAt = new Date(Date.now() - 3 * 60_000).toISOString();

const proven: SupersededView = {
  state: "proven",
  at: fencedAt,
  replacement: { key: "7Q2M-0K4D-91XB-3FHA", address: "192.168.1.20" },
  worker: "seedbox",
};

const title = "This Controller was replaced";

const fencedWords = () => {
  const words = whenWords(fencedAt, "at", { now: new Date(), zone: undefined });
  return `${words.charAt(0).toUpperCase()}${words.slice(1)}`;
};

function wipes() {
  const sent: unknown[] = [];
  const handler = http.post("*/api/v1/superseded/wipe", async ({ request }) => {
    sent.push(await request.json());
    return new HttpResponse(null, { status: 202 });
  });
  return { sent, handler };
}

test.each(["/", "/sign-in"])(
  "a signed-in Admin's visit to %s on a Superseded Controller lands on the Superseded page",
  async (path) => {
    const app = renderApp({ path, handlers: [fenced, getGetSupersededMockHandler(proven)] });
    await expect.element(page.getByRole("heading", { name: title })).toBeVisible();
    expect(app.router.state.location.pathname).toBe("/superseded");
    await expect
      .element(
        page.getByText(
          `${fencedWords()}, a Backup of this library was restored on the Controller at 192.168.1.20. This Controller's torrents are paused and its searches have stopped.`,
        ),
      )
      .toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Wipe and join as a Worker" }))
      .toBeVisible();
    await expect
      .element(page.getByRole("button", { name: "Resume this Controller…" }))
      .not.toBeInTheDocument();
  },
);

test("Wipe and join sends the Join Code once confirmed, then waits for the restart", async () => {
  const { sent, handler } = wipes();
  renderApp({
    path: "/superseded",
    handlers: [fenced, getGetSupersededMockHandler(proven), handler],
  });
  await userEvent.fill(page.getByLabelText("Join Code"), " SLJ1-ABCD ");
  await userEvent.click(page.getByRole("button", { name: "Wipe and join as a Worker" }));
  await expect.element(page.getByRole("dialog", { name: "Wipe this Controller?" })).toBeVisible();
  expect(sent).toEqual([]);
  await userEvent.click(
    page.getByRole("dialog").getByRole("button", { name: "Wipe and join as a Worker" }),
  );
  await expect
    .element(
      page.getByText(
        "Wiping this Controller. It restarts, then joins the new Controller as a Worker.",
      ),
    )
    .toBeVisible();
  expect(sent).toEqual([{ joinCode: "SLJ1-ABCD" }]);
});

test("a damaged Join Code shows on the field and nothing is wiped", async () => {
  renderApp({
    path: "/superseded",
    handlers: [
      fenced,
      getGetSupersededMockHandler(proven),
      http.post("*/api/v1/superseded/wipe", () =>
        problem(
          422,
          "join_code_damaged",
          "The Join Code is mistyped, cut short or not a SlipStream code",
        ),
      ),
    ],
  });
  await userEvent.fill(page.getByLabelText("Join Code"), "SLJ1-AB");
  await userEvent.click(page.getByRole("button", { name: "Wipe and join as a Worker" }));
  await userEvent.click(
    page.getByRole("dialog").getByRole("button", { name: "Wipe and join as a Worker" }),
  );
  await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  await expect
    .element(page.getByText("The Join Code is mistyped, cut short or not a SlipStream code"))
    .toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Wipe and join as a Worker" }))
    .toBeVisible();
});

test("Details names the new Controller's key and the Worker that said so, and Resume goes back to the app", async () => {
  let resumed = false;
  const server = liveServer();
  const app = renderApp({
    path: "/superseded",
    handlers: [
      ...server.handlers,
      fenced,
      getGetSupersededMockHandler(proven),
      http.post("*/api/v1/superseded/resume", () => {
        resumed = true;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("7Q2M-0K4D-91XB-3FHA")).toBeVisible();
  await expect.element(page.getByText("seedbox")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Resume this Controller…" }));
  await userEvent.click(page.getByRole("button", { name: "Resume", exact: true }));
  await expect.poll(() => resumed).toBe(true);
  worker.use(getGetStatusMockHandler({ claim: "controller", passkeys: noPasskeys, startedAt }));
  server.clients.at(-1)?.send({ event: "nodes.fenced", data: { superseded: false } });
  await expect.poll(() => app.router.state.location.pathname).not.toBe("/superseded");
  await expect.element(page.getByRole("heading", { name: title })).not.toBeInTheDocument();
});

test.each(["retired", "stopped"] as const)("a Controller says an Admin %s it", async (state) => {
  renderApp({
    path: "/superseded",
    handlers: [fenced, getGetSupersededMockHandler({ state, at: fencedAt })],
  });
  await expect
    .element(
      page.getByText(
        `${fencedWords()}, an Admin ${state} this Controller. Its torrents are paused and its searches have stopped.`,
      ),
    )
    .toBeVisible();
});

test("signed out, the page says only that it was replaced and offers Sign in", async () => {
  const app = renderApp({ path: "/nodes", handlers: [fenced, signedOut] });
  await expect.element(page.getByRole("heading", { name: title })).toBeVisible();
  expect(app.router.state.location.pathname).toBe("/superseded");
  await expect
    .element(page.getByText("Sign in as an Admin to wipe this Controller and join the new one."))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Sign in" }));
  await expect.element(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
});

test("a Member sees that an Admin can act", async () => {
  renderApp({ path: "/portal", handlers: [fenced, signedInAsGrace] });
  await expect
    .element(page.getByText("An Admin can wipe this Controller and join the new one."))
    .toBeVisible();
});

test("a tab open when the Controller is fenced moves to the Superseded page", async () => {
  const server = liveServer();
  const app = renderApp({ path: "/library", handlers: server.handlers });
  await synced(app);
  worker.use(
    fenced,
    getGetSupersededMockHandler(proven),
    http.all("*/api/v1/*", () => problem(409, "superseded", "This Controller was replaced")),
  );
  server.clients[0]?.send({ event: "nodes.fenced", data: { superseded: true } });
  await expect.poll(() => app.router.state.location.pathname).toBe("/superseded");
  await expect.element(page.getByRole("heading", { name: title })).toBeVisible();
});
