import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { HealthItem } from "../api/gen/model";
import { healthServer, minutesAgo, reportOf } from "../test/health";
import { renderApp } from "../test/render-app";
import { seedpoolId, trackersServer } from "../test/trackers";
import { asPhone } from "../test/viewport";

const trackerPage = `/settings/trackers/${seedpoolId}`;
const tabs = () => page.getByRole("navigation", { name: "seedpool" });
const goalForm = () => page.getByRole("form", { name: "Seeding goal" });
const ruleSection = () => page.getByRole("region", { name: "Rule" });
const ruleForm = () => page.getByRole("form", { name: "Seeding rule" });

test("Settings > Trackers lists each Tracker's privacy, rule, goal and torrents, and links to its page", async () => {
  const { router } = renderApp({ path: "/settings/trackers", handlers: trackersServer().handlers });
  const list = page.getByRole("list", { name: "Trackers" });
  const row = list.getByRole("link", { name: /^seedpool/u });
  await expect.element(row).toMatchTextContent("Private · Rule 10 days · Default goal");
  await expect.element(row).toMatchTextContent("3 torrents· 2 owe seeding");
  await expect
    .element(row.getByText("1 torrent owes seeding and isn't seeding"))
    .toBeInTheDocument();
  await expect
    .element(list.getByRole("link", { name: /^TorrentLeech/u }))
    .toMatchTextContent("Private · Rule ratio 1.0 or 10 days · Default goal");
  await expect
    .element(list.getByRole("link", { name: /^Aither/u }))
    .toMatchTextContent("Private · No rule known · Default goal");
  await userEvent.click(row);
  await expect.element(page.getByRole("heading", { name: "seedpool", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe(trackerPage);
});

test("the default goal saves whole, and Forever sends no ratio, time or action", async () => {
  const server = trackersServer();
  renderApp({ path: "/settings/trackers", handlers: server.handlers });
  await expect
    .element(
      goalForm().getByText(/^Each torrent seeds until it reaches a ratio of 2.0 or 14 days/u),
    )
    .toBeVisible();
  await userEvent.click(goalForm().getByRole("radio", { name: "Both" }));
  await userEvent.fill(goalForm().getByRole("textbox", { name: "Ratio" }), "1.5");
  await userEvent.fill(goalForm().getByRole("textbox", { name: "Seeding time" }), "36");
  await userEvent.click(goalForm().getByRole("radio", { name: "hours" }));
  await userEvent.click(goalForm().getByRole("combobox", { name: "When the goal is met" }));
  await userEvent.click(page.getByRole("option", { name: "Pause" }));
  await userEvent.click(goalForm().getByRole("button", { name: "Save goal" }));
  await expect
    .element(
      goalForm().getByText(
        "Each torrent seeds until it reaches a ratio of 1.5 and 36 h of seeding. Then SlipStream pauses it.",
      ),
    )
    .toBeVisible();
  await userEvent.click(goalForm().getByRole("radio", { name: "Forever" }));
  await expect.element(goalForm().getByRole("textbox", { name: "Ratio" })).not.toBeInTheDocument();
  await userEvent.click(goalForm().getByRole("button", { name: "Save goal" }));
  await expect
    .element(goalForm().getByText("Each torrent seeds with no end until you remove it."))
    .toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "PUT",
      id: "default-goal",
      body: { combine: "both", ratio: 1.5, seedSeconds: 129_600, action: "pause" },
    },
    {
      method: "PUT",
      id: "default-goal",
      body: { combine: "forever", ratio: null, seedSeconds: null, action: null },
    },
  ]);
});

test("a refused value shows on the field seeding_value_refused names", async () => {
  const detail = "Use a ratio of 0.01 or more with at most two decimals.";
  const server = trackersServer({ refuse: { field: "ratio", detail } });
  renderApp({ path: "/settings/trackers", handlers: server.handlers });
  await userEvent.fill(goalForm().getByRole("textbox", { name: "Ratio" }), "0.001");
  await userEvent.click(goalForm().getByRole("button", { name: "Save goal" }));
  await expect
    .element(goalForm().getByRole("textbox", { name: "Ratio" }))
    .toHaveAttribute("aria-invalid", "true");
  await expect.element(goalForm().getByText(detail)).toBeVisible();
});

test("the seeding settings save by key", async () => {
  const server = trackersServer();
  renderApp({ path: "/settings/trackers", handlers: server.handlers });
  const margin = page.getByRole("textbox", { name: "Margin past a rule" });
  await userEvent.fill(margin, "3");
  await userEvent.click(page.getByRole("button", { name: "Save" }).first());
  await expect
    .poll(() => server.writes)
    .toEqual([
      { method: "PATCH", id: "settings", body: { values: { "trackers.margin": 10_800 } } },
    ]);
});

test("the Tracker page leads with its rule and standing, and its Torrents tab holds only its torrents", async () => {
  const { router } = renderApp({ path: trackerPage, handlers: trackersServer().handlers });
  await expect
    .element(
      page.getByText(
        /^seedpool requires 10 days of seeding for each torrent\. Ratio doesn't count\./u,
      ),
    )
    .toBeVisible();
  await expect.element(page.getByText("From seedpool's Releases.")).toBeVisible();
  await expect
    .element(
      page.getByText("2 torrents are seeding. 2 still owe seedpool, and 1 of those isn't seeding."),
    )
    .toBeVisible();
  await expect.element(page.getByRole("definition").filter({ hasText: "1.50" })).toBeVisible();
  await expect.element(page.getByText("Private", { exact: true })).toBeVisible();
  expect(
    tabs()
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Torrents3 needs attention", "Seeding2 needs attention", "Routing", "Indexers1"]);
  await expect
    .element(tabs().getByRole("link", { name: /^Torrents/u }))
    .toHaveAttribute("aria-current", "page");
  await userEvent.click(page.getByRole("radio", { name: /^All/u }));
  await expect.element(page.getByText("Hold.Fast.2024.1080p.WEB-DL")).toBeVisible();
  await expect.element(page.getByText("Salt.Line.2022.1080p.WEB-DL")).toBeVisible();
  expect(page.getByText("Elsewhere.2021.1080p.WEB-DL").query()).toBeNull();
  await userEvent.click(page.getByText("Hold.Fast.2024.1080p.WEB-DL"));
  await expect
    .poll(() => router.state.location.pathname)
    .toMatch(new RegExp(`^${trackerPage}/torrents/`, "u"));
  await expect
    .element(tabs().getByRole("link", { name: /^Torrents/u }))
    .toHaveAttribute("aria-current", "page");
});

test("the Seeding tab saves the rule and the goal whole, and each can go back", async () => {
  const server = trackersServer();
  renderApp({ path: `${trackerPage}/seeding`, handlers: server.handlers });
  await userEvent.click(ruleForm().getByRole("radio", { name: "Either one" }));
  await userEvent.fill(ruleForm().getByRole("textbox", { name: "Ratio" }), "1");
  await userEvent.fill(ruleForm().getByRole("textbox", { name: "Seeding time" }), "8");
  await userEvent.click(ruleForm().getByRole("button", { name: "Save rule" }));
  await expect
    .element(
      ruleSection().getByText(
        /^seedpool requires a ratio of 1\.0 or 8 days of seeding for each torrent\./u,
      ),
    )
    .toBeVisible();
  await expect.element(ruleSection().getByText("Set by you.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Use what seedpool states" }));
  await expect.element(ruleSection().getByText("From seedpool's Releases.")).toBeVisible();

  await expect.element(page.getByText(/^seedpool follows the default goal/u)).toBeVisible();
  await userEvent.fill(goalForm().getByRole("textbox", { name: "Ratio" }), "3");
  await userEvent.click(goalForm().getByRole("button", { name: "Save goal" }));
  await expect.element(page.getByText(/^seedpool has its own goal/u)).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Use the default goal" }));
  await expect.element(page.getByText(/^seedpool follows the default goal/u)).toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "PUT rule",
      id: seedpoolId,
      body: { clears: "either", ratio: 1, seedSeconds: 691_200 },
    },
    { method: "DELETE rule", id: seedpoolId },
    {
      method: "PUT goal",
      id: seedpoolId,
      body: { combine: "first", ratio: 3, seedSeconds: 1_209_600, action: "remove-delete" },
    },
    { method: "DELETE goal", id: seedpoolId },
  ]);
});

test("the Indexers tab lists the Indexers that reach the Tracker with their switches", async () => {
  renderApp({ path: `${trackerPage}/indexers`, handlers: trackersServer().handlers });
  const rows = page.getByRole("list", { name: "Indexers reaching seedpool" });
  await expect.element(rows.getByRole("link", { name: "Larkspur" })).toBeVisible();
  await expect.element(rows.getByRole("switch").first()).toBeVisible();
  expect(rows.getByRole("listitem").all()).toHaveLength(1);
});

test("Remove shows what still uses the Tracker and keeps it", async () => {
  const inUse = "seedpool is still in use. Delete its Indexer and remove its 3 torrents first.";
  const server = trackersServer({ inUse });
  renderApp({ path: trackerPage, handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "seedpool actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Remove…" }));
  const dialog = page.getByRole("dialog", { name: "Remove seedpool" });
  await expect.element(dialog.getByText(/also deletes its seeding rule/u)).toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Remove" }));
  await expect.element(dialog.getByRole("alert")).toMatchTextContent(inUse);
  expect(server.state.trackers.some((tracker) => tracker.id === seedpoolId)).toBe(true);
});

test("Remove of a Tracker nothing uses goes back to Settings > Trackers", async () => {
  const server = trackersServer();
  const { router } = renderApp({ path: trackerPage, handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "seedpool actions" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Remove…" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "Remove seedpool" }).getByRole("button", { name: "Remove" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe("/settings/trackers");
  expect(page.getByRole("link", { name: /^seedpool/u }).query()).toBeNull();
});

test("on a phone the Tracker page lists its sections as rows, with no tab row", async () => {
  await asPhone();
  const { router } = renderApp({ path: trackerPage, handlers: trackersServer().handlers });
  await expect.element(page.getByText(/^seedpool requires 10 days of seeding/u)).toBeVisible();
  await expect.element(tabs()).not.toBeInTheDocument();
  const sections = page.getByRole("list", { name: "seedpool" });
  expect(
    sections
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Seeding2 needs attention", "Routing", "Indexers1"]);
  await userEvent.click(sections.getByRole("link", { name: /^Seeding/u }));
  await expect.poll(() => router.state.location.pathname).toBe(`${trackerPage}/seeding`);
  const back = page.getByRole("banner").getByRole("link", { name: "seedpool" });
  await expect.element(back).toHaveAttribute("href", trackerPage);
});

test("Settings > Trackers shows how often the Seeding goals task runs", async () => {
  const server = trackersServer();
  renderApp({ path: "/settings/trackers", handlers: server.handlers });
  const cadence = page.getByRole("definition").filter({ hasText: "Every 15 min" });
  await expect.element(cadence).toBeVisible();
  await expect.element(page.getByRole("term").filter({ hasText: "Seeding goals" })).toBeVisible();
});

const stoppedSeeding: HealthItem = {
  key: seedpoolId,
  subject: "seedpool",
  level: "warn",
  text: "3 torrents on seedpool stopped seeding before seedpool's rule was met. Resume or retry them before seedpool counts a hit and run.",
  since: minutesAgo(400),
  changedAt: minutesAgo(400),
  unsure: false,
  detail: [],
  action: null,
};

test("Settings > Trackers carries the Seeding Health category's warning on its link and at its top", async () => {
  const health = healthServer(reportOf({ seeding: [stoppedSeeding] }));
  renderApp({
    path: "/settings/trackers",
    handlers: [...health.handlers, ...trackersServer().handlers],
  });
  await expect
    .element(
      page
        .getByRole("navigation", { name: "Settings" })
        .getByRole("link", { name: "Trackers 1 warning" }),
    )
    .toBeVisible();
  await expect.element(page.getByText(stoppedSeeding.text)).toBeVisible();
});
