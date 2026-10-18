import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { z } from "zod";

import type { NodesSettings, SettingDurationField } from "../../../api/gen/model";
import { renderApp } from "../../../test/render-app";

const duration = (key: string, label: string, value: number): SettingDurationField => ({
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min: 60,
  max: 2_592_000,
  step: 60,
});

const nodeSettings = (offlineGrace: number): NodesSettings => ({
  joinCodeLifetime: {
    ...duration("nodes.joinCodeLifetime", "Join Code lifetime", 3600),
    min: 600,
    max: 604_800,
  },
  offlineGrace: {
    ...duration("nodes.offlineGrace", "Offline grace", offlineGrace),
    default: 900,
    isDefault: offlineGrace === 900,
    min: 60,
    max: 86_400,
  },
});

test("Nodes shows the Join Code lifetime and the offline grace, and saves the grace in seconds", async () => {
  const sent: unknown[] = [];
  let grace = 900;
  renderApp({
    path: "/settings/nodes",
    handlers: [
      http.get("*/api/v1/nodes/settings", () => HttpResponse.json(nodeSettings(grace))),
      http.patch("*/api/v1/nodes/settings", async ({ request }) => {
        const body = z.object({ offlineGrace: z.number() }).parse(await request.json());
        sent.push(body);
        grace = body.offlineGrace;
        return HttpResponse.json(nodeSettings(grace));
      }),
    ],
  });
  await expect.element(page.getByRole("heading", { name: "Nodes", level: 1 })).toBeVisible();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .all()
      .map((heading) => heading.element().textContent),
  ).toEqual(["Join Codes", "Offline Nodes"]);
  await expect.element(page.getByRole("textbox", { name: "Join Code lifetime" })).toHaveValue("1");
  const offline = page.getByRole("region", { name: "Offline Nodes" });
  await expect.element(offline.getByText("Default 15 min. Between 1 min and 1 day.")).toBeVisible();

  await userEvent.fill(offline.getByRole("textbox", { name: "Offline grace" }), "2");
  await userEvent.click(offline.getByRole("button", { name: "Save" }));
  await expect.poll(() => sent).toEqual([{ offlineGrace: 120 }]);
  await expect.element(offline.getByRole("button", { name: "Reset to default" })).toBeVisible();
});
