import { HttpResponse, http } from "msw";
import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksDeleted, WorksDeleting } from "../../api/gen/model";
import { picked, renderAction } from "../../test/actions";
import type { Subject } from "./actions";
import { DeleteDialog } from "./delete-dialog";

function Opened({ subjects }: { subjects: readonly Subject[] }) {
  const [open, setOpen] = useState(true);
  return <DeleteDialog subjects={subjects} open={open} onOpenChange={setOpen} />;
}

const offline =
  "seedbox is offline, so its files can't be deleted. Delete it and leave its files, or try again once seedbox is back.";

// The Quiet Orchard's files sit on seedbox, which is offline, so only leaving them deletes it.
function deleteServer(subjects: readonly Subject[]) {
  const sent: WorksDeleting[] = [];
  const [orchard] = subjects;
  const handler = http.post("*/api/v1/works/delete", async ({ request }) => {
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the test's own request body
    const body = (await request.json()) as WorksDeleting;
    sent.push(body);
    const stuck = body.files === "delete";
    const answer: WorksDeleted = {
      plan: {
        works: stuck ? 2 : 3,
        files: 4,
        kept: 1,
        bytes: 3 * 734_003_200,
        downloading: 1,
      },
      outcomes: subjects.map(({ id }) =>
        stuck && id === orchard?.id
          ? {
              workId: id,
              outcome: "failed",
              problem: {
                code: "storage_unreachable",
                title: "A Node holding its files is offline",
                detail: offline,
              },
            }
          : { workId: id, outcome: "done" },
      ),
    };
    return HttpResponse.json(answer);
  });
  return { sent, handler };
}

test("the delete dialog deletes files by default, says what it takes and what keeps seeding", async () => {
  const works = picked(3);
  const server = deleteServer(works);
  await renderAction(<Opened subjects={works} />, [server.handler]);

  const dialog = page.getByRole("dialog", {
    name: "Delete The Quiet Orchard, Film 2, and Film 3?",
  });
  await expect.element(dialog.getByRole("radio", { name: "Delete files from disk" })).toBeChecked();
  await expect
    .element(
      dialog.getByText("Their 4 files (2.1 GB), including 1 Kept file, are deleted from disk."),
    )
    .toBeVisible();
  await expect
    .element(
      dialog.getByText(
        "1 download in progress keeps downloading, and torrents keep seeding in the torrent client.",
      ),
    )
    .toBeVisible();
  await expect
    .element(dialog.getByText(`The Quiet Orchard can't be deleted with its files. ${offline}`))
    .toBeVisible();
  expect(server.sent).toEqual([
    { workIds: works.map(({ id }) => id), files: "delete", preview: true },
  ]);
});

test("a Work on an offline Node is deleted by leaving its files", async () => {
  const works = picked(3);
  const server = deleteServer(works);
  await renderAction(<Opened subjects={works} />, [server.handler]);

  const dialog = page.getByRole("dialog", {
    name: "Delete The Quiet Orchard, Film 2, and Film 3?",
  });
  await userEvent.click(dialog.getByRole("button", { name: "Leave files" }));
  await expect.element(dialog.getByRole("radio", { name: "Leave files on disk" })).toBeChecked();
  await expect
    .element(dialog.getByText("Their 4 files (2.1 GB), including 1 Kept file, stay on disk."))
    .toBeVisible();
  await expect.element(dialog.getByText("The next library scan finds them again.")).toBeVisible();
  await expect
    .element(dialog.getByText(/can't be deleted with its files/u))
    .not.toBeInTheDocument();

  await userEvent.click(dialog.getByRole("button", { name: "Delete", exact: true }));
  await expect
    .element(
      page.getByText("Deleted The Quiet Orchard, Film 2, and Film 3. Their files stay on disk."),
    )
    .toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.sent.at(-1)).toEqual({ workIds: works.map(({ id }) => id), files: "leave" });
});
