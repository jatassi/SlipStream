import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { WorksContentType, WorksFieldSpec, WorksWorkView } from "../../api/gen/model";
import { anthologies, longShelf } from "../../test/anthology";
import { cinema, orchard } from "../../test/library";
import { renderKit } from "../../test/render-kit";
import { shelfCells } from "../../test/shelf-cells";
import { About } from "./about";

const spec = (key: string, label: string, more: Partial<WorksFieldSpec>): WorksFieldSpec => ({
  key,
  label,
  tier: "work",
  kind: "text",
  in: ["about"],
  list: { column: false, sort: false, filter: false },
  with: null,
  art: null,
  ...more,
});

const rated: WorksContentType = {
  ...cinema,
  fields: [
    spec("certification", "Rated", { with: "country" }),
    spec("country", "Country", { in: [] }),
    spec("studio", "Studio", { art: "studio-logo" }),
  ],
};

const work: WorksWorkView = {
  ...orchard,
  fields: [
    { key: "certification", kind: "text", text: "R" },
    { key: "country", kind: "text", text: "United States" },
    { key: "studio", kind: "text", text: "Orchard Pictures" },
  ],
  artwork: [{ kind: "studio-logo", url: "/api/v1/artwork/orchard-pictures" }],
  links: [
    {
      key: "tmdb",
      kind: "page",
      label: "TMDB",
      url: "https://www.themoviedb.org/movie/9000001",
    },
    {
      key: "trailer",
      kind: "trailer",
      label: "Trailer",
      url: "https://www.youtube.com/watch?v=x",
    },
  ],
};

test("About brackets a field's With value and shows its Art beside it", async () => {
  renderKit(<About type={rated} work={work} statuses={[]} />);
  const about = page.getByRole("region", { name: "About" });
  await expect
    .element(about.getByRole("definition").first())
    .toHaveTextContent("R (United States)");
  expect(
    about
      .getByRole("term")
      .all()
      .map((term) => term.element().textContent),
  ).toEqual(["Rated", "Studio"]);
  const studio = about.getByRole("definition").nth(1);
  await expect.element(studio).toHaveTextContent("Orchard Pictures");
  expect(studio.element().querySelector("img")?.getAttribute("src")).toBe(
    "/api/v1/artwork/orchard-pictures?size=small",
  );
  await expect
    .element(about.getByText("Counts as out on April 9, 2021, Digital release in United States."))
    .toBeVisible();
  await expect.element(about.getByText("TMDB lists this film as 9000001.")).toBeVisible();
});

test("its links open the source's page and the trailer in a new tab", async () => {
  renderKit(<About type={rated} work={work} statuses={[]} />);
  await expect.element(page.getByRole("list", { name: "Links" })).toBeVisible();
  expect(
    page
      .getByRole("list", { name: "Links" })
      .getByRole("link")
      .all()
      .map((link) => {
        const anchor = link.element();
        return [
          anchor.textContent,
          anchor.getAttribute("href"),
          anchor.getAttribute("target"),
          anchor.getAttribute("rel"),
        ];
      }),
  ).toEqual([
    ["TMDB", "https://www.themoviedb.org/movie/9000001", "_blank", "noopener noreferrer"],
    ["Trailer", "https://www.youtube.com/watch?v=x", "_blank", "noopener noreferrer"],
  ]);
});

test("a many-Item Work's About names the latest Item out and the next one to come", async () => {
  const [, , next, later] = longShelf.items;
  const cells = shelfCells(longShelf, (item) =>
    item === next || item === later ? "unreleased" : "met",
  );
  renderKit(<About type={anthologies} work={longShelf} statuses={cells.items} />);
  const about = page.getByRole("region", { name: "About" });
  await expect
    .element(about.getByText("The latest story, V01S27, counted as out on January 28, 2020."))
    .toBeVisible();
  await expect
    .element(about.getByText("The next story, V01S03, counts as out on January 4, 2020."))
    .toBeVisible();
});

test("an Item not out yet is the next one whatever its cells hold, such as a download", async () => {
  const [, , next] = longShelf.items;
  const cells = shelfCells(longShelf, (item) => (item === next ? "downloading" : "met"));
  for (const status of cells.items) {
    status.released = status.itemId !== next?.id;
  }
  renderKit(<About type={anthologies} work={longShelf} statuses={cells.items} />);
  await expect
    .element(
      page
        .getByRole("region", { name: "About" })
        .getByText("The next story, V01S03, counts as out on January 4, 2020."),
    )
    .toBeVisible();
});

const plex =
  "https://app.plex.invalid/desktop/#!/server/living-room/details?key=/library/metadata/7";
const den = "https://den.invalid/web/#/details?id=a&serverId=b";
const loft = "https://loft.invalid/web/#/details?id=c&serverId=d";

const held: WorksWorkView = {
  ...work,
  openIn: [
    { serverId: "s1", kind: "plex", label: "Plex", name: "Living room Plex", url: plex },
    { serverId: "s2", kind: "jellyfin", label: "Jellyfin", name: "Den", url: den },
    { serverId: "s3", kind: "jellyfin", label: "Jellyfin", name: "Loft", url: loft },
  ],
};

test("Open in links each Media Server that holds the Work in a new tab, by name when two share a kind", async () => {
  renderKit(<About type={rated} work={held} statuses={[]} />);
  const list = page.getByRole("list", { name: "Open in" });
  await expect.element(list).toBeVisible();
  expect(
    list
      .getByRole("link")
      .all()
      .map((link) => {
        const anchor = link.element();
        return [
          anchor.textContent,
          anchor.getAttribute("href"),
          anchor.getAttribute("target"),
          anchor.getAttribute("rel"),
        ];
      }),
  ).toEqual([
    ["Open in Plex", plex, "_blank", "noopener noreferrer"],
    ["Open in Den", den, "_blank", "noopener noreferrer"],
    ["Open in Loft", loft, "_blank", "noopener noreferrer"],
  ]);
});

test("a Work on no Media Server shows no Open in", async () => {
  renderKit(<About type={rated} work={work} statuses={[]} />);
  await expect.element(page.getByRole("list", { name: "Links" })).toBeVisible();
  await expect.element(page.getByRole("list", { name: "Open in" })).not.toBeInTheDocument();
});
