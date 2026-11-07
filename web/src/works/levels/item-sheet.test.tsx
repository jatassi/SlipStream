import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksWorkMonitored } from "../../api/gen/model";
import { anthologyId, longShelf } from "../../test/anthology";
import { SetWorkMonitoredBody } from "../../api/gen/zod/works/works.zod";
import { renderApp } from "../../test/render-app";
import { shelfCells } from "../../test/shelf-cells";
import { asPhone } from "../../test/viewport";
import { workServer } from "../../test/work-server";

async function openOnPhone() {
  await asPhone();
  const puts: WorksWorkMonitored[] = [];
  const monitored = http.put("*/api/v1/works/:workId/monitored", async ({ request }) => {
    puts.push(SetWorkMonitoredBody.parse(await request.json()));
    return new HttpResponse(null, { status: 204 });
  });
  const server = workServer({
    work: longShelf,
    startCells: shelfCells(longShelf, () => "missing"),
  });
  renderApp({ path: `/works/${anthologyId}`, handlers: [monitored, ...server.handlers] });
  return { puts };
}

const levels = () => page.getByRole("region", { name: "Volumes" });
const sheet = () => page.getByRole("dialog", { name: "V01S02 Story V01S02" });

test("on a phone each level is a card of rows, and a row opens a sheet with its cells and Monitor", async () => {
  const { puts } = await openOnPhone();
  await expect.element(levels().getByRole("switch", { name: "Monitor Volume 1" })).toBeVisible();
  expect(levels().getByRole("button", { name: "V01S02 1080p, Missing" }).elements()).toHaveLength(
    0,
  );

  await userEvent.click(levels().getByRole("button", { name: /^V01S02 ?Story V01S02/u }));
  await expect
    .element(sheet().getByRole("list", { name: "Cells" }))
    .toHaveTextContent("V01S02 1080p, Missing");
  await userEvent.click(sheet().getByRole("switch"));
  await expect
    .poll(() => puts)
    .toEqual([{ levelIds: [], itemIds: [longShelf.items[1]?.id], on: false }]);
  await expect.element(sheet().getByRole("switch")).not.toBeChecked();
});

test("a row's sheet leads to the Item's Details", async () => {
  await openOnPhone();
  await userEvent.click(levels().getByRole("button", { name: /^V01S02 ?Story V01S02/u }));
  await userEvent.click(sheet().getByRole("button", { name: "Details" }));
  await expect
    .element(page.getByRole("dialog", { name: "V01S02 Story V01S02" }))
    .toMatchTextContent(/Fixture source lists this story as v1s2\.$/u);
});
