import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { FilesCellList, FilesNeed, GrabsGrab } from "../../api/gen/model";
import { getGetWorkCellsMockHandler } from "../../api/gen/query/files/files.msw";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { clock } from "../../format";
import { orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { awayGrab as stuck, upgradable } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

function openWith(needsYou: FilesNeed[], grabs: GrabsGrab[]) {
  const cells: FilesCellList = { ...upgradable, needsYou };
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      getGetWorkCellsMockHandler(cells),
      getListGrabsMockHandler({ grabs }),
      ...workServer().handlers,
    ],
  });
  return page.getByRole("region", { name: "Needs you" });
}

test("a download whose client reports an error quotes it and says when the Grab fails", async () => {
  const erroring = {
    ...stuck,
    torrentId: "4b1e7c2a-9d3f-4a6b-8e5c-2f7a9b1d3e6c",
  } as const;
  const failsAt = "2026-11-12T16:10:00Z";
  const needsYou = openWith(
    [
      {
        kind: "waiting",
        grabId: erroring.id,
        code: "client_error",
        clientError: {
          client: "Transmission",
          node: "Seedbox",
          message: "No space left on device",
          failsAt,
        },
      },
    ],
    [erroring],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      `Needs you1Waiting${erroring.releaseTitle}Transmission on Seedbox reports "No space left on device." Fix it in Transmission. SlipStream fails this Grab if the error lasts past ${clock(failsAt)}.Open torrent`,
    );
});

test("a download whose Import waits for space sends the Admin to Pending imports", async () => {
  const needsYou = openWith(
    [{ kind: "waiting", grabId: stuck.id, code: "no_space", clientError: null }],
    [stuck],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      `Needs you1Waiting${stuck.releaseTitle}Its library root has too little free space for a file of this download. Free up space there. SlipStream retries automatically once there's room.Open Pending imports`,
    );
});
