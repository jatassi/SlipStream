import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { FilesCellList, FilesNeed, GrabsGrab } from "../../api/gen/model";
import { getGetWorkCellsMockHandler } from "../../api/gen/query/files/files.msw";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { getListRootsMockHandler } from "../../api/gen/query/roots/roots.msw";
import { clock } from "../../format";
import { orchardId } from "../../test/library";
import { home } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
// oxlint-disable-next-line import/max-dependencies -- an Import wait names its root, so the test serves roots on a served Node
import { filmsRoot } from "../../test/roots";
import { awayGrab as stuck, upgradable } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

function openWith(needsYou: FilesNeed[], grabs: GrabsGrab[]) {
  const cells: FilesCellList = { ...upgradable, needsYou };
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      getGetWorkCellsMockHandler(cells),
      getListGrabsMockHandler({ grabs }),
      getListRootsMockHandler({ roots: [{ ...filmsRoot, nodeId: home.id }] }),
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
        importWait: null,
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

test("a download whose Import waits for space names the root, its free space and the file's size", async () => {
  const needsYou = openWith(
    [
      {
        kind: "waiting",
        grabId: stuck.id,
        code: "no_space",
        clientError: null,
        importWait: {
          rootId: filmsRoot.id,
          space: { free: 12 * 1024 ** 3, needs: 48 * 1024 ** 3 },
        },
      },
    ],
    [stuck],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      `Needs you1Waiting${stuck.releaseTitle}The Films library root on home has 12 GB free above its error level, and this file needs 48 GB. Free up space there. SlipStream retries automatically once there's room.Open Pending imports`,
    );
});

test("a download whose Import waits on its root says the root isn't ready", async () => {
  const needsYou = openWith(
    [
      {
        kind: "waiting",
        grabId: stuck.id,
        code: "root_not_ready",
        clientError: null,
        importWait: { rootId: filmsRoot.id, space: null },
      },
    ],
    [stuck],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      `Needs you1Waiting${stuck.releaseTitle}The Films library root on home isn't ready. SlipStream retries once it is.Open Pending imports`,
    );
});
