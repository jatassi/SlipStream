import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import {
  copying,
  failed,
  needsSpace,
  queued,
  speedFromSeedbox,
  transfersApi,
  waiting,
} from "../../../test/transfers";
import { asPhone } from "../../../test/viewport";

const path = `/nodes/${homeId}/transfers`;
const mib = 1024 * 1024;
const twoNodes = nodesAre(home, seedbox);
const everyState = [copying, queued, waiting, needsSpace, failed];

const rowOf = (jobName: string) =>
  page.getByRole("list", { name: "Transfers of home" }).getByRole("listitem").filter({
    hasText: jobName,
  });

test("the section opens with what the Node receives, then each Transfer's own sentence", async () => {
  const api = transfersApi({ transfers: everyState, speeds: [speedFromSeedbox] });
  renderApp({ path, handlers: [...api.handlers, twoNodes] });
  await expect
    .element(page.getByText("Receiving Heat (1995) from seedbox at 48 MB/s"))
    .toBeVisible();
  await expect.element(page.getByText("3 waiting")).toBeVisible();
  await expect
    .element(page.getByRole("link", { name: "1 failure" }))
    .toHaveAttribute("href", "#transfers");

  await expect.element(rowOf("Heat (1995)").getByText("Transferring from seedbox")).toBeVisible();
  await expect
    .element(rowOf("Heat (1995)").getByText("2.1 GB of 8.4 GB · 48 MB/s · 2 min left"))
    .toBeVisible();
  await expect
    .element(rowOf("Heat (1995)").getByRole("progressbar", { name: "Transferring Heat (1995)" }))
    .toHaveAttribute("aria-valuenow", "25");
  for (const transfer of [queued, waiting, needsSpace, failed]) {
    await expect
      .element(rowOf(transfer.jobName).getByText(transfer.reason ?? "", { exact: true }))
      .toBeVisible();
    await expect.element(rowOf(transfer.jobName).getByText("From seedbox")).toBeVisible();
  }
  await expect.element(rowOf("Miami Vice (2006)").getByText("Failed")).toBeVisible();
  await expect.element(rowOf("Manhunter (1986)").getByText("Needs space")).toBeVisible();

  const speeds = page.getByRole("list", { name: "Recent speeds of home" });
  await expect.element(speeds.getByText("From seedbox")).toBeVisible();
  await expect.element(speeds.getByText("48 MB/s")).toBeVisible();
});

test("the sending Node reads the same Transfer from its own side", async () => {
  const api = transfersApi({ transfers: [copying], speeds: [speedFromSeedbox] });
  renderApp({ path: `/nodes/${seedboxId}/transfers`, handlers: [...api.handlers, twoNodes] });
  await expect.element(page.getByText("Sending Heat (1995) to home at 48 MB/s")).toBeVisible();
  await expect.element(page.getByText("Transferring to home")).toBeVisible();
  await expect
    .element(page.getByRole("list", { name: "Recent speeds of seedbox" }).getByText("To home"))
    .toBeVisible();
});

test("lanes, round trip, route and retried chunks stay under Details until opened", async () => {
  const api = transfersApi({ transfers: [copying] });
  renderApp({ path, handlers: [...api.handlers, twoNodes] });
  const row = rowOf("Heat (1995)");
  await expect.element(row.getByText("Transferring from seedbox")).toBeVisible();
  await expect.element(row.getByText("home dials seedbox")).not.toBeInTheDocument();
  await expect.element(row.getByText("42 ms")).not.toBeInTheDocument();

  await userEvent.click(row.getByRole("button", { name: "Details" }));
  await expect.element(row.getByText("home dials seedbox")).toBeVisible();
  await expect.element(row.getByText("42 ms")).toBeVisible();
  await expect.element(row.getByText("1 of 2 verified")).toBeVisible();
});

test("with nothing to send or receive, the section says so", async () => {
  const api = transfersApi();
  renderApp({ path, handlers: [...api.handlers, twoNodes] });
  await expect.element(page.getByText("Nothing moving")).toBeVisible();
  await expect
    .element(page.getByText("No Transfer has moved bytes between home and another Node yet."))
    .toBeVisible();
  expect(page.getByRole("list", { name: "Transfers of home" }).query()).toBeNull();
});

test("a send limit saves for the Node and reads back as saved", async () => {
  const api = transfersApi();
  renderApp({ path, handlers: [...api.handlers, twoNodes] });
  const limit = page.getByRole("textbox", { name: "Send limit" });
  await expect.element(limit).toHaveValue("0");
  await userEvent.fill(limit, "10");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Transfer settings saved")).toBeVisible();
  expect(api.writes).toEqual([{ nodeId: homeId, body: { sendLimit: 10 * mib } }]);
  await expect.element(page.getByRole("button", { name: "Reset to default" })).toBeVisible();

  renderApp({ path, handlers: [...api.handlers, twoNodes] });
  await expect.element(page.getByRole("textbox", { name: "Send limit" }).last()).toHaveValue("10");
});

test("a single Node has no Transfers tab, and its address is not found", async () => {
  const api = transfersApi();
  renderApp({ path: `/nodes/${homeId}/web`, handlers: [...api.handlers, nodesAre(home)] });
  const tabs = page.getByRole("navigation", { name: "home" });
  await expect.element(tabs.getByRole("link", { name: "Web and API" })).toBeVisible();
  expect(tabs.getByRole("link", { name: "Transfers" }).query()).toBeNull();

  renderApp({ path, handlers: [...api.handlers, nodesAre(home)] });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});

test("on a phone the Node page lists Transfers as a pushed section", async () => {
  await asPhone();
  const api = transfersApi({ transfers: [copying] });
  const { router } = renderApp({ path: `/nodes/${homeId}`, handlers: [...api.handlers, twoNodes] });
  await userEvent.click(
    page.getByRole("list", { name: "home" }).getByRole("link", { name: "Transfers" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe(path);
  await expect.element(page.getByText("Transferring from seedbox")).toBeVisible();
});
