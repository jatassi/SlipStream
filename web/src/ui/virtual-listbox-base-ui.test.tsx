// oxlint-disable-next-line no-restricted-imports -- this test pins what Base UI itself can and cannot do
import { Autocomplete } from "@base-ui/react/autocomplete";
import { useVirtualizer } from "@tanstack/react-virtual";
import { type KeyboardEvent, useEffect, useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";

const folders = Array.from({ length: 5000 }, (_, index) => `Folder ${index + 1}`);

function Window({ active }: { active: number }) {
  const [scroller, setScroller] = useState<HTMLDivElement | null>(null);
  // oxlint-disable-next-line react/incompatible-library -- Window is the leaf that holds the virtualizer
  const virtualizer = useVirtualizer({
    count: folders.length,
    getScrollElement: () => scroller,
    estimateSize: () => 44,
    overscan: 10,
  });
  useEffect(() => {
    if (active >= 0) {
      virtualizer.scrollToIndex(active);
    }
  }, [active, virtualizer]);
  return (
    <div ref={setScroller} className="h-[440px] overflow-y-auto">
      <Autocomplete.List className="relative" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => (
          <Autocomplete.Item
            key={item.key}
            index={item.index}
            value={folders[item.index]}
            aria-setsize={folders.length}
            aria-posinset={item.index + 1}
            className="absolute inset-x-0 top-0 h-11"
            style={{ transform: `translateY(${item.start}px)` }}
          >
            {folders[item.index]}
          </Autocomplete.Item>
        ))}
      </Autocomplete.List>
    </div>
  );
}

// The folder browser's reducer owns the active row and moves it by index for
// Page Up, Page Down, filtering and going up. Base UI 1.9's Autocomplete keeps
// its own highlight and has no prop that sets it, so the browser uses
// VirtualListbox's plain listbox instead. If this file starts failing, Base UI
// may have learned a controlled highlight worth revisiting.
function VirtualAutocomplete() {
  const [active, setActive] = useState(-1);
  return (
    <Autocomplete.Root
      open
      inline
      mode="none"
      virtualized
      filter={null}
      items={folders}
      onItemHighlighted={(_, details) => {
        setActive(details.index);
      }}
    >
      <Autocomplete.Input
        aria-label="Folders"
        onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => {
          if (event.key === "PageDown") {
            event.preventDefault();
            setActive((from) => from + 100);
          }
        }}
      />
      <output>{active}</output>
      <Window active={active} />
    </Autocomplete.Root>
  );
}

const field = () => page.getByRole("combobox", { name: "Folders" });
const described = () => {
  const id = field().element().getAttribute("aria-activedescendant");
  return id === null ? null : document.querySelector(`#${CSS.escape(id)}`)?.textContent;
};

test("Base UI's own arrows walk the highlight through a 5,000-row virtual window", async () => {
  renderKit(<VirtualAutocomplete />);
  await userEvent.click(field());
  for (let press = 0; press < 30; press += 1) {
    // oxlint-disable-next-line no-await-in-loop -- one key after another
    await userEvent.keyboard("{ArrowDown}");
  }
  await expect.element(page.getByRole("status")).toHaveTextContent("29");
  expect(described()).toBe("Folder 30");
});

test("a highlight set from outside never reaches Base UI's aria-activedescendant", async () => {
  renderKit(<VirtualAutocomplete />);
  await userEvent.click(field());
  await userEvent.keyboard("{ArrowDown}{PageDown}");
  await expect.element(page.getByRole("status")).toHaveTextContent("100");
  await expect.element(page.getByText("Folder 101", { exact: true })).toBeInTheDocument();
  expect(field().element().getAttribute("aria-activedescendant")).toMatch(/-0$/u);
  expect(described()).toBeUndefined();
});
