import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { z } from "zod";

import type { TargetsTarget } from "../api/gen/model";
import { noImpact } from "../test/impact";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

test("a saved Target's Hold opens in its largest whole unit and saves in seconds", async () => {
  const sent: unknown[] = [];
  const held: TargetsTarget = { ...starter, holdSeconds: 5400 };
  renderApp({
    path: `/settings/targets/${starter.id}`,
    handlers: [
      ...targetHandlers([held, keeper]),
      http.put("*/api/v1/targets/:targetId", async ({ request }) => {
        const body = z.object({ holdSeconds: z.number() }).parse(await request.json());
        if (new URL(request.url).searchParams.get("preview") !== "true") {
          sent.push(body.holdSeconds);
        }
        return HttpResponse.json({
          target: { ...held, holdSeconds: body.holdSeconds },
          impact: noImpact,
        });
      }),
    ],
  });
  const hold = page.getByRole("textbox", {
    name: "Wait for a better release for",
  });
  await expect.element(hold).toHaveValue("90");
  await expect.element(page.getByRole("radio", { name: "minutes" })).toBeChecked();
  await userEvent.fill(hold, "8");
  await userEvent.click(page.getByRole("radio", { name: "days" }));
  await expect.element(page.getByText("Between 0 and 7 days.")).toBeVisible();
  await userEvent.fill(hold, "2");
  await userEvent.click(page.getByRole("radio", { name: "hours" }));
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await expect.poll(() => sent).toEqual([7200]);
  await expect.element(page.getByText("Saved 1080p.")).toBeVisible();
});
