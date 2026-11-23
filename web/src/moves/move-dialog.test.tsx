import { HttpResponse, http } from "msw";
import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MovesMovePreview, MovesStartedMoves } from "../api/gen/model";
import { film, renderAction } from "../test/actions";
import { cinema } from "../test/library";
import { filmRoots, threeFilms } from "../test/moves";
import { archiveRoot, filmsRoot, refused } from "../test/roots";
import { MoveDialog } from "./move-dialog";

type Sent = { previews: unknown[]; starts: { confirm: string | null; body: unknown }[] };

function moveServer(
  previews: readonly MovesMovePreview[],
  starts: readonly (MovesStartedMoves | "changed")[],
  sent: Sent,
) {
  return [
    ...filmRoots,
    http.post("*/api/v1/moves/preview", async ({ request }) => {
      sent.previews.push(await request.json());
      return HttpResponse.json(previews[Math.min(sent.previews.length, previews.length) - 1]);
    }),
    http.post("*/api/v1/moves", async ({ request }) => {
      sent.starts.push({
        confirm: new URL(request.url).searchParams.get("confirm"),
        body: await request.json(),
      });
      const answer = starts[sent.starts.length - 1];
      return answer === "changed"
        ? refused(
            409,
            "move_plan_changed",
            "The Move changed since its preview. Open the preview again.",
          )
        : HttpResponse.json(answer, { status: 202 });
    }),
  ];
}

function Opened() {
  const [open, setOpen] = useState(true);
  return (
    <MoveDialog
      type={cinema}
      works={[film(1), film(2), film(3)]}
      from={filmsRoot.id}
      origin="bulk"
      open={open}
      onOpenChange={setOpen}
    />
  );
}

const started = (over: Partial<MovesStartedMoves> = {}): MovesStartedMoves => ({
  batchId: "0199f0b1-1111-7000-8000-000000000001",
  started: 2,
  dropped: [],
  ...over,
});

test("the Move Dialog groups each film by verdict", async () => {
  const sent: Sent = { previews: [], starts: [] };
  await renderAction(<Opened />, moveServer([threeFilms], [started()], sent));

  const dialog = page.getByRole("dialog", { name: "Move 3 films" });
  const to = dialog.getByRole("combobox", { name: "Move to" });
  await expect.poll(() => to.element().textContent).toContain(archiveRoot.path);
  const listed = (region: string) =>
    dialog.getByRole("region", { name: region, exact: true }).getByRole("listitem");
  await expect.element(listed("Can't move").getByText("Film 3")).toBeVisible();
  await expect.element(listed("Moves, with a note").getByText("Film 2")).toBeVisible();
  await expect.element(listed("Moves").getByText("The Quiet Orchard")).toBeVisible();
  expect(sent.previews).toEqual([
    {
      origin: "bulk",
      works: [film(1), film(2), film(3)].map(({ id }) => ({ workId: id, rootId: archiveRoot.id })),
      options: { includeLinkedTorrents: true },
    },
  ]);
});

test("unticking Also move their torrents previews the films alone", async () => {
  const sent: Sent = { previews: [], starts: [] };
  await renderAction(<Opened />, moveServer([threeFilms], [started()], sent));

  const dialog = page.getByRole("dialog", { name: "Move 3 films" });
  await expect.element(dialog.getByRole("button", { name: "Move 2 of 3" })).toBeVisible();
  await userEvent.click(dialog.getByRole("checkbox", { name: "Also move their torrents" }));
  await expect.poll(() => sent.previews.length).toBe(2);
  expect(sent.previews[1]).toMatchObject({ options: { includeLinkedTorrents: false } });
});

test("Start confirms the count the preview showed and toasts each film it left out", async () => {
  const sent: Sent = { previews: [], starts: [] };
  await renderAction(
    <Opened />,
    moveServer(
      [threeFilms],
      [
        started({
          started: 1,
          dropped: [
            { workId: film(2).id, name: "Film 2", words: "It's already moving in another Move." },
          ],
        }),
      ],
      sent,
    ),
  );

  await userEvent.click(page.getByRole("button", { name: "Move 2 of 3" }));
  await expect.element(page.getByText(/^Film 2 /u)).toBeVisible();
  await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  expect(sent.starts).toEqual([{ confirm: "2", body: { token: "plan-1" } }]);
});

test("a plan that changed since its preview shows the new preview in place and starts it next", async () => {
  const sent: Sent = { previews: [], starts: [] };
  const fewer: MovesMovePreview = {
    ...threeFilms,
    token: "plan-2",
    recommendation: "Move 1 of 3 films to home. 2 can't move: Film 2 and Film 3.",
    button: "Move 1 of 3",
    movable: 1,
    units: threeFilms.units.map((unit) =>
      unit.verdict === "amber" ? { ...unit, verdict: "red" } : unit,
    ),
  };
  await renderAction(
    <Opened />,
    moveServer([threeFilms, fewer], ["changed", started({ started: 1 })], sent),
  );

  const dialog = page.getByRole("dialog", { name: "Move 3 films" });
  await userEvent.click(dialog.getByRole("button", { name: "Move 2 of 3" }));
  await userEvent.click(dialog.getByRole("button", { name: "Move 1 of 3" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent.starts).toEqual([
    { confirm: "2", body: { token: "plan-1" } },
    { confirm: "1", body: { token: "plan-2" } },
  ]);
});
