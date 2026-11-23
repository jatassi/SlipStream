import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MovesMoveAsk } from "../api/gen/model";
import { act, openRoots } from "../test/roots-page";
import { archiveRoot, filmsRoot, serveRoots } from "../test/roots";

test("Move root… leads with editing the root, and Move its films previews every film to another root", async () => {
  const previews: MovesMoveAsk[] = [];
  serveRoots({ roots: [filmsRoot, { ...archiveRoot, state: "ready" }] });
  openRoots({
    handlers: [
      http.post("*/api/v1/moves/preview", async ({ request }) => {
        // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client sends the body it types
        previews.push((await request.json()) as MovesMoveAsk);
        return HttpResponse.json(
          { status: 500, code: "unexpected", title: "Held" },
          { status: 500 },
        );
      }),
    ],
  });
  await act(filmsRoot, "Move root…");
  const chooser = page.getByRole("dialog", { name: `Move ${filmsRoot.path}` });
  await expect
    .element(
      chooser.getByText(
        "If you moved this disk to another machine, edit the root. Nothing is copied. To copy its films to another root, move them.",
      ),
    )
    .toBeVisible();
  await userEvent.click(chooser.getByRole("button", { name: "Move its films" }));
  const moving = page.getByRole("dialog", { name: `Move the films in ${filmsRoot.path}` });
  await expect
    .element(moving.getByRole("checkbox", { name: "Delete this root when it's empty" }))
    .not.toBeChecked();
  await expect
    .poll(() => previews)
    .toEqual([
      {
        origin: "root",
        works: [],
        roots: [{ rootId: filmsRoot.id, to: archiveRoot.id }],
        options: { deleteEmptiedRoots: false },
      },
    ]);
});

test("Edit the root opens the root's own dialog", async () => {
  serveRoots();
  openRoots();
  await act(filmsRoot, "Move root…");
  await userEvent.click(page.getByRole("button", { name: "Edit the root" }));
  await expect.element(page.getByRole("dialog", { name: "Edit the Films root" })).toBeVisible();
});
