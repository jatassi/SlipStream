import { useState } from "react";
import { expect, test } from "vitest";
import { commands, page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Checkbox } from "./checkbox";
import { type RankedRow, RankedList, tiersOf } from "./ranked-list";

const item = (id: string, tiedAbove = false): RankedRow<string> => ({
  kind: "item",
  id,
  value: id,
  tiedAbove,
});
const cutoff: RankedRow<string> = { kind: "marker", id: "cutoff", label: "Cutoff" };

let latest: readonly RankedRow<string>[] = [];

function Qualities({ initial, ties = false }: { initial: RankedRow<string>[]; ties?: boolean }) {
  const [rows, setRows] = useState(initial);
  const [accepted, setAccepted] = useState<ReadonlySet<string>>(new Set());
  return (
    <RankedList
      label="Qualities"
      caption="Higher ranks are grabbed first."
      rows={rows}
      onChange={(next) => {
        latest = next;
        setRows(next);
      }}
      name={(value) => value}
      renderRow={(value) => <span>{value}</span>}
      renderTrailing={(value) => (
        <Checkbox
          label={`Accept ${value}`}
          checked={accepted.has(value)}
          onCheckedChange={(checked) => {
            setAccepted((current) =>
              checked
                ? new Set([...current, value])
                : new Set([...current].filter((v) => v !== value)),
            );
          }}
        />
      )}
      ties={ties}
    />
  );
}

const list = () => page.getByRole("list", { name: "Qualities" });

const shown = () =>
  list()
    .getByRole("listitem")
    .all()
    .map((row) => row.element().textContent.replace(/Accept \w+$/u, ""));

const announced = () => page.getByText(/ moved /u);

test("the arrow keys on a handle move its row and keep focus there, announcing the rank", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB"), item("HDTV")]} />);
  await expect.element(list()).toHaveAccessibleDescription("Higher ranks are grabbed first.");
  expect(shown()).toEqual(["Rank 1Bluray", "Rank 2WEB", "Rank 3HDTV"]);

  const handle = page.getByRole("button", { name: "Move HDTV", exact: true });
  await expect
    .element(handle)
    .toHaveAccessibleDescription("Press the up and down arrow keys to move it.");
  handle.element().focus();
  await userEvent.keyboard("{ArrowUp}");
  await expect.poll(shown).toEqual(["Rank 1Bluray", "Rank 2HDTV", "Rank 3WEB"]);
  await expect.element(handle).toHaveFocus();
  await expect.element(announced()).toHaveTextContent("HDTV moved to rank 2 of 3.");

  await userEvent.keyboard("{ArrowUp}");
  await userEvent.keyboard("{ArrowUp}");
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2Bluray", "Rank 3WEB"]);
  await expect.element(handle).toHaveFocus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.poll(shown).toEqual(["Rank 1Bluray", "Rank 2HDTV", "Rank 3WEB"]);
});

test("Move up and Move down step a row, and the end rows can't step past the list", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB"), item("HDTV")]} />);
  await expect.element(page.getByRole("button", { name: "Move Bluray up" })).toBeDisabled();
  await expect.element(page.getByRole("button", { name: "Move HDTV down" })).toBeDisabled();

  await userEvent.click(page.getByRole("button", { name: "Move Bluray down" }));
  await expect.poll(shown).toEqual(["Rank 1WEB", "Rank 2Bluray", "Rank 3HDTV"]);
  await expect.element(page.getByRole("button", { name: "Move Bluray down" })).toHaveFocus();
  await expect.element(announced()).toHaveTextContent("Bluray moved to rank 2 of 3.");

  await userEvent.click(page.getByRole("button", { name: "Move Bluray down" }));
  await expect.poll(shown).toEqual(["Rank 1WEB", "Rank 2HDTV", "Rank 3Bluray"]);
  await expect.element(page.getByRole("button", { name: "Move Bluray down" })).toBeDisabled();
  await expect
    .element(page.getByRole("button", { name: "Move Bluray", exact: true }))
    .toHaveFocus();
});

test("a row tied with the one above shares its rank, and a step takes it out of the tier", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB"), item("HDTV")]} ties />);
  await expect
    .element(page.getByRole("button", { name: "Tie Bluray with the one above" }))
    .toBeDisabled();
  const tieWeb = page.getByRole("button", { name: "Tie WEB with the one above" });
  await expect.element(tieWeb).toHaveAttribute("aria-pressed", "false");

  await userEvent.click(tieWeb);
  await expect.element(tieWeb).toHaveAttribute("aria-pressed", "true");
  expect(shown()).toEqual(["Rank 1Bluray", "Rank 1WEB", "Rank 2HDTV"]);
  expect(tiersOf(latest)).toEqual([["Bluray", "WEB"], ["HDTV"]]);

  await userEvent.click(page.getByRole("button", { name: "Move HDTV up" }));
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2Bluray", "Rank 2WEB"]);

  await userEvent.click(page.getByRole("button", { name: "Move Bluray down" }));
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2WEB", "Rank 3Bluray"]);
  expect(tiersOf(latest)).toEqual([["HDTV"], ["WEB"], ["Bluray"]]);

  await userEvent.click(page.getByRole("button", { name: "Tie Bluray with the one above" }));
  await userEvent.click(page.getByRole("button", { name: "Tie WEB with the one above" }));
  expect(shown()).toEqual(["Rank 1HDTV", "Rank 1WEB", "Rank 1Bluray"]);
  await userEvent.click(page.getByRole("button", { name: "Tie WEB with the one above" }));
  expect(tiersOf(latest)).toEqual([["HDTV"], ["WEB", "Bluray"]]);
});

test("a marker moves like a row, passing a whole tier, and holds no rank", async () => {
  renderKit(
    <Qualities
      initial={[item("Remux"), cutoff, item("Bluray"), item("WEB", true), item("HDTV")]}
      ties
    />,
  );
  await expect
    .poll(shown)
    .toEqual(["Rank 1Remux", "Cutoff", "Rank 2Bluray", "Rank 2WEB", "Rank 3HDTV"]);
  await expect
    .element(page.getByRole("button", { name: "Tie Bluray with the one above" }))
    .toBeDisabled();

  const handle = page.getByRole("button", { name: "Move Cutoff", exact: true });
  handle.element().focus();
  await userEvent.keyboard("{ArrowDown}");
  await expect
    .poll(shown)
    .toEqual(["Rank 1Remux", "Rank 2Bluray", "Rank 2WEB", "Cutoff", "Rank 3HDTV"]);
  await expect.element(handle).toHaveFocus();
  await expect.element(announced()).toHaveTextContent("Cutoff moved below WEB.");

  await userEvent.click(page.getByRole("button", { name: "Move Cutoff down" }));
  await expect
    .poll(shown)
    .toEqual(["Rank 1Remux", "Rank 2Bluray", "Rank 2WEB", "Rank 3HDTV", "Cutoff"]);
  await expect.element(page.getByRole("button", { name: "Move Cutoff down" })).toBeDisabled();
  expect(tiersOf(latest)).toEqual([["Remux"], ["Bluray", "WEB"], ["HDTV"]]);
});

test("a row's trailing control works in place", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB")]} />);
  const accept = page.getByRole("checkbox", { name: "Accept WEB" });
  await userEvent.click(accept);
  await expect.element(accept).toBeChecked();
  await userEvent.click(page.getByRole("button", { name: "Move WEB up" }));
  await expect.poll(shown).toEqual(["Rank 1WEB", "Rank 2Bluray"]);
  await expect.element(page.getByRole("checkbox", { name: "Accept WEB" })).toBeChecked();
});

test("dragging a row by its handle reorders the list", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB"), item("HDTV")]} />);
  await commands.drag('[data-row="HDTV"] [data-control="handle"]', '[data-row="Bluray"]');
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2Bluray", "Rank 3WEB"]);
  await expect.element(announced()).toHaveTextContent("HDTV moved to rank 1 of 3.");
});

test("a row dragged into a tier joins it, and a marker dragged into one passes it", async () => {
  renderKit(
    <Qualities
      initial={[item("Remux"), cutoff, item("Bluray"), item("WEB", true), item("HDTV")]}
      ties
    />,
  );
  await commands.drag('[data-row="HDTV"] [data-control="handle"]', '[data-row="WEB"]');
  await expect
    .poll(shown)
    .toEqual(["Rank 1Remux", "Cutoff", "Rank 2Bluray", "Rank 2HDTV", "Rank 2WEB"]);

  await commands.drag('[data-row="cutoff"] [data-control="handle"]', '[data-row="Bluray"]');
  await expect
    .poll(shown)
    .toEqual(["Rank 1Remux", "Rank 2Bluray", "Rank 2HDTV", "Rank 2WEB", "Cutoff"]);
  expect(tiersOf(latest)).toEqual([["Remux"], ["Bluray", "HDTV", "WEB"]]);
});

test("at the Touch density one row action holds the moves and the tie", async () => {
  renderKit(<Qualities initial={[item("Bluray"), item("WEB"), item("HDTV")]} ties />, {
    shell: "phone",
  });
  await expect.element(page.getByRole("button", { name: "Move WEB up" })).not.toBeInTheDocument();
  await expect.element(page.getByRole("button", { name: "Move WEB", exact: true })).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Actions for WEB" }));
  await userEvent.click(page.getByRole("button", { name: "Tie with the one above" }));
  await expect.poll(shown).toEqual(["Rank 1Bluray", "Rank 1WEB", "Rank 2HDTV"]);
  await expect.element(page.getByRole("button", { name: "Actions for WEB" })).toHaveFocus();

  await userEvent.click(page.getByRole("button", { name: "Actions for HDTV" }));
  await userEvent.click(page.getByRole("button", { name: "Move up" }));
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2Bluray", "Rank 2WEB"]);
  await expect.element(announced()).toHaveTextContent("HDTV moved to rank 1 of 2.");

  await userEvent.click(page.getByRole("button", { name: "Actions for HDTV" }));
  await expect.element(page.getByRole("button", { name: "Move down" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Move up" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
  await userEvent.click(page.getByRole("button", { name: "Actions for WEB" }));
  await userEvent.click(page.getByRole("button", { name: "Untie from the one above" }));
  await expect.poll(shown).toEqual(["Rank 1HDTV", "Rank 2Bluray", "Rank 3WEB"]);
});
