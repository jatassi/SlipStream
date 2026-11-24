import type { HTMLAttributes } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { List, ListRow } from "./list";

function link(href: string) {
  return ({ children, ...props }: HTMLAttributes<HTMLElement>) => (
    <a {...props} href={href}>
      {children}
    </a>
  );
}

test("arrow keys move between the rows of an interactive list", async () => {
  renderKit(
    <List label="Content Types">
      <ListRow title="Films" render={link("#films")} />
      <ListRow title="Shows" meta="12 Works" render={link("#shows")} />
      <ListRow title="Albums" render={link("#albums")} />
    </List>,
  );
  const list = page.getByRole("list", { name: "Content Types" });
  await expect.element(list.getByRole("listitem")).toHaveLength(3);
  const films = page.getByRole("link", { name: "Films" });
  const shows = page.getByRole("link", { name: "Shows 12 Works" });
  const albums = page.getByRole("link", { name: "Albums" });
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
