import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { HealthItem } from "../api/gen/model";
import { healthServer, minutesAgo, reportOf } from "../test/health";
import { renderApp } from "../test/render-app";
import { seedpoolId, trackersServer } from "../test/trackers";

const trackerPage = `/settings/trackers/${seedpoolId}`;
const tabs = () => page.getByRole("navigation", { name: "seedpool" });
const goalForm = () => page.getByRole("form", { name: "Seeding goal" });
const ruleSection = () => page.getByRole("region", { name: "Rule" });
const ruleForm = () => page.getByRole("form", { name: "Seeding rule" });

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
  await userEvent.click(goalForm().getByRole("radio", { name: "h", exact: true }));
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
    .element(goalForm().getByText("Each torrent seeds until you remove it."))
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
  const margin = page.getByRole("textbox", { name: "Extra seeding time" });
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
  await expect.element(page.getByText("From seedpool's releases.")).toBeVisible();
  await expect
    .element(
      page.getByText(
        "2 torrents are seeding. 2 haven't met seedpool's seeding rule yet, and 1 of those isn't seeding.",
      ),
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
  await expect.element(ruleSection().getByText("From seedpool's releases.")).toBeVisible();

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
  const rows = page.getByRole("list", { name: "Indexers searching seedpool" });
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
  const dialog = page.getByRole("dialog", { name: "Remove seedpool?" });
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
    page.getByRole("dialog", { name: "Remove seedpool?" }).getByRole("button", { name: "Remove" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe("/settings/trackers");
  expect(page.getByRole("link", { name: /^seedpool/u }).query()).toBeNull();
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
  text: "3 torrents on seedpool stopped seeding before seedpool's seeding rule was met. Seed them again before seedpool counts a hit and run.",
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
