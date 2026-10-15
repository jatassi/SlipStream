import type { HTMLAttributes } from "react";
import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { List, ListRow } from "./list";
import type { ShellKind } from "./shell-context";

function link(href: string) {
  return ({ children, ...props }: HTMLAttributes<HTMLElement>) => (
    <a {...props} href={href}>
      {children}
    </a>
  );
}

const shells = [
  { shell: "desktop", row: "44px", twoLine: "60px" },
  { shell: "phone", row: "54px", twoLine: "64px" },
] as const satisfies readonly { shell: ShellKind; row: string; twoLine: string }[];

describe.each(shells)("in the $shell shell", ({ shell, row, twoLine }) => {
  test("arrow keys move between the rows of an interactive list", async () => {
    renderKit(
      <List label="Content Types">
        <ListRow title="Films" render={link("#films")} />
        <ListRow title="Series" meta="12 Works" render={link("#series")} />
        <ListRow title="Albums" render={link("#albums")} />
      </List>,
      { shell },
    );
    const list = page.getByRole("list", { name: "Content Types" });
    await expect.element(list.getByRole("listitem")).toHaveLength(3);
    const films = page.getByRole("link", { name: "Films" });
    const series = page.getByRole("link", { name: "Series 12 Works" });
    const albums = page.getByRole("link", { name: "Albums" });
    expect(getComputedStyle(films.element()).height).toBe(row);
    expect(getComputedStyle(series.element()).height).toBe(twoLine);
    films.element().focus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(series).toHaveFocus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(albums).toHaveFocus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(albums).toHaveFocus();
    await userEvent.keyboard("{ArrowUp}");
    await expect.element(series).toHaveFocus();
    await userEvent.keyboard("{Home}");
    await expect.element(films).toHaveFocus();
  });
});
