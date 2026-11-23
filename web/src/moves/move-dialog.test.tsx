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

test("the Move Dialog leads with the recommendation and groups each film by verdict", async () => {
  const sent: Sent = { previews: [], starts: [] };
  await renderAction(<Opened />, moveServer([threeFilms], [started()], sent));

  const dialog = page.getByRole("dialog", { name: "Move 3 films" });
  await expect
    .element(dialog.getByRole("combobox", { name: "Move to" }))
    .toHaveTextContent("/mnt/archive/films on home · Free space not read yet");
  await expect
    .element(dialog.getByText("Move 2 of 3 films to home. 1 can't move: Film 3."))
    .toBeVisible();
  const red = dialog.getByRole("region", { name: "Can't move" });
  await expect
    .element(red.getByRole("listitem"))
    .toHaveTextContent(
      "Film 3 /mnt/archive on home would go below its Error level. Free space there or pick another root.",
    );
  await expect
    .element(dialog.getByRole("region", { name: "Moves, with a note" }).getByRole("listitem"))
    .toHaveTextContent("Film 2 /mnt/archive on home would go below its Warning level.");
  await expect
    .element(dialog.getByRole("region", { name: "Moves", exact: true }).getByRole("listitem"))
    .toHaveTextContent("The Quiet Orchard");
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
  await expect
    .element(dialog.getByText("Move 2 of 3 films to home. 1 can't move: Film 3."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("checkbox", { name: "Also move their torrents" }));
  await expect.poll(() => sent.previews.length).toBe(2);
  expect(sent.previews[1]).toMatchObject({ options: { includeLinkedTorrents: false } });
});

test("a film's linked torrent doesn't count as a film it started", async () => {
  const sent: Sent = { previews: [], starts: [] };
  const [orchard] = threeFilms.units;
  if (orchard === undefined) {
    throw new Error("threeFilms has no unit");
  }
  const withTorrent: MovesMovePreview = {
    ...threeFilms,
    units: [
      { ...orchard, key: "torrent:1", kind: "torrent", name: "Orchard.mkv" },
      ...threeFilms.units,
    ],
  };
  await renderAction(<Opened />, moveServer([withTorrent], [started({ started: 3 })], sent));

  await userEvent.click(page.getByRole("button", { name: "Move 2 of 3" }));
  await expect.element(page.getByText("Moving 2 films to home.")).toBeVisible();
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
  await expect
    .element(page.getByText("Film 2 stays where it is. It's already moving in another Move."))
    .toBeVisible();
  await expect.element(page.getByText("Moving 1 film to home.")).toBeVisible();
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
  await expect
    .element(
      dialog.getByText("The Move changed since its preview. Check the new preview, then start it."),
    )
    .toBeVisible();
  await expect
    .element(dialog.getByText("Move 1 of 3 films to home. 2 can't move: Film 2 and Film 3."))
    .toBeVisible();

  await userEvent.click(dialog.getByRole("button", { name: "Move 1 of 3" }));
  await expect.element(page.getByText("Moving 1 film to home.")).toBeVisible();
  expect(sent.starts).toEqual([
    { confirm: "2", body: { token: "plan-1" } },
    { confirm: "1", body: { token: "plan-2" } },
  ]);
});
