import { HttpResponse, http } from "msw";
import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../../api/gen/query/roots/roots.msw";
import { film, renderAction } from "../../test/actions";
import { cinema } from "../../test/library";
import { archiveRoot, filmsRoot, mediaVolumes } from "../../test/roots";
import { type Rooted, RootDialog } from "./root-dialog";

function rootServer(works: readonly Rooted[], sent: unknown[]) {
  return [
    getListRootsMockHandler({ roots: [filmsRoot, archiveRoot] }),
    getListVolumesMockHandler({ nodes: [mediaVolumes] }),
    http.put("*/api/v1/works/root", async ({ request }) => {
      sent.push(await request.json());
      return HttpResponse.json({
        outcomes: works.map(({ id, files }) =>
          files === 0
            ? { workId: id, outcome: "done" }
            : { workId: id, outcome: "skipped", reason: "work_has_files" },
        ),
      });
    }),
  ];
}

function Opened({ works }: { works: readonly Rooted[] }) {
  const [open, setOpen] = useState(true);
  return <RootDialog type={cinema} works={works} open={open} onOpenChange={setOpen} />;
}

test("the root dialog shows free space and names the Works with files that stay", async () => {
  const works: Rooted[] = [
    { ...film(1), files: 0 },
    { ...film(2), files: 1 },
    { ...film(3), files: 2 },
  ];
  const sent: unknown[] = [];
  await renderAction(<Opened works={works} />, rootServer(works, sent));

  const dialog = page.getByRole("dialog", {
    name: "Set the library root of The Quiet Orchard, Film 2, and Film 3",
  });
  const picker = dialog.getByRole("combobox", { name: "Library root" });
  await expect.element(picker).toHaveTextContent("/srv/media/films · 1.2 TB free");
  await userEvent.click(picker);
  await expect
    .element(page.getByRole("option", { name: "/mnt/archive/films · Free space not read yet" }))
    .toBeVisible();
  await userEvent.click(
    page.getByRole("option", { name: "/mnt/archive/films · Free space not read yet" }),
  );
  await expect
    .element(
      dialog.getByText(
        "Film 2 and Film 3 have files, so they stay in their library roots. Moving a Work that has files isn't supported yet.",
      ),
    )
    .toBeVisible();

  await userEvent.click(dialog.getByRole("button", { name: "Set library root" }));
  await expect
    .element(
      page.getByText(
        "Set the library root of The Quiet Orchard. Film 2 and Film 3 have files, so they stay in their library roots. Moving a Work that has files isn't supported yet.",
      ),
    )
    .toBeVisible();
  expect(sent).toEqual([{ workIds: works.map(({ id }) => id), rootId: archiveRoot.id }]);
});
