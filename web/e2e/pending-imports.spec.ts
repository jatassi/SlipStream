import { copyFile, mkdir } from "node:fs/promises";
import { join } from "node:path";

import type { Locator, Page, TestInfo } from "@playwright/test";
import { z } from "zod";

import { type World, expect, expectAccessible, test } from "./fixtures.ts";
import { browser, drillInto, menuItem } from "./folder-browser.ts";
import { isPhone, navigate } from "./shell.ts";
import { screenshotWholePage } from "./whole-page.ts";

test.use({ scenario: "imports" });

const orchard = "The Quiet Orchard";
const release = "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW";
const video = `${release}.mkv`;
const kept = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv";
const harbor = "Harbor.Lights.2019.1080p.WEB-DL.H264-FKW.mkv";
const notBetter = "This isn't better than the file you have for 1080p.";

const Library = z.object({
  rows: z.array(z.object({ id: z.string(), label: z.string(), files: z.number() })),
});
const Added = z.object({ work: z.object({ id: z.string() }) });
const Waiting = z.object({ imports: z.array(z.object({ id: z.string() })) });
const Files = z.object({ files: z.array(z.object({ path: z.string(), quality: z.string() })) });

async function libraryRow(world: World, label: string) {
  const { rows } = Library.parse((await world.api("GET", "/api/v1/library")).body);
  return rows.find((row) => row.label === label);
}

// Each of the Work's files as "name quality".
async function filesOf(world: World, workId: string): Promise<string[]> {
  const { files } = Files.parse((await world.api("GET", `/api/v1/works/${workId}/files`)).body);
  return files.map(({ path, quality }) => `${path.split(/[/\\]/u).at(-1)} ${quality}`);
}

async function orchardId(world: World): Promise<string> {
  const row = await libraryRow(world, orchard);
  if (row === undefined) {
    throw new Error(`the scan added no ${orchard}`);
  }
  return row.id;
}

// The Quiet Orchard searched the way a script would: its 1080p BluRay
// release is grabbed, reads as 720p, and waits as no upgrade.
async function waitingDownload(world: World): Promise<string> {
  const search = await world.api("POST", `/api/v1/works/${await orchardId(world)}/search`);
  expect(search.status, "its search starts").toBe(202);
  let ids: string[] = [];
  await expect
    .poll(
      async () => {
        ids = Waiting.parse(
          (await world.api("GET", "/api/v1/imports?waiting=true")).body,
        ).imports.map(({ id }) => id);
        return ids.length;
      },
      { message: "the download waits for the Admin", timeout: 120_000 },
    )
    .toBe(1);
  const [id] = ids;
  if (id === undefined) {
    throw new Error("no Import waits");
  }
  return id;
}

async function openPendingImports(page: Page): Promise<void> {
  await navigate(page, "Library");
  await page.getByRole("link", { name: /^1 pending import needs you/u }).click();
  await expect(page).toHaveURL(/\/library\/pending$/u);
}

// Harbor Lights added, and the shelf's video of it dropped into home's
// download folder the way an Admin copies a file there.
async function droppedHarborLights(world: World): Promise<string> {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000002",
    preset: "all",
  });
  expect(added.status, "Harbor Lights is added").toBe(200);
  const { mediaDir } = world.manifest;
  if (mediaDir === undefined) {
    throw new Error("the Scenario writes no media dir");
  }
  const folder = join(world.node.dataDir, "downloads", "Harbor Lights (2019)");
  await mkdir(folder, { recursive: true });
  await copyFile(join(mediaDir, "shelf", harbor), join(folder, harbor));
  return Added.parse(added.body).work.id;
}

// The phone lists each download as a row that pushes its own page.
async function openDownload(page: Page): Promise<Locator> {
  const downloads = page.getByRole("list", { name: "Pending imports from downloads" });
  if (!isPhone(page)) {
    return downloads.getByRole("listitem", { name: release, exact: true });
  }
  await downloads.getByRole("link").filter({ hasText: release }).click();
  await expect(page).toHaveURL(/\/library\/pending\/[\da-f-]+$/u);
  return page.getByRole("listitem", { name: release, exact: true });
}

const fileRow = (scope: Locator, name: string) =>
  scope.getByRole("listitem").filter({ has: scope.page().getByText(name, { exact: true }) });

// Import files on the folder dropped in home's download folder: its one
// video waits to be confirmed, and Import all places it in Harbor Lights.
async function importHarborLightsByHand(
  page: Page,
  world: World,
  testInfo: TestInfo,
): Promise<void> {
  const harborId = await droppedHarborLights(world);
  await page.getByRole("button", { name: "Import files", exact: true }).click();
  const sheet = page.getByRole("dialog", { name: "Import files" });
  await sheet.getByRole("button", { name: "Choose a folder", exact: true }).click();
  await browser(page)
    .getByRole("listbox", { name: "Folders" })
    .getByRole("option")
    .filter({ hasText: join(world.node.dataDir, "downloads") })
    .click();
  await drillInto(page, "Harbor Lights (2019)");
  await browser(page)
    .getByRole("button", { name: "Open Harbor Lights (2019)", exact: true })
    .click();
  const harborRows = sheet.getByRole("region", { name: "Harbor Lights" });
  await expect(
    fileRow(harborRows, harbor).getByRole("button", { name: "Import", exact: true }),
  ).toBeVisible({
    timeout: 30_000,
  });
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("import-files.png"));
  await sheet.getByRole("button", { name: "Import all", exact: true }).click();
  await expect(page.getByRole("region", { name: "Notifications" })).toContainText(
    "Importing 1 file.",
  );
  await expect(sheet).toBeHidden();
  await expect
    .poll(async () => (await libraryRow(world, "Harbor Lights"))?.files, {
      message: "Harbor Lights holds its file",
      timeout: 60_000,
    })
    .toBe(1);
  expect(await filesOf(world, harborId)).toEqual(["Harbor Lights (2019) - 1080p.mkv webdl-1080p"]);
}

test("a folder imports by hand, and a download that isn't an upgrade forces into 1080p from Pending imports", async ({
  adminPage: page,
  world,
}, testInfo) => {
  test.slow();
  await waitingDownload(world);
  await openPendingImports(page);

  await importHarborLightsByHand(page, world, testInfo);

  const download = await openDownload(page);
  await expect(download).toContainText(`${orchard} · Downloaded on home`);
  const file = download.getByRole("listitem", { name: video, exact: true });
  await expect(file.getByText(notBetter, { exact: true })).toBeVisible();
  await expect(file.getByRole("button", { name: "Don't import", exact: true })).toBeVisible();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("pending-imports.png"));

  await file.getByRole("button", { name: `Other actions for ${video}` }).click();
  await menuItem(page, `Other actions for ${video}`, "Force into 1080p").click();
  const confirm = page.getByRole("dialog", { name: "Force into 1080p?" });
  await expect(
    confirm.getByText(`This deletes ${kept}, because it fills no other Target.`, { exact: true }),
  ).toBeVisible();
  await expectAccessible(page);
  await confirm.getByRole("button", { name: "Force into 1080p", exact: true }).click();
  if (isPhone(page)) {
    await expect(page.getByText("Nothing in this download waits for you now.")).toBeVisible({
      timeout: 60_000,
    });
  } else {
    await expect(download).toBeHidden({ timeout: 60_000 });
  }
  await expect
    .poll(async () => filesOf(world, await orchardId(world)), {
      message: "the forced file replaced the better one",
      timeout: 60_000,
    })
    .toEqual(["The Quiet Orchard (2021) {tmdb-9000001} - 1080p.mkv bluray-720p"]);
});

test("Activity reads the Grab's Import, and its Waiting for you link opens the Import", async ({
  adminPage: page,
  world,
}, testInfo) => {
  const importId = await waitingDownload(world);
  await navigate(page, "Activity");
  const row = page.getByRole("table", { name: "Grabs" }).getByRole("row", { name: /Quiet/u });
  await expect(row.getByText("Importing 0 of 1 movie", { exact: true })).toBeVisible();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("activity.png"));

  await row.getByRole("link", { name: "Waiting for you", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/library/pending/${importId}$`, "u"));
  await expect(
    page.getByRole("listitem", { name: video, exact: true }).getByText(notBetter),
  ).toBeVisible();
});

test("on a phone Pending imports list the download, and its pushed page declines the file @phone", async ({
  adminPage: page,
  world,
}, testInfo) => {
  await waitingDownload(world);
  await openPendingImports(page);
  const downloads = page.getByRole("list", { name: "Pending imports from downloads" });
  const entry = downloads.getByRole("link").filter({ hasText: release });
  await expect(entry).toContainText("1 file waits for you");
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("pending-imports-phone.png"));

  await entry.click();
  await expect(page).toHaveURL(/\/library\/pending\/[\da-f-]+$/u);
  const file = page.getByRole("listitem", { name: video, exact: true });
  await expect(file.getByText(notBetter, { exact: true })).toBeVisible();
  await expectAccessible(page);
  await screenshotWholePage(page, testInfo.outputPath("pending-import-phone.png"));

  await file.getByRole("button", { name: "Don't import", exact: true }).click();
  await expect(page.getByText("Nothing in this download waits for you now.")).toBeVisible({
    timeout: 60_000,
  });
});
