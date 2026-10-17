import { http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../../../test/handlers";
import { nodeSystems } from "../../../test/node-system";
import { home, homeId, homeSystem, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const system = nodeSystems({ systems: { [homeId]: homeSystem, [seedboxId]: "offline" } }).handlers;
const tabsOf = (name: string) => page.getByRole("navigation", { name });
const linkTexts = (name: string) =>
  tabsOf(name)
    .getByRole("link")
    .all()
    .map((link) => [link.element().textContent, link.element().getAttribute("href")]);

test("/system and /nodes land on the Overview of the Node the browser talks to", async () => {
  const { router } = renderApp({ path: "/system", handlers: system });
  await expect.poll(() => router.state.location.href).toBe(`/nodes/${homeId}`);
  await expect.element(page.getByRole("heading", { name: "home", level: 1 })).toBeVisible();
  await expect.element(page.getByText("0.0.0-dev+3f2a1c9")).toBeVisible();
  await expect.element(page.getByText("/data/home", { exact: true })).toBeVisible();
  await expect.element(page.getByText("127.0.0.2:8080")).toBeVisible();
});

test("the page tabs link to every section and mark the current one", async () => {
  renderApp({ path: `/nodes/${homeId}/logs`, handlers: system });
  await expect
    .element(tabsOf("home").getByRole("link", { name: "Logs" }))
    .toHaveAttribute("aria-current", "page");
  expect(linkTexts("home")).toEqual([
    ["Overview", `/nodes/${homeId}`],
    ["Web and API", `/nodes/${homeId}/web`],
    ["Logs", `/nodes/${homeId}/logs`],
    ["Log settings", `/nodes/${homeId}/log-settings`],
  ]);

  await userEvent.click(tabsOf("home").getByRole("link", { name: "Overview" }));
  await expect.element(page.getByText("127.0.0.2:8080")).toBeVisible();
  await expect
    .element(tabsOf("home").getByRole("link", { name: "Overview" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(tabsOf("home").getByRole("link", { name: "Logs" }))
    .not.toHaveAttribute("aria-current");
});

test("with one Node there is no picker", async () => {
  renderApp({ path: `/nodes/${homeId}`, handlers: system });
  await expect.element(page.getByText("127.0.0.2:8080")).toBeVisible();
  await expect.element(page.getByRole("combobox", { name: "Node" })).not.toBeInTheDocument();
});

test("the Node picker lists each Node with its state and switches Node, keeping the section", async () => {
  const { router } = renderApp({
    path: `/nodes/${homeId}/logs`,
    handlers: [...system, nodesAre(home, seedbox)],
  });
  const picker = page.getByRole("combobox", { name: "Node" });
  await expect.element(picker).toHaveTextContent("onlinehome");
  await picker.click();
  await expect.element(page.getByRole("option", { name: "online home" })).toBeVisible();
  await userEvent.click(page.getByRole("option", { name: "offline seedbox" }));

  await expect.poll(() => router.state.location.href).toBe(`/nodes/${seedboxId}/logs`);
  await expect.element(page.getByRole("heading", { name: "seedbox", level: 1 })).toBeVisible();
  await expect
    .element(tabsOf("seedbox").getByRole("link", { name: "Logs" }))
    .toHaveAttribute("aria-current", "page");
});

test("an unknown Node is not found", async () => {
  renderApp({ path: "/nodes/00000000-0000-0000-0000-000000000000", handlers: system });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});

test("a Member gets the server's refusal", async () => {
  renderApp({
    path: "/nodes",
    handlers: [
      http.get("*/api/v1/nodes", () => problem(403, "needs_admin", "Only an Admin can do this")),
    ],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("This page is for Admins.");
  await expect
    .element(page.getByRole("button", { name: "Open the portal" }))
    .toHaveAttribute("href", "/portal");
});

test("on phone, Nodes lists the Node's sections, and each pushes with a back link to Nodes", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/nodes", handlers: system });
  await expect.element(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  await expect.element(page.getByText("127.0.0.2:8080")).toBeVisible();
  await expect.element(tabsOf("home")).not.toBeInTheDocument();
  const sections = page.getByRole("list", { name: "home" });
  expect(
    sections
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Web and API", "Logs", "Log settings"]);

  await userEvent.click(sections.getByRole("link", { name: "Logs" }));
  await expect.poll(() => router.state.location.href).toBe(`/nodes/${homeId}/logs`);
  const bar = page.getByRole("banner");
  await expect.element(bar.getByRole("heading", { name: "Logs" })).toBeVisible();
  const back = bar.getByRole("link", { name: "Nodes" });
  await expect.element(back).toHaveAttribute("href", `/nodes/${homeId}`);
  await userEvent.click(back);
  await expect.poll(() => router.state.location.href).toBe(`/nodes/${homeId}`);
});

test("on phone, the Node row opens a sheet of Nodes that switches Node", async () => {
  await asPhone();
  const { router } = renderApp({
    path: `/nodes/${homeId}`,
    handlers: [...system, nodesAre(home, seedbox)],
  });
  await userEvent.click(page.getByRole("button", { name: "online home Node" }));
  const sheet = page.getByRole("dialog", { name: "Node" });
  await userEvent.click(sheet.getByRole("button", { name: "offline seedbox" }));
  await expect.poll(() => router.state.location.href).toBe(`/nodes/${seedboxId}`);
  await expect
    .element(
      page.getByText("seedbox is offline. Its settings and logs come back when it reconnects."),
    )
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "offline seedbox Node" })).toBeVisible();
});
