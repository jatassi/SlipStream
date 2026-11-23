import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { artworkHandlers, artworkUrl } from "../../test/ambient/fixtures";
import { renderKit } from "../../test/render-kit";
import { worker } from "../../test/worker";
import { TitleHeading } from "./hero-art";

const treatment = artworkUrl("white");

test("the title treatment is the heading's image, and the label its accessible name", async () => {
  worker.use(...artworkHandlers);
  renderKit(<TitleHeading label="The Quiet Orchard" treatment={treatment} />);
  const heading = page.getByRole("heading", { name: "The Quiet Orchard", level: 1 });
  await expect.element(heading).toBeVisible();
  await expect.poll(() => heading.element().querySelector("img")?.complete).toBe(true);
  const image = heading.element().querySelector("img");
  expect(image?.getAttribute("alt")).toBe("");
  expect(image?.getAttribute("src")).toBe(`${treatment}?size=medium`);
  expect(image?.naturalWidth).toBeGreaterThan(0);
});

test("a Work without a title treatment shows its label as text", async () => {
  renderKit(<TitleHeading label="The Quiet Orchard" treatment={undefined} />);
  const heading = page.getByRole("heading", { name: "The Quiet Orchard", level: 1 });
  await expect.element(heading).toHaveTextContent("The Quiet Orchard");
  expect(heading.element().querySelector("img")).toBeNull();
});

test("a title treatment that fails to load falls back to the label", async () => {
  renderKit(<TitleHeading label="The Quiet Orchard" treatment="/api/v1/artwork/missing" />);
  await expect.poll(() => document.querySelector("h1 img")).toBeNull();
  await expect
    .element(page.getByRole("heading", { name: "The Quiet Orchard", level: 1 }))
    .toHaveTextContent("The Quiet Orchard");
});
