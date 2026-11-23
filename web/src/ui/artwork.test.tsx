import { http } from "msw";
import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { artwork, artworkHandlers, artworkResponse, artworkUrl } from "../test/ambient/fixtures";
import { renderKit } from "../test/render-kit";
import { worker } from "../test/worker";
import { Artwork, type ArtworkProps, initialsOf } from "./artwork";

const red = artworkUrl("red");
const missing = "/api/v1/artwork/missing";

const slot = () => document.querySelector<HTMLElement>("[data-art]");
const stateOf = () => slot()?.dataset["artState"];

let served = 0;

// A route no earlier test loaded, so the browser holds no decoded copy of it.
function fresh(): { url: string; release: () => void } {
  served += 1;
  const url = `/artwork/fresh-${served}.png`;
  let release: (() => void) | undefined;
  const waiting = new Promise<void>((resolve) => {
    release = resolve;
  });
  worker.use(
    http.get(url, async () => {
      await waiting;
      return artworkResponse(artwork.white);
    }),
  );
  return {
    url,
    release: () => {
      release?.();
    },
  };
}

test("each slot asks the proxy for its size", async () => {
  worker.use(...artworkHandlers);
  const cases: readonly [ArtworkProps, string][] = [
    [{ use: "poster", size: "row", url: red }, "small"],
    [{ use: "poster", size: "card", url: red, label: "Orchard", quiet: false }, "medium"],
    [{ use: "headshot", size: "tile", url: red, label: "Ada Moss" }, "medium"],
    [{ use: "backdrop", size: "backdrop", url: red, brightness: 1 }, "large"],
  ];
  for (const [props, size] of cases) {
    const container = renderKit(<Artwork {...props} />);
    // oxlint-disable-next-line no-await-in-loop -- one slot at a time names the one that fails
    await expect
      .poll(() => container.querySelector("img")?.getAttribute("src"), { message: props.size })
      .toBe(`${red}?size=${size}`);
    container.remove();
  }
});

test("a poster on its way keeps an empty frame, then fades in", async () => {
  const { url, release } = fresh();
  renderKit(<Artwork use="poster" size="card" url={url} label="The Quiet Orchard" quiet={false} />);
  await expect.poll(stateOf).toBe("loading");
  expect(slot()?.textContent).toBe("");
  release();
  await expect.poll(stateOf).toBe("faded");
});

test("a missing or failed image leaves the slot's placeholder", async () => {
  const cases: readonly [ArtworkProps, string][] = [
    [
      { use: "poster", size: "card", url: undefined, label: "The Quiet Orchard", quiet: false },
      "The Quiet Orchard",
    ],
    [
      { use: "poster", size: "hero", url: missing, label: "The Quiet Orchard" },
      "The Quiet Orchard",
    ],
    [{ use: "headshot", size: "tile", url: missing, label: "Ada Moss" }, "AM"],
    [{ use: "logo", size: "title", url: missing, label: "The Quiet Orchard" }, "The Quiet Orchard"],
    [{ use: "logo", size: "mark", url: missing }, ""],
    [{ use: "poster", size: "row", url: missing }, ""],
  ];
  for (const [props, text] of cases) {
    const container = renderKit(<Artwork {...props} />);
    const art = () => container.querySelector<HTMLElement>("[data-art]");
    // oxlint-disable-next-line no-await-in-loop -- one slot at a time names the one that fails
    await expect
      .poll(() => art()?.dataset["artState"], { message: props.size })
      .toMatch(/^(none|failed)$/u);
    expect(art()?.querySelector("img"), props.size).toBeNull();
    expect(art()?.textContent, props.size).toBe(text);
    container.remove();
  }
});

test("a headshot shows the person's initials while the photo loads", async () => {
  const { url, release } = fresh();
  renderKit(<Artwork use="headshot" size="tile" url={url} label="Ada Moss" />);
  await expect.poll(stateOf).toBe("loading");
  expect(slot()?.textContent).toBe("AM");
  release();
  await expect.poll(stateOf).toBe("faded");
});

test("a new image starts over, and one already decoded shows at once", async () => {
  const first = fresh();
  const second = fresh();
  first.release();
  function Switcher() {
    const [url, setUrl] = useState(first.url);
    return (
      <>
        <button
          type="button"
          onClick={() => {
            setUrl(url === first.url ? second.url : first.url);
          }}
        >
          Next
        </button>
        <Artwork use="poster" size="row" url={url} />
      </>
    );
  }
  renderKit(<Switcher />);
  await expect.poll(stateOf).toBe("faded");
  await userEvent.click(page.getByRole("button", { name: "Next" }));
  expect(slot()?.querySelector("img")?.getAttribute("src")).toBe(`${second.url}?size=small`);
  expect(stateOf()).toBe("loading");
  second.release();
  await expect.poll(stateOf).toBe("faded");
  await userEvent.click(page.getByRole("button", { name: "Next" }));
  await expect.poll(stateOf).toBe("at-once");
});

test("initials are the first letters of a name's first and last words", () => {
  expect(
    ["Timothée Chalamet", "zendaya", "Mary Elizabeth Winstead", "  ", "🍎 Pie"].map((name) =>
      initialsOf(name),
    ),
  ).toEqual(["TC", "Z", "MW", "", "🍎P"]);
});
