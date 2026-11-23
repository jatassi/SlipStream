import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { homeId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { readyLead, subdl } from "../test/subtitles";
import { type SubtitlesWorld, subtitlesWorld } from "../test/subtitles-world";

const path = "/settings/subtitles";

function openSection(start: Parameters<typeof subtitlesWorld>[0] = {}) {
  const world: SubtitlesWorld = subtitlesWorld(start);
  renderApp({ path, handlers: world.handlers });
  return { world, section: page.getByRole("region", { name: "Subtitles", exact: true }) };
}

test("with no subtitle language yet, the lead says so and opens Profiles", async () => {
  const { section } = openSection({
    lead: { ...readyLead, kind: "no_languages", scans: [] },
    profiles: [],
  });
  await expect
    .element(
      section.getByText(
        "No Profile has a subtitle language yet. Add a language to a Profile to start.",
      ),
    )
    .toBeVisible();
  await expect
    .element(section.getByRole("link", { name: "Open Profiles" }))
    .toHaveAttribute("href", "/settings/profiles");
});

test("with movies wanted and no provider for them, the lead offers SubDL's key and names what Gestdown searches", async () => {
  const { section } = openSection({
    lead: { ...readyLead, kind: "uncovered", uncovered: ["cinema"] },
  });
  await expect
    .element(
      section.getByText(
        "Movies has no Subtitle Provider. Add a free SubDL key, which takes about a minute. SlipStream already searches Gestdown for TV.",
      ),
    )
    .toBeVisible();
  await userEvent.click(section.getByRole("button", { name: "Add SubDL key" }));
  await expect.element(page.getByRole("dialog", { name: "Add your SubDL key" })).toBeVisible();
});

test("with Subtitle search off, the lead says how to turn it on", async () => {
  const { section } = openSection({ lead: { ...readyLead, kind: "off" } });
  await expect
    .element(section.getByText("Subtitle search is off. Turn it on to fetch missing subtitles."))
    .toBeVisible();
});

test("when ready, the lead counts the Profiles and missing subtitles, the cadence and each Node's scans, then lists each Profile's languages", async () => {
  const { section } = openSection();
  await expect
    .element(
      section.getByText(
        "Subtitles are on for 1 Profile. 1,204 subtitles are missing, and Subtitle search runs every 8 hours.",
      ),
    )
    .toBeVisible();
  await expect
    .element(section.getByRole("list", { name: "Audio scans left" }))
    .toMatchTextContent(/home has 312 files/u);
  const row = section.getByRole("link", { name: /Movies/u });
  await expect.element(row).toHaveTextContent("MoviesEnglish, full, hearing impaired allowed");
});

test("a setting saves on its own, and Reset to default sends null", async () => {
  const { world } = openSection();
  const search = page.getByRole("region", { name: "Subtitle search" });
  await userEvent.click(search.getByRole("button", { name: "Details" }));
  await userEvent.fill(
    search.getByRole("textbox", { name: "Keep a subtitle only when its timing match is at least" }),
    "50",
  );
  await userEvent.click(search.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ values: { "subtitles.sync.match": 50 } });
  await expect
    .element(
      search.getByText(
        "0% is what random timing scores, and 100% means every line lands on speech.",
      ),
    )
    .toBeVisible();
  await userEvent.click(search.getByRole("button", { name: "Reset to default" }));
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ values: { "subtitles.sync.match": null } });
  await expect
    .element(search.getByRole("button", { name: "Reset to default" }))
    .not.toBeInTheDocument();
});

function searchFields() {
  const search = page.getByRole("region", { name: "Subtitle search" });
  return {
    search,
    every: search.getByRole("textbox", { name: "Subtitle search runs every" }),
    gap: search.getByRole("textbox", {
      name: "Search for a subtitle that keeps coming up empty at least every",
    }),
    save: search.getByRole("button", { name: "Save" }),
  };
}

test("an interval longer than the longest gap is refused on the interval", async () => {
  const { world } = openSection();
  const { search, every, gap, save } = searchFields();
  await userEvent.click(search.getByRole("button", { name: "Details" }));
  await userEvent.fill(gap, "1");
  await userEvent.click(save);
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ values: { "subtitles.longestGap": 86_400 } });
  await userEvent.fill(every, "36");
  await userEvent.click(save);
  await expect
    .element(
      search
        .getByRole("alert")
        .getByText(
          "Use 1 day or shorter. A subtitle that keeps coming up empty is searched for at least every 1 day.",
        ),
    )
    .toBeVisible();
});

test("a longest gap shorter than the interval is refused on the gap", async () => {
  const { world } = openSection();
  const { search, every, gap, save } = searchFields();
  await userEvent.fill(every, "36");
  await userEvent.click(save);
  await expect
    .poll(() => world.writes.at(-1)?.body)
    .toEqual({ values: { "tasks.subtitle-search.cadence": 129_600 } });
  await userEvent.click(search.getByRole("button", { name: "Details" }));
  await userEvent.fill(gap, "1");
  await userEvent.click(save);
  await expect
    .element(
      search
        .getByRole("alert")
        .getByText("Use 36 hours or longer. Subtitle search runs every 36 hours."),
    )
    .toBeVisible();
});

test("each Storage Node's card sets its audio scans at once and says how many files are left", async () => {
  const { world } = openSection();
  const card = page.getByRole("region", { name: "home" });
  await expect.element(card.getByText("home has 312 files left to scan for speech.")).toBeVisible();
  await expect.element(page.getByRole("region", { name: "seedbox" })).not.toBeInTheDocument();
  await userEvent.fill(
    card.getByRole("textbox", { name: "Audio scans at once on this Node" }),
    "4",
  );
  await userEvent.click(card.getByRole("button", { name: "Save" }));
  await expect
    .poll(() => world.writes.at(-1))
    .toEqual({
      path: `/api/v1/subtitles/nodes/${homeId}/settings`,
      body: { values: { "subtitles.mapSlots": 4 } },
    });
});

test("a provider's key opens from its row", async () => {
  openSection();
  await userEvent.click(
    page
      .getByRole("list", { name: "Subtitle Providers" })
      .getByRole("button", { name: "Add key" })
      .first(),
  );
  await expect
    .element(page.getByRole("dialog", { name: `Add your ${subdl.name} key` }))
    .toBeVisible();
});
