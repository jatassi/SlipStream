import type { Page } from "@playwright/test";
import { z } from "zod";

import { node } from "../harness/manifest.ts";
import { orchardCells } from "./cells.ts";
import { type World, expect, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";
import {
  browser,
  drillInto,
  folders,
  menuItem,
  newFolder,
  upTo,
  useButton,
} from "./folder-browser.ts";

test.use({ scenario: "home" });
test.describe.configure({ timeout: 180_000 });

const Roots = z.object({
  roots: z.array(z.object({ path: z.string(), works: z.number() })),
});
const Volumes = z.object({
  nodes: z.array(z.object({ volumes: z.array(z.object({ rootIds: z.array(z.string()) })) })),
});
const Settings = z.object({ sampleEvery: z.object({ value: z.number() }) });
const Added = z.object({ work: z.object({ id: z.string() }) });
const Cells = z.object({ cells: z.array(z.object({ state: z.string() })) });

async function rootsOf(world: World) {
  return Roots.parse((await world.api("GET", "/api/v1/roots")).body).roots;
}

async function moviesRoot(world: World) {
  const [movies] = await rootsOf(world);
  if (movies === undefined) {
    throw new Error("the home Scenario has no Movies root");
  }
  return { ...movies, media: movies.path.slice(0, movies.path.lastIndexOf("/")) };
}

function homeId(world: World): string {
  const id = node(world.manifest, "home").nodeId;
  if (id === undefined) {
    throw new Error("the manifest has no id for home");
  }
  return id;
}

async function addOrchard(world: World): Promise<string> {
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    monitor: true,
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  return Added.parse(added.body).work.id;
}

async function cellStates(world: World, workId: string): Promise<string[]> {
  const reply = await world.api("GET", `/api/v1/works/${workId}/cells`);
  return Cells.parse(reply.body).cells.map((cell) => cell.state);
}

const roots = (page: Page) => page.getByRole("table", { name: "Movies roots" });
const rootRow = (page: Page, path: string) =>
  roots(page).getByRole("row").filter({ hasText: path });
const toasts = (page: Page) => page.getByRole("region", { name: "Notifications" });

async function act(page: Page, path: string, item: string): Promise<void> {
  await page.getByRole("button", { name: `Actions for ${path}` }).click();
  await menuItem(page, item).click();
}

type Movies = Awaited<ReturnType<typeof moviesRoot>>;

async function browseToFilms(page: Page, movies: Movies): Promise<void> {
  await expect(folders(page).getByRole("option").first()).toContainText(movies.path);
  await folders(page).getByRole("option").filter({ hasText: /^\/\d/u }).click();
  for (const name of movies.media.split("/").slice(1)) {
    // oxlint-disable-next-line no-await-in-loop -- each folder opens before the next, as a person's does
    await drillInto(page, name);
  }
  await folders(page).getByRole("option", { name: "movies", exact: true }).click();
  await expect(browser(page)).toContainText(
    "Can't use this folder because it's inside the Movies library root.",
  );
  await expect(useButton(page, "movies")).toBeDisabled();

  await upTo(page, "media");
  await expect(browser(page)).toContainText(
    "Can't use this folder because it contains the Movies library root.",
  );
  await expect(useButton(page, "media")).toBeDisabled();
  await newFolder(page, "films");
  await expect(useButton(page, "films")).toBeEnabled();
  await expect(browser(page)).toContainText(/This folder is on \/, .+ free\./u);
  await useButton(page, "films").click();
}

async function expectOneVolume(page: Page, world: World, paths: readonly string[]): Promise<void> {
  const [volume] = Volumes.parse((await world.api("GET", "/api/v1/volumes")).body).nodes.flatMap(
    (group) => group.volumes,
  );
  expect(volume?.rootIds, "both roots sit on one volume, counted once").toHaveLength(paths.length);
  const onHome = page.getByRole("list", { name: "Volumes on home" }).getByRole("listitem");
  await expect(onHome).toHaveCount(1);
  await expect(onHome).toContainText(/free of/u);
  for (const path of paths) {
    // oxlint-disable-next-line no-await-in-loop -- one assertion per root, on the same row
    await expect(onHome).toContainText(path);
  }
  await expect(onHome.getByRole("meter", { name: "/ used" })).toBeVisible();
}

async function checkStorageEveryTenMinutes(page: Page, world: World): Promise<void> {
  await page
    .getByText("Storage Nodes check each volume every 5 min")
    .locator("..")
    .getByRole("button", { name: "Details" })
    .click();
  await page.getByRole("textbox", { name: "Check storage every" }).fill("10");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(
    page.getByText("Storage Nodes check each volume every 10 min, and again after each import."),
  ).toBeVisible();
  const settings = Settings.parse((await world.api("GET", "/api/v1/roots/settings")).body);
  expect(settings.sampleEvery.value).toBe(600);
}

async function deleteMovingWorks(page: Page, root: string, to: string): Promise<void> {
  await act(page, root, "Delete");
  const deleting = page.getByRole("dialog", { name: `Delete ${root}?` });
  await expect(deleting).toContainText("Its 1 Movie move to the root you pick.");
  await expect(deleting.getByRole("combobox", { name: "Move its Movies to" })).toContainText(to);
  await deleting.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(toasts(page)).toContainText(
    `Deleted the ${root} root. The folder is still on disk.`,
  );
}

test("an Admin adds a root through home's folder browser, makes it the default, reads its volume, and deletes it with its Movie moving back", async ({
  adminPage: page,
  world,
}) => {
  const movies = await moviesRoot(world);
  const films = `${movies.media}/films`;
  await page.goto("/settings/library-roots");
  await expect(rootRow(page, movies.path)).toContainText("Default");

  await page.getByRole("button", { name: "Add a library root" }).click();
  const adding = page.getByRole("dialog", { name: "Add a library root" });
  await adding.getByRole("button", { name: "Browse" }).click();
  await browseToFilms(page, movies);
  await expect(adding.getByRole("textbox", { name: "Folder" })).toHaveValue(films);
  await expect(adding).toContainText(/This folder is on \/, .+ free\./u);
  await adding.getByRole("checkbox", { name: "Make this the default for Movies" }).check();
  await adding.getByRole("button", { name: "Add", exact: true }).click();
  await expect(toasts(page)).toContainText(`Added ${films}. New Movies go there.`);
  await expect(rootRow(page, films)).toContainText("Default");
  await expect(rootRow(page, movies.path)).not.toContainText("Default");

  await expectOneVolume(page, world, [movies.path, films]);
  await checkStorageEveryTenMinutes(page, world);

  const workId = await addOrchard(world);
  await expect(rootRow(page, films)).toContainText("1 Movie");
  await act(page, movies.path, "Make default");
  await expect(toasts(page)).toContainText(`New Movies now go into ${movies.path}.`);
  await expect(rootRow(page, movies.path)).toContainText("Default");

  await deleteMovingWorks(page, films, movies.path);
  await expect(rootRow(page, films)).toHaveCount(0);
  await expect(rootRow(page, movies.path)).toContainText("1 Movie");
  const kept = await rootsOf(world);
  expect(kept.map((root) => [root.path, root.works])).toEqual([[movies.path, 1]]);
  expect(await cellStates(world, workId)).toEqual(["missing"]);
});

test("the Diagram's Storage cabinet and inspector show home's roots and volume", async ({
  adminPage: page,
  world,
}) => {
  const movies = await moviesRoot(world);
  const made = await world.api("POST", "/api/v1/roots", {
    contentType: "movie",
    nodeId: homeId(world),
    path: movies.media,
  });
  expect(made.status, "a root that contains another is refused").toBe(422);

  await page.goto("/nodes?view=diagram");
  const storage = page
    .getByRole("figure", { name: "Diagram" })
    .locator(`[data-node="${homeId(world)}"] [data-role="storage"]:not([data-chip])`);
  const filled = storage.locator("[data-drawer][data-used]");
  await expect(filled, "one drawer for the one volume").toHaveCount(1);
  await expect(filled).toHaveAttribute("data-used", /^[01]\.\d\d$/u);
  await storage.first().click();
  const inspector = isPhone(page)
    ? page.getByRole("dialog", { name: "Storage on home" })
    : page.getByRole("complementary", { name: "Storage on home" });
  await expect(inspector).toContainText(/1 root on 1 volume, .+ free\./u);
  await inspector.getByRole("link", { name: "Library roots" }).click();
  await expect(page).toHaveURL(/\/settings\/library-roots$/u);
  await expect(rootRow(page, movies.path)).toContainText("Default");
});

async function imported(world: World): Promise<string> {
  const workId = await addOrchard(world);
  expect((await world.api("POST", `/api/v1/works/${workId}/search`)).status).toBe(202);
  await expect
    .poll(async () => (await cellStates(world, workId)).join(), {
      message: "the WEB-DL imports into the Movies root",
      timeout: 90_000,
    })
    .toBe("upgradable");
  return workId;
}

async function refusedWhileHolding(page: Page, root: string): Promise<void> {
  await act(page, root, "Delete");
  const refusing = page.getByRole("dialog", { name: `Delete ${root}?` });
  await refusing.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(refusing).toContainText("It can't be deleted while it holds 1 file.");
  await refusing.getByRole("button", { name: "Cancel" }).click();
}

async function rehomeForgetting(page: Page, movies: Movies): Promise<string> {
  await act(page, movies.path, "Edit");
  const editing = page.getByRole("dialog", { name: "Edit the Movies root" });
  await editing.getByRole("button", { name: "Browse" }).click();
  await upTo(page, "media");
  await newFolder(page, "moved");
  await useButton(page, "moved").click();
  const moved = `${movies.media}/moved`;
  await expect(editing.getByRole("textbox", { name: "Folder" })).toHaveValue(moved);
  await editing.getByRole("button", { name: "Save", exact: true }).click();
  await expect(editing).toContainText("1 file isn't at the new place.");
  await expect(editing.getByRole("list", { name: "Missing files" })).toContainText(
    "The Quiet Orchard (2021) - 1080p.mkvnot there",
  );
  await editing.getByRole("button", { name: "Save anyway, forget 1 file" }).click();
  await expect(toasts(page)).toContainText(`The Movies root is now ${moved} on home.`);
  return moved;
}

test("re-homing a root checks its files at the new place, and Save anyway forgets the one that isn't there", async ({
  adminPage: page,
  world,
}) => {
  const movies = await moviesRoot(world);
  const workId = await imported(world);
  await page.goto("/settings/library-roots");
  await expect(rootRow(page, movies.path)).toContainText("1 Movie");
  await refusedWhileHolding(page, movies.path);

  const moved = await rehomeForgetting(page, movies);
  await expect(rootRow(page, moved)).toContainText("Default");
  await expect(rootRow(page, moved)).toContainText("1 Movie");
  await expect.poll(() => cellStates(world, workId)).toEqual(["missing"]);
  await page.goto(`/works/${workId}`);
  await expect(orchardCells(page)).toContainText("Missing");

  await page.goto("/settings/library-roots");
  await act(page, moved, "Delete");
  const deleting = page.getByRole("dialog", { name: `Delete ${moved}?` });
  await expect(deleting).toContainText(
    "Its 1 Movie wait without a root until you add another Movies root.",
  );
  await deleting.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(toasts(page)).toContainText(
    `Deleted the ${moved} root. The folder is still on disk.`,
  );
  await expect(page.getByRole("button", { name: "Add a root for Movies" })).toBeVisible();
  await expect(page.getByRole("main")).toContainText(
    "Add a library root for Movies so SlipStream knows where to put them.",
  );
});
