import type { Page } from "@playwright/test";
import { z } from "zod";

import { type World, expect, test } from "./fixtures.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 300_000 });

const NodeIds = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });

async function idOf(world: World, name: string): Promise<string> {
  const node = NodeIds.parse((await world.api("GET", "/api/v1/nodes")).body).nodes.find(
    (each) => each.name === name,
  );
  if (node === undefined) {
    throw new Error(`no Node named ${name}`);
  }
  return node.id;
}

async function centreOf(page: Page, selector: string): Promise<{ x: number; y: number }> {
  const box = await page.locator(selector).first().boundingBox();
  if (box === null) {
    throw new Error(`${selector} is not drawn`);
  }
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

test("the Diagram draws each Node and its link under the Node's own CSP, and a chip dropped on another Node opens the Review", async ({
  adminPage: page,
  world,
}) => {
  const home = await idOf(world, "home");
  const seedbox = await idOf(world, "seedbox");
  await page.goto("/nodes");
  const diagram = page.getByRole("figure", { name: "Diagram" });
  await expect(diagram.locator("[data-node]")).toHaveCount(2);
  await expect(diagram.locator(`[data-link="${seedbox}"]`)).toHaveCount(1);
  await expect(diagram.locator(`[data-node="${home}"]`)).toHaveAttribute("data-state", "online");
  const rolesOf = (selector: string): Promise<string[]> =>
    diagram
      .locator(selector)
      .evaluateAll((parts: SVGElement[]) =>
        [...new Set(parts.map((part) => part.dataset["role"] ?? ""))].toSorted((a, b) =>
          a.localeCompare(b),
        ),
      );
  const held = await Promise.all(
    [home, seedbox].map((node) =>
      Promise.all([
        rolesOf(`[data-node="${node}"] [data-role]:not([data-chip])`),
        rolesOf(`[data-node="${node}"] [data-chip]`),
      ]),
    ),
  );
  for (const [machines, chips] of held) {
    expect(machines).toEqual(chips);
  }

  const from = await centreOf(page, `[data-node="${seedbox}"] [data-chip="downloading"]`);
  const to = await centreOf(page, `[data-node="${home}"] [data-item="tile.base"]`);
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps: 20 });
  await expect(diagram.locator(`[data-node="${home}"]`)).toHaveAttribute(
    "data-verdict",
    /allow|warn/u,
  );
  await page.mouse.up();
  const review = page.getByRole("dialog", { name: "Review the change" });
  await expect(review).toBeVisible();
  await review.getByRole("button", { name: "Cancel" }).click();
  await expect(review).toHaveCount(0);
});
