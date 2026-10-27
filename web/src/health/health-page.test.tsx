import { HttpResponse, http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  awayNode,
  failedTask,
  fillingVolume,
  fullVolume,
  healthNow,
  healthServer,
  minutesAgo,
  reportOf,
  stoppedIndexer,
} from "../test/health";
import { larkspurId } from "../test/indexers";
import { indexersServer } from "../test/indexers-server";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

afterEach(() => {
  vi.useRealTimers();
});

const atHealthNow = () => {
  vi.useFakeTimers({ toFake: ["Date"], now: healthNow });
};

const region = (name: string) => page.getByRole("region", { name });

const literal = (text: string) => text.replaceAll(/[.()]/gu, String.raw`\$&`);

const rowOf = (subject: string) =>
  page.getByRole("button", { name: new RegExp(`^(?:Error|Warning) ${literal(subject)}`, "u") });

test("with nothing wrong the page says everything is working and names what it checked", async () => {
  renderApp({ path: "/health", handlers: healthServer(reportOf()).handlers });
  await expect.element(page.getByText("Everything is working.")).toBeVisible();
  await expect
    .element(page.getByText("Checked storage, Nodes, Indexers and scheduled tasks"))
    .toBeVisible();
  await expect.element(region("Storage").getByText("Nothing is wrong with storage.")).toBeVisible();
  await expect
    .element(region("Storage").getByRole("button", { name: "Check now: Storage" }))
    .toBeVisible();
  await expect.element(region("Nodes")).toHaveTextContent("Nothing is wrong with Nodes.Check now");
  await expect.element(region("Indexers")).toHaveTextContent("Nothing is wrong with Indexers.");
  expect(region("Indexers").getByRole("button").query()).toBeNull();
  await expect
    .element(region("Scheduled tasks"))
    .toHaveTextContent("Nothing is wrong with scheduled tasks.");
  expect(region("Scheduled tasks").getByRole("button").query()).toBeNull();
});

test("an error leads the sentence and links to its item", async () => {
  atHealthNow();
  const report = reportOf({ storage: [fillingVolume, fullVolume] });
  renderApp({ path: "/health", handlers: healthServer(report).handlers });
  const anchor = `health-storage-${fullVolume.key}`;
  await expect
    .element(page.getByRole("link", { name: fullVolume.text }))
    .toHaveAttribute("href", `#${anchor}`);
  expect(document.querySelector(`#${CSS.escape(anchor)}`)?.textContent).toContain(
    fullVolume.subject,
  );
});

test("categories with errors come first, then warnings, then the quiet ones", async () => {
  atHealthNow();
  const report = reportOf({ storage: [fillingVolume], nodes: [awayNode] });
  renderApp({ path: "/health", handlers: healthServer(report).handlers });
  await expect.element(region("Nodes").getByRole("heading")).toHaveTextContent("NodesError1");
  await expect.element(region("Storage").getByRole("heading")).toHaveTextContent("StorageWarning1");
  expect([...document.querySelectorAll("section[id]")].map((section) => section.id)).toEqual([
    "nodes",
    "storage",
    "indexers",
    "scheduled-tasks",
  ]);
});

test("each item reads its level first, then its subject, what to do and since when", async () => {
  atHealthNow();
  renderApp({
    path: "/health",
    handlers: healthServer(reportOf({ storage: [fullVolume] })).handlers,
  });
  const row = rowOf(fullVolume.subject);
  await expect
    .element(row)
    .toHaveAccessibleName(
      new RegExp(
        `^Error ${literal(fullVolume.subject)} ${literal(fullVolume.text)} since \\d\\d:\\d\\d$`,
        "u",
      ),
    );
  await userEvent.click(row);
  const facts = page.getByRole("definition");
  await expect.element(facts.first()).toHaveTextContent("18 GB");
  await expect.element(facts.nth(1)).toHaveTextContent("20 GB, the global level");
});

test("Check now on an item posts its check and the answer becomes the page", async () => {
  atHealthNow();
  const health = healthServer(reportOf({ storage: [fullVolume, fillingVolume] }));
  renderApp({ path: "/health", handlers: health.handlers });
  await userEvent.click(rowOf(fillingVolume.subject));
  health.state.report = reportOf({ storage: [fullVolume] });
  await userEvent.click(page.getByRole("button", { name: `Check now: ${fillingVolume.subject}` }));
  await expect.poll(() => health.checks).toEqual([`storage/${fillingVolume.key}`]);
  await expect.element(rowOf(fillingVolume.subject)).not.toBeInTheDocument();
  await expect.element(region("Storage").getByRole("heading")).toHaveTextContent("StorageError1");
});

test("Check now that can't reach a Node says so, and the item stays", async () => {
  atHealthNow();
  const health = healthServer(reportOf({ storage: [fullVolume] }));
  health.state.unreached = [
    { key: fullVolume.key, words: "home is offline, so its figures are as of 05:58." },
  ];
  renderApp({ path: "/health", handlers: health.handlers });
  await userEvent.click(rowOf(fullVolume.subject));
  await userEvent.click(page.getByRole("button", { name: `Check now: ${fullVolume.subject}` }));
  await expect
    .element(page.getByRole("status"))
    .toHaveTextContent("Checked. home is offline, so its figures are as of 05:58.");
  await expect.element(rowOf(fullVolume.subject)).toBeVisible();
});

test("Run now on a failed task runs it", async () => {
  atHealthNow();
  const runs: string[] = [];
  renderApp({
    path: "/health",
    handlers: [
      ...healthServer(reportOf({ tasks: [failedTask] })).handlers,
      http.post("*/api/v1/tasks/:taskId/runs", ({ params }) => {
        runs.push(String(params["taskId"]));
        return HttpResponse.json({ runId: 42 }, { status: 202 });
      }),
    ],
  });
  await userEvent.click(rowOf(failedTask.subject));
  await userEvent.click(page.getByRole("button", { name: `Run now: ${failedTask.subject}` }));
  await expect.poll(() => runs).toEqual(["housekeeping"]);
  await expect.element(page.getByRole("status")).toHaveTextContent("Started.");
});

test("an item's link leads to its page", async () => {
  atHealthNow();
  renderApp({ path: "/health", handlers: healthServer(reportOf({ nodes: [awayNode] })).handlers });
  await userEvent.click(rowOf(awayNode.subject));
  await expect
    .element(page.getByRole("button", { name: "Open seedbox" }))
    .toHaveAttribute("href", `/nodes/${awayNode.key}`);
});

test("a link with a fragment lands on that part of the page", async () => {
  atHealthNow();
  const { router } = renderApp({
    path: "/health",
    handlers: [
      ...healthServer(reportOf({ indexers: [stoppedIndexer] })).handlers,
      ...indexersServer().handlers,
    ],
  });
  await userEvent.click(rowOf(stoppedIndexer.subject));
  await userEvent.click(page.getByRole("button", { name: "Open Larkspur" }));
  await expect
    .poll(() => ({ path: router.state.location.pathname, hash: router.state.location.hash }))
    .toEqual({ path: `/settings/indexers/${larkspurId}`, hash: "allow-failover" });
});

test("Check all and a category's Check now post their checks", async () => {
  const health = healthServer(reportOf({ nodes: [awayNode] }));
  renderApp({ path: "/health", handlers: health.handlers });
  await userEvent.click(page.getByRole("button", { name: "Check all" }));
  await expect.element(page.getByRole("status")).toHaveTextContent("Checked.");
  await userEvent.click(page.getByRole("button", { name: "Check now: Storage" }));
  await expect.poll(() => health.checks).toEqual(["all", "storage"]);
});

test("rows keep the order they were first shown in while the page is open", async () => {
  atHealthNow();
  const server = liveServer();
  const health = healthServer(reportOf({ storage: [fullVolume, fillingVolume] }));
  renderApp({ path: "/health", handlers: [...server.handlers, ...health.handlers] });
  await expect.element(rowOf(fillingVolume.subject)).toBeVisible();
  const newer = { ...fillingVolume, key: "v.home.Rjo", subject: "F: (Scratch) on home" };
  health.state.report = reportOf({
    storage: [newer, { ...fillingVolume, level: "fail", since: minutesAgo(50) }, fullVolume],
  });
  server.clients[0]?.send({ event: "health.changed", data: { categories: ["storage"] } });
  await expect.element(rowOf(newer.subject)).toBeVisible();
  expect(
    region("Storage")
      .getByRole("button", { name: / on home/u })
      .all()
      .map((row) => row.element().textContent),
  ).toEqual([
    expect.stringContaining(fullVolume.subject),
    expect.stringContaining(fillingVolume.subject),
    expect.stringContaining(newer.subject),
  ]);
});

test("on a phone an item opens a sheet with its action", async () => {
  atHealthNow();
  await asPhone();
  renderApp({
    path: "/health",
    handlers: healthServer(reportOf({ storage: [fullVolume] })).handlers,
  });
  await userEvent.click(rowOf(fullVolume.subject));
  const sheet = page.getByRole("dialog", { name: fullVolume.subject });
  await expect
    .element(sheet.getByRole("button", { name: `Check now: ${fullVolume.subject}` }))
    .toBeVisible();
  await expect.element(sheet.getByText("18 GB")).toBeVisible();
});
