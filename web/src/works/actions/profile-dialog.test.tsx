import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { picked, renderAction } from "../../test/actions";
import { changeOp, noImpact } from "../../test/impact";
import { cinema } from "../../test/library";
import { films, profileHandlers, ultra } from "../../test/profiles";
import type { Subject } from "./actions";
import { ProfileDialog } from "./profile-dialog";

function Opened({ subjects }: { subjects: readonly Subject[] }) {
  const [open, setOpen] = useState(true);
  return (
    <ProfileDialog
      type={cinema}
      subjects={subjects}
      profileId={films.id}
      open={open}
      onOpenChange={setOpen}
    />
  );
}

test("many Works change Profile through one Impact across all of them", async () => {
  const works = picked(4);
  const op = changeOp(
    "put",
    "*/api/v1/works/profile",
    () => ({ ...noImpact, works: 4, unwanted: { files: 2, bytes: 1_468_006_400 }, newlyWanted: 3 }),
    { outcomes: works.map(({ id }) => ({ workId: id, outcome: "done" })) },
  );
  await renderAction(<Opened subjects={works} />, [...profileHandlers(), op.handler]);

  const dialog = page.getByRole("dialog", { name: "Change the Profile of 4 Films" });
  await expect
    .element(dialog.getByRole("combobox", { name: "Profile" }))
    .toHaveTextContent("Films (default)");
  await userEvent.click(dialog.getByRole("combobox", { name: "Profile" }));
  await userEvent.click(page.getByRole("option", { name: "Ultra" }));
  await userEvent.click(dialog.getByRole("button", { name: "Next" }));

  const impact = page.getByRole("dialog", { name: "Give 4 Films the Profile Ultra" });
  await expect
    .element(impact.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("4 Works change.2 files would fill nothing (1.4 GB).3 cells become wanted.");
  await userEvent.click(impact.getByRole("button", { name: "Save and delete 2 files" }));

  await expect.element(page.getByText("Changed the Profile of 4 Films.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  const body = { workIds: works.map(({ id }) => id), profileId: ultra.id };
  expect(op.sent).toEqual([
    { query: { preview: "true" }, body },
    { query: { unwanted: "delete", confirm: "2" }, body },
  ]);
});
