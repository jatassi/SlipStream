import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { WorksContentType, WorksFieldSpec, WorksWorkView } from "../../api/gen/model";
import { cinema, orchard } from "../../test/library";
import { renderKit } from "../../test/render-kit";
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
    { key: "tmdb", kind: "page", label: "TMDB", url: "https://www.themoviedb.org/movie/9000001" },
    { key: "trailer", kind: "trailer", label: "Trailer", url: "https://www.youtube.com/watch?v=x" },
  ],
};

test("About brackets a field's With value and shows its Art beside it", async () => {
  renderKit(<About type={rated} work={work} />);
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
  renderKit(<About type={rated} work={work} />);
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
