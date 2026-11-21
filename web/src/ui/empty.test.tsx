import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Button } from "./button";
import { type EmptyLook, Empty } from "./empty";
import { List, ListRow } from "./list";
import type { ShellKind } from "./shell-context";

test.each(["sheet", "row", "inline"] satisfies EmptyLook[])(
  "a %s empty shows its sentence and its way forward",
  async (look) => {
    renderKit(
      <Empty look={look} action={<Button>Clear filters</Button>}>
        No entries match these filters.
      </Empty>,
    );
    await expect.element(page.getByText("No entries match these filters.")).toBeVisible();
    await expect.element(page.getByRole("button", { name: "Clear filters" })).toBeVisible();
  },
);

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a sheet empty is as wide as the list it stands in for",
  async (shell) => {
    renderKit(
      <div className="flex w-[1200px] flex-col gap-4">
        <Empty look="sheet">Nothing is downloading.</Empty>
        <List label="Torrents">
          <ListRow title="Orchard.Harvest.2019.720p.HDTV" />
        </List>
      </div>,
      { shell },
    );
    const list = page.getByRole("list", { name: "Torrents" });
    await expect.element(list).toBeVisible();
    const empty = page.getByText("Nothing is downloading.").element().parentElement;
    expect(empty?.getBoundingClientRect().width).toBe(list.element().getBoundingClientRect().width);
  },
);
