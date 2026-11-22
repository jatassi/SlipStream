import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../../../test/live-server";
import { logRecord, logServer } from "../../../test/log-server";
import { homeConfig, nodeSystems, restarts, withViewerLines } from "../../../test/node-system";
import { home, homeId, homeSystem, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";
import { worker } from "../../../test/worker";
import { placeOf, timeForms } from "../../../ui/time-words";

const stampOf = (at: string) =>
  timeForms.stamp(placeOf(at, { now: new Date(at), zone: undefined }));

const logsPage = `/nodes/${homeId}/logs`;
const ten = Array.from({ length: 10 }, (_, index) => logRecord(index + 1));
const rows = () => page.getByRole("list", { name: "Log" }).getByRole("listitem");
const line = (msg: string) => page.getByRole("button", { name: new RegExp(` ${msg}$`, "u") });
const status = () => page.getByRole("status");

function openLogs(ring = ten) {
  const log = logServer(ring);
  renderApp({ path: logsPage, handlers: [log.handler, ...nodeSystems().handlers] });
  return log;
}

test("the ring's 10 records show as 10 rows with the count", async () => {
  openLogs();
  await expect.element(line("Line 10")).toBeVisible();
  expect(rows().all()).toHaveLength(10);
  await expect.element(status()).toHaveTextContent("10 lines");
  await expect
    .element(line("Line 1"))
    .toHaveTextContent(`${stampOf(ten[0]?.time ?? "")}InfonodeLine 1`);
});

test("search matches a field's value, and a search with no match offers Clear filters", async () => {
  openLogs([
    logRecord(1, { msg: "Sent", attrs: { action: "grab", work: "Dune" } }),
    logRecord(2, { msg: "Swept" }),
  ]);
  await expect.element(line("Swept")).toBeVisible();
  await userEvent.fill(page.getByRole("searchbox", { name: "Search the log" }), "GRAB");
  await expect.element(line("Swept")).not.toBeInTheDocument();
  await expect.element(line("Sent")).toBeVisible();

  await userEvent.fill(page.getByRole("searchbox", { name: "Search the log" }), "nothing");
  await expect.element(page.getByText("No lines match your filters")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear filters" }));
  await expect.element(line("Swept")).toBeVisible();
});

const seedboxUp = (since: string) =>
  ({ ...seedbox, link: { state: "up", since }, standing: "in-step" }) as const;

test("a Worker restarted while its Logs are open keeps the lines before, a divider and the lines after", async () => {
  const server = liveServer();
  const log = logServer(ten);
  const app = renderApp({
    path: `/nodes/${seedboxId}/logs`,
    handlers: [
      ...server.handlers,
      log.handler,
      nodesAre(home, seedboxUp("2026-10-12T10:00:00Z")),
      ...nodeSystems({
        systems: { [homeId]: homeSystem, [seedboxId]: homeSystem },
        configs: { [homeId]: homeConfig, [seedboxId]: homeConfig },
      }).handlers,
      restarts().handler,
    ],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.element(line("Line 10")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Restart", exact: true }));
  await userEvent.click(
    page.getByRole("dialog", { name: "Restart seedbox?" }).getByRole("button", { name: "Restart" }),
  );
  await expect.element(page.getByText("seedbox is restarting…")).toBeVisible();

  const startedAt = "2026-10-12T10:05:00Z";
  log.reset({ reason: "restarted", startedAt }, 0);
  log.send(logRecord(1, { msg: "Started again" }));
  worker.use(nodesAre(home, seedboxUp(startedAt)));
  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedboxId } });
  await expect.element(page.getByText("seedbox is restarting…")).not.toBeInTheDocument();
  await expect.element(line("Started again")).toBeVisible();
  await expect
    .element(page.getByText(`SlipStream restarted at ${stampOf(startedAt)}`))
    .toBeVisible();
  expect(rows().all()).toHaveLength(12);
  expect(log.resumes).toEqual([0]);
});

test("a row expands to every attribute, with Copy as JSON", async () => {
  const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
  openLogs([
    logRecord(1, {
      msg: "Grabbed",
      attrs: { work: "Dune", size: 4096, tags: ["a"], msg: "an attribute" },
    }),
  ]);
  await userEvent.click(line("Grabbed"));
  await expect.element(line("Grabbed")).toHaveAttribute("aria-expanded", "true");
  const attributes = page.getByRole("definition");
  expect(attributes.all().map((value) => value.element().textContent)).toEqual([
    "Dune",
    "4096",
    '["a"]',
    "an attribute",
  ]);
  expect(
    page
      .getByRole("term")
      .all()
      .map((term) => term.element().textContent),
  ).toEqual(["work", "size", "tags", "msg"]);
  await userEvent.click(page.getByRole("button", { name: "Copy as JSON" }));
  await expect.element(page.getByText("Line copied as JSON")).toBeVisible();
  expect(JSON.parse(String(writeText.mock.calls[0]?.[0]))).toMatchObject({
    msg: "Grabbed",
    level: "info",
    work: "Dune",
    size: 4096,
  });
});

test("the list keeps the newest Lines kept for the viewer", async () => {
  const log = logServer(ten);
  renderApp({
    path: logsPage,
    handlers: [log.handler, ...nodeSystems({ configs: { [homeId]: withViewerLines(5) } }).handlers],
  });
  await expect.element(line("Line 10")).toBeVisible();
  await expect.element(status()).toHaveTextContent("5 lines");
  await expect.element(line("Line 5")).not.toBeInTheDocument();
});

test("a reader that loses the stream resumes after the last record it had", async () => {
  const log = openLogs();
  await expect.element(line("Line 10")).toBeVisible();
  log.clients[0]?.error();
  await expect.poll(() => log.resumes, { timeout: 3000 }).toEqual([0, 10]);
  log.send(logRecord(11));
  await expect.element(line("Line 11")).toBeVisible();
  expect(rows().all()).toHaveLength(11);
});

test("scrolling up stops following, and Jump to latest returns to the tail", async () => {
  openLogs(Array.from({ length: 200 }, (_, index) => logRecord(index + 1)));
  await expect.element(line("Line 200")).toBeVisible();
  const scroller = page.getByRole("list", { name: "Log" }).element().parentElement;
  scroller?.scrollTo({ top: 0 });
  await expect.element(status()).toHaveTextContent("Not following,");
  await expect.element(line("Line 1")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Jump to latest" }));
  await expect.element(line("Line 200")).toBeVisible();
  await expect.element(status()).toHaveTextContent("200 lines");
});

test("Download logs from a Node that doesn't answer says so and saves nothing", async () => {
  const click = vi.spyOn(HTMLAnchorElement.prototype, "click");
  const log = logServer(ten);
  renderApp({
    path: logsPage,
    handlers: [log.handler, ...nodeSystems({ systems: { [homeId]: "offline" } }).handlers],
  });
  await userEvent.click(page.getByRole("button", { name: "Download logs" }));
  await expect
    .element(page.getByText("Couldn't download the logs. The Node is offline."))
    .toBeVisible();
  expect(click).not.toHaveBeenCalled();
});

test("on a phone the filters wait behind Filter", async () => {
  await asPhone();
  openLogs([
    logRecord(1, { level: "info", msg: "Started" }),
    logRecord(2, { level: "error", msg: "Failed" }),
  ]);
  await expect.element(line("Started")).toBeVisible();
  await expect.element(page.getByRole("radio", { name: "Error" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Filter" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "Filters" }).getByRole("radio", { name: "Error" }),
  );
  await userEvent.keyboard("{Escape}");
  await expect.element(line("Failed")).toBeVisible();
  await expect.element(line("Started")).not.toBeInTheDocument();
});
