import { type KeyboardEvent, useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { type Move, moveIndex } from "./move-index";
import { VirtualListbox, optionId } from "./virtual-listbox";

import { renderKit } from "../test/render-kit";

const folders = Array.from({ length: 5000 }, (_, index) => `Folder ${index + 1}`);

const keys: Readonly<Record<string, Move>> = {
  ArrowDown: "next",
  ArrowUp: "prev",
  Home: "first",
  End: "last",
  PageDown: "pageDown",
  PageUp: "pageUp",
};

function Picker() {
  const [active, setActive] = useState(0);
  const [chosen, setChosen] = useState("nothing");
  return (
    <div className="flex h-[440px] flex-col">
      <input
        role="combobox"
        aria-label="Folders"
        aria-expanded
        aria-controls="folders"
        aria-activedescendant={optionId("folders", active)}
        onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => {
          const move = keys[event.key];
          if (move !== undefined) {
            event.preventDefault();
            setActive((from) => moveIndex(from, move, folders.length, 10));
          }
        }}
      />
      <output>Chose {chosen}</output>
      <VirtualListbox
        id="folders"
        label="Folders"
        rows={folders}
        rowKey={(row) => row}
        estimate={44}
        active={active}
        onChoose={(index) => {
          setChosen(folders[index] ?? "a missing row");
        }}
        renderRow={(row, isActive) => (
          <div className={`h-11 ${isActive ? "bg-fill-hover" : ""}`}>{row}</div>
        )}
      />
    </div>
  );
}

const field = () => page.getByRole("combobox", { name: "Folders" });
const option = (name: string) => page.getByRole("option", { name, exact: true });

async function expectActive(name: string, place: number) {
  await expect.element(option(name)).toBeInViewport();
  await expect.element(option(name)).toHaveAttribute("aria-selected", "true");
  await expect.element(option(name)).toHaveAttribute("aria-posinset", String(place));
  await expect.element(option(name)).toHaveAttribute("aria-setsize", "5000");
  expect(field().element().getAttribute("aria-activedescendant")).toBe(option(name).element().id);
}

test("End and Home carry the active row through 5,000 folders, kept in view", async () => {
  renderKit(<Picker />);
  await userEvent.click(field());
  await expectActive("Folder 1", 1);

  await userEvent.keyboard("{End}");
  await expectActive("Folder 5000", 5000);
  expect(page.getByRole("option").all().length).toBeLessThan(40);

  await userEvent.keyboard("{PageUp}{ArrowUp}");
  await expectActive("Folder 4989", 4989);

  await userEvent.keyboard("{Home}");
  await expectActive("Folder 1", 1);
  await expect.element(option("Folder 5000")).not.toBeInTheDocument();
});

test("a click chooses its row and leaves focus in the field", async () => {
  renderKit(<Picker />);
  await userEvent.click(field());
  await userEvent.click(option("Folder 3"));
  await expect.element(page.getByRole("status")).toHaveTextContent("Chose Folder 3");
  await expect.element(field()).toHaveFocus();
});
