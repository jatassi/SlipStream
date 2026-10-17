import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { z } from "zod";

import type { TargetsTarget } from "../api/gen/model";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

const sentence = () => page.getByText(/^Grabs /u);

const newTarget = z.object({
  name: z.string(),
  tiers: z.array(z.array(z.string())),
  cutoff: z.string(),
  filters: z.array(
    z.object({ attribute: z.string(), mode: z.string(), values: z.array(z.string()) }),
  ),
  sizes: z.array(z.object({ quality: z.string(), min: z.number(), max: z.number().nullable() })),
});

function creating() {
  const sent: z.infer<typeof newTarget>[] = [];
  const handler = http.post("*/api/v1/targets", async ({ request }) => {
    const body = newTarget.parse(await request.json());
    sent.push(body);
    const made: TargetsTarget = {
      ...starter,
      id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e3b",
      name: body.name,
      profiles: [],
    };
    return HttpResponse.json(made, { status: 201 });
  });
  return { sent, handler };
}

async function pick(...names: string[]) {
  for (const name of names) {
    // oxlint-disable-next-line no-await-in-loop -- each tick lands before the next, as a person's does
    await userEvent.click(page.getByRole("checkbox", { name, exact: true }));
  }
}

test("a Target built from a preset says what its order means as the cutoff moves, and creates", async () => {
  const { sent, handler } = creating();
  const { router } = renderApp({
    path: "/settings/targets/new",
    handlers: [...targetHandlers(), handler],
  });
  await expect.element(page.getByRole("heading", { name: "New Target", level: 1 })).toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "HD");

  await userEvent.click(page.getByRole("button", { name: "Fill from a preset" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Upgrade for anything better" }));
  await pick("Remux-2160p", "Bluray-2160p", "WEBDL-2160p", "WEBRip-2160p", "HDTV-2160p");
  await pick("WEBRip-480p", "DVD", "SDTV");
  await expect
    .element(sentence())
    .toHaveTextContent("Grabs HDTV-720p or better. Upgrades until Remux-1080p, then stops.");

  page.getByRole("button", { name: "Move Cutoff", exact: true }).element().focus();
  await userEvent.keyboard("{ArrowDown}");
  await expect
    .element(sentence())
    .toHaveTextContent("Grabs HDTV-720p or better. Upgrades until Bluray-1080p, then stops.");
  const samples = page.getByRole("list", { name: "Sample moves" });
  await expect
    .element(samples.getByText("Bluray-1080p has reached the cutoff, so nothing replaces it."))
    .toBeVisible();
  await expect.element(samples.getByText("This Target doesn't accept Remux-2160p.")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Create Target" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/targets");
  expect(sent[0]).toMatchObject({
    name: "HD",
    cutoff: "bluray-1080p",
    tiers: [
      ["remux-1080p"],
      ["bluray-1080p"],
      ["webdl-1080p"],
      ["webrip-1080p"],
      ["hdtv-1080p"],
      ["bluray-720p"],
      ["webdl-720p"],
      ["webrip-720p"],
      ["hdtv-720p"],
    ],
    filters: [],
  });
  expect(sent[0]?.sizes).toContainEqual({ quality: "webdl-1080p", min: 12.5, max: null });
  await expect.element(page.getByText("Created HD.")).toBeVisible();
});

test("the cutoff stays below an accepted quality, and No upgrades keeps the first file", async () => {
  renderApp({ path: "/settings/targets/new", handlers: targetHandlers() });
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Grabs SDTV or WEBRip-480p or better. Upgrades until Bluray-2160p or Remux-2160p, then stops.",
    );
  await pick("Bluray-2160p", "Remux-2160p");
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Grabs SDTV or WEBRip-480p or better. Upgrades until HDTV-2160p, WEBRip-2160p or WEBDL-2160p, then stops.",
    );
  await userEvent.click(page.getByRole("button", { name: "Fill from a preset" }));
  await userEvent.click(page.getByRole("menuitem", { name: "No upgrades" }));
  await expect
    .element(sentence())
    .toHaveTextContent("Grabs SDTV or WEBRip-480p or better. Keeps the first file it grabs.");
});

test("CAM sits apart, off, and once accepted ranks below everything", async () => {
  const { sent, handler } = creating();
  renderApp({ path: "/settings/targets/new", handlers: [...targetHandlers(), handler] });
  const cinema = page.getByRole("checkbox", { name: "CAM", exact: true });
  await expect.element(cinema).not.toBeChecked();
  await expect
    .element(page.getByText(/Cinema recordings \(CAM, TS, TELESYNC, HDCAM\)/u))
    .toBeVisible();
  await userEvent.click(cinema);
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Grabs CAM or better. Upgrades until Bluray-2160p or Remux-2160p, then stops.",
    );
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Anything");
  await userEvent.click(page.getByRole("button", { name: "Create Target" }));
  await expect.poll(() => sent[0]?.tiers.at(-1)).toEqual(["cam"]);
});

test("a filter that isn't Any needs a value, and Original language is one", async () => {
  const { sent, handler } = creating();
  renderApp({ path: "/settings/targets/new", handlers: [...targetHandlers(), handler] });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "HDR");
  await userEvent.click(
    page.getByRole("radiogroup", { name: "HDR" }).getByRole("radio", { name: "Only these" }),
  );
  await expect.element(page.getByText("Pick at least one, or choose Any.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Create Target" }));
  await expect
    .element(page.getByRole("alert").getByText("Pick a value for every filter that isn't Any."))
    .toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "HDR: Pick values" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "HDR10", exact: true }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "DV" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("button", { name: "HDR: DV, HDR10" })).toBeVisible();
  await userEvent.click(
    page
      .getByRole("radiogroup", { name: "Audio language" })
      .getByRole("radio", { name: "Not these" }),
  );
  await userEvent.click(page.getByRole("button", { name: "Audio language: Pick values" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Original language" }));
  await userEvent.keyboard("{Escape}");

  await userEvent.click(page.getByRole("button", { name: "Create Target" }));
  await expect
    .poll(() => sent[0]?.filters)
    .toEqual([
      { attribute: "hdr", mode: "only", values: ["DV", "HDR10"] },
      { attribute: "language", mode: "not", values: ["original"] },
    ]);
});

test("each accepted quality has a size in MiB per minute and what it means for a two-hour film", async () => {
  renderApp({ path: "/settings/targets/new", handlers: targetHandlers() });
  const sizes = page.getByRole("list", { name: "Sizes" });
  await expect.element(sizes.getByRole("listitem")).toHaveLength(17);
  const max = page.getByRole("textbox", { name: "WEBDL-1080p maximum" });
  await expect
    .element(page.getByRole("textbox", { name: "WEBDL-1080p minimum" }))
    .toHaveValue("12.5");
  await userEvent.fill(max, "abc");
  await expect
    .element(page.getByText("Enter a number, or leave it empty for no limit."))
    .toBeVisible();
  await userEvent.fill(max, "40");
  await expect.element(page.getByText("A two-hour film takes 1.5 GB to 4.7 GB.")).toBeVisible();
  await pick("WEBDL-1080p");
  await expect.element(sizes.getByRole("listitem")).toHaveLength(16);
});

test("a saved Target opens as its draft, its accepted qualities first", async () => {
  renderApp({ path: `/settings/targets/${starter.id}`, handlers: targetHandlers() });
  await expect.element(page.getByRole("heading", { name: "1080p", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("1080p");
  await expect.element(page.getByRole("checkbox", { name: "WEBDL-1080p" })).toBeChecked();
  await expect.element(page.getByRole("checkbox", { name: "Remux-2160p" })).not.toBeChecked();
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Grabs HDTV-720p, WEBRip-720p or WEBDL-720p or better. Upgrades until Bluray-1080p or Remux-1080p, then stops.",
    );
  await expect
    .element(page.getByRole("textbox", { name: "WEBDL-1080p maximum" }))
    .toHaveValue("40");
  await expect.element(page.getByRole("button", { name: "Create Target" })).not.toBeInTheDocument();
});

test("a saved Target's filters open as they were saved", async () => {
  renderApp({ path: `/settings/targets/${keeper.id}`, handlers: targetHandlers() });
  await expect
    .element(
      page.getByRole("radiogroup", { name: "HDR" }).getByRole("radio", { name: "Only these" }),
    )
    .toBeChecked();
  await expect.element(page.getByRole("button", { name: "HDR: HDR10" })).toBeVisible();
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Grabs WEBDL-2160p or WEBRip-2160p or better. Keeps the first file it grabs.",
    );
});
