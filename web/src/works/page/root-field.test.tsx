import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderAction } from "../../test/actions";
import { cinema } from "../../test/library";
import { filmRoots } from "../../test/moves";
import { filmsRoot } from "../../test/roots";
import { RootField } from "./root-field";

function Field({ files, onMove }: { files: number; onMove: () => void }) {
  const [root, setRoot] = useState(filmsRoot.id);
  return (
    <RootField type={cinema} value={root} files={files} onValueChange={setRoot} onMove={onMove} />
  );
}

test("a film with files keeps its root field fixed and offers Move to…", async () => {
  let moves = 0;
  await renderAction(
    <Field
      files={2}
      onMove={() => {
        moves += 1;
      }}
    />,
    filmRoots,
  );

  await expect.element(page.getByRole("combobox", { name: "Library root" })).toBeDisabled();
  await userEvent.click(page.getByRole("button", { name: "Move to…" }));
  expect(moves).toBe(1);
  expect(page.getByText(/can't move files between library roots/u).elements()).toEqual([]);
});

test("a film without files picks its root and offers no Move", async () => {
  await renderAction(<Field files={0} onMove={() => {}} />, filmRoots);

  await expect.element(page.getByRole("combobox", { name: "Library root" })).toBeEnabled();
  await expect.element(page.getByRole("button", { name: "Move to…" })).not.toBeInTheDocument();
});
