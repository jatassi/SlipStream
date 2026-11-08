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
        <ListRow title="Shows" meta="12 Works" render={link("#shows")} />
        <ListRow title="Albums" render={link("#albums")} />
      </List>,
      { shell },
    );
    const list = page.getByRole("list", { name: "Content Types" });
    await expect.element(list.getByRole("listitem")).toHaveLength(3);
    const films = page.getByRole("link", { name: "Films" });
    const shows = page.getByRole("link", { name: "Shows 12 Works" });
    const albums = page.getByRole("link", { name: "Albums" });
    expect(getComputedStyle(films.element()).height).toBe(row);
    expect(getComputedStyle(shows.element()).height).toBe(twoLine);
    films.element().focus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(shows).toHaveFocus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(albums).toHaveFocus();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(albums).toHaveFocus();
    await userEvent.keyboard("{ArrowUp}");
    await expect.element(shows).toHaveFocus();
    await userEvent.keyboard("{Home}");
    await expect.element(films).toHaveFocus();
  });
});

describe.each(shells)("in the $shell shell", ({ shell }) => {
  test("a row's detail spans the row under its title and trailing slot", async () => {
    renderKit(
      <div style={{ width: "360px" }}>
        <List label="Files">
          <ListRow
            title="Lanterns at Low Tide (2016).mkv"
            meta="Lanterns at Low Tide (2016) · Fills 1080p"
            detail={<p>x265 · 1920×1080 · 10-bit · Dolby Vision profile 8, HDR10</p>}
            trailing="HDTV-1080p · 1920×1080 · 524 KB"
          />
        </List>
      </div>,
      { shell },
    );
    await expect.element(page.getByRole("listitem")).toBeVisible();
    const item = page.getByRole("listitem").element().getBoundingClientRect();
    const title = page
      .getByText("Lanterns at Low Tide (2016).mkv")
      .element()
      .getBoundingClientRect();
    const detail = page
      .getByText("Dolby Vision", { exact: false })
      .element()
      .getBoundingClientRect();
    const trailing = page
      .getByText("HDTV-1080p", { exact: false })
      .element()
      .getBoundingClientRect();
    expect(detail.left).toBe(title.left);
    expect(detail.right).toBe(trailing.right);
    expect(detail.width).toBe(item.width - 32);
    expect(trailing.bottom).toBeLessThanOrEqual(detail.top);
  });
});
