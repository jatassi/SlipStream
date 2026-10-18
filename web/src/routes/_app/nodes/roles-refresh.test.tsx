import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { NodesBoardNode } from "../../../api/gen/model";
import { getGetRoleBoardQueryKey } from "../../../api/gen/query/nodes/nodes";
import { refresh } from "../../../live/refresh";
import { home, nodesAre, seedbox } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { board, boardIs, homeSeat, seedboxSeat } from "../../../test/roles";
import { worker } from "../../../test/worker";

test("a Role Group opens once a refresh ticks one of its Roles", async () => {
  const bare: NodesBoardNode = { ...seedboxSeat, assigned: [], roles: [] };
  const app = renderApp({
    path: `/nodes/${seedboxSeat.id}/roles`,
    handlers: [nodesAre(home, seedbox), boardIs(board(homeSeat, bare))],
  });
  const acquire = page.getByRole("button", { name: "Acquire", exact: true });
  await expect.element(acquire).toHaveAttribute("aria-expanded", "false");

  worker.use(boardIs(board(homeSeat, seedboxSeat)));
  await refresh(app.queryClient, { queryKey: getGetRoleBoardQueryKey() });
  await expect
    .element(page.getByRole("checkbox", { name: "Downloading", exact: true }))
    .toBeChecked();
});
