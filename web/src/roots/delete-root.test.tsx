import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { act, openRoots } from "../test/roots-page";
import { archiveRoot, filmsRoot, serveRoots } from "../test/roots";

test("the default root can't be deleted while its type has another root", async () => {
  const { sent } = serveRoots({ roots: [filmsRoot, archiveRoot] });
  openRoots();
  await act(filmsRoot, "Delete");
  const dialog = page.getByRole("dialog", { name: `Delete ${filmsRoot.path}?` });
  await expect
    .element(
      dialog.getByText(
        "New films go into this root. Make another Films root the default first, then delete this one.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog.getByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  await userEvent.click(dialog.getByRole("button", { name: "Close" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.deleted).toEqual([]);
});
