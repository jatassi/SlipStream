import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListRootsMockHandler } from "../api/gen/query/roots/roots.msw";
import { CheckFolderBody } from "../api/gen/zod/roots/roots.zod";
import { brazilRow, folderEntries } from "../test/import-entries";
import { serveFolders, unixFolder } from "../test/folders";
import { homeId } from "../test/nodes";
import { openPending } from "../test/pending-page";
import { asPhone } from "../test/viewport";

const sheet = () => page.getByRole("dialog", { name: "Import files" });
const [brazilImport, heatImport] = folderEntries;

const verdicts = [
  getListRootsMockHandler({ roots: [] }),
  http.post("*/api/v1/nodes/:id/folders/check", async ({ request }) => {
    const { path } = CheckFolderBody.parse(await request.json());
    return HttpResponse.json({ usable: true, path });
  }),
];

test("Import files opens a folder in the Node's download folders and lists what each file would fill", async () => {
  const { sent } = openPending({ handlers: verdicts });
  serveFolders({
    [homeId]: {
      places: [{ name: "Films", path: "/downloads/Films", free: 0, total: 0 }],
      folders: { "/downloads/Films": unixFolder("/downloads/Films") },
    },
  });
  await userEvent.click(page.getByRole("button", { name: "Import files" }));
  await userEvent.click(sheet().getByRole("button", { name: "Choose a folder" }));
  const browser = page.getByRole("dialog", { name: "Choose a folder on home" });
  await userEvent.click(browser.getByRole("option", { name: /Films/u }));
  await userEvent.click(browser.getByRole("button", { name: /^Open/u }));
  await expect.element(sheet().getByText("Fills 1080p for Brazil (1985).")).toBeVisible();
  expect(sent.opened).toEqual([{ nodeId: homeId, folder: "/downloads/Films" }]);
  await expect
    .element(
      sheet().getByText("This doesn't meet any Target this Work wants. 1080p needs Bluray or WEB."),
    )
    .toBeVisible();
  await expect.element(sheet().getByRole("heading", { name: "Heat" })).toBeVisible();
});

test("Import all confirms each file that would fill something, and Move files applies to the batch", async () => {
  const { sent } = openPending({ start: { imports: [...folderEntries] } });
  await userEvent.click(
    page.getByRole("listitem", { name: "/downloads/Films" }).getByRole("button", {
      name: "Continue",
    }),
  );
  await userEvent.click(sheet().getByRole("switch", { name: "Move files" }));
  await expect
    .poll(() => sent.moved)
    .toEqual([{ importIds: [brazilImport?.id, heatImport?.id], on: true }]);
  await expect.element(sheet().getByRole("switch", { name: "Move files" })).toBeChecked();
  await userEvent.click(sheet().getByRole("button", { name: "Import all" }));
  await expect.element(page.getByText("Importing 1 file.")).toBeVisible();
  expect(sent.confirmed).toEqual([[brazilImport?.id, heatImport?.id]]);
  await expect.element(sheet()).not.toBeInTheDocument();
});

test("Change the match sets a file's Work, Items and release name", async () => {
  const { sent } = openPending({ start: { imports: [...folderEntries] } });
  await userEvent.click(
    page.getByRole("listitem", { name: "/downloads/Films" }).getByRole("button", {
      name: "Continue",
    }),
  );
  await userEvent.click(sheet().getByRole("button", { name: "Other actions for Heat (1995).mkv" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Change the match" }));
  const match = page.getByRole("dialog", { name: "Change the match for Heat (1995).mkv" });
  await userEvent.click(match.getByRole("combobox", { name: "Film" }));
  await userEvent.click(page.getByRole("option", { name: "Brazil" }));
  const release = match.getByRole("textbox", { name: "Release name" });
  await expect.element(release).toHaveValue("Heat (1995)");
  await userEvent.fill(release, "Brazil.1985.1080p.BluRay-GRP");
  await userEvent.click(match.getByRole("button", { name: "Save the match" }));
  await expect.element(match).not.toBeInTheDocument();
  expect(sent.fileMatches).toEqual([
    {
      action: "match",
      workId: brazilRow.id,
      itemIds: ["0199f0d4-0000-7000-8000-000000001985"],
      edition: null,
      releaseName: "Brazil.1985.1080p.BluRay-GRP",
    },
  ]);
});

test("on a phone the sheet's rows stack and Continue still opens them", async () => {
  await asPhone();
  openPending({ start: { imports: [...folderEntries] } });
  await userEvent.click(
    page.getByRole("listitem", { name: "/downloads/Films" }).getByRole("button", {
      name: "Continue",
    }),
  );
  await expect.element(page.getByText("Fills 1080p for Brazil (1985).")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Import all" })).toBeEnabled();
});
