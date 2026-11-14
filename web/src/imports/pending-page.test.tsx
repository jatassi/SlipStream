import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetCountsMockHandler } from "../api/gen/query/shell/shell.msw";
import { halfLoop } from "../test/import-entries";
import { brazil, halloween, heat, orchardLeft, severanceEntry } from "../test/pending-imports";
import { openPending } from "../test/pending-page";
import { asPhone } from "../test/viewport";

const entry = (folder: string) => page.getByRole("listitem", { name: folder });

test("Pending imports leads with what each root's last scan added and how many folders need you", async () => {
  openPending();
  await expect
    .element(
      page.getByText(
        "The last scan of the Films library root on home added 398 films. 3 folders need you.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      page.getByText(
        "The last scan of the Shows library root on home didn't add any shows. 1 folder needs you.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByRole("link", { name: "Show the films it added" }))
    .toHaveAttribute("href", "/library/cinema?scanned=1&root=0199f0a1-1111-7000-8000-000000000001");
  expect(page.getByRole("link", { name: "Show the shows it added" }).elements()).toHaveLength(0);
});

test("an entry leads with why it waits and its suggested match, and keeps the files under Details", async () => {
  openPending();
  const row = entry(halloween.folder);
  await expect.element(row.getByText("1 file · 8.0 GB · Bluray-1080p")).toBeVisible();
  await expect
    .element(row.getByText("3 films match Halloween. The name has no year."))
    .toBeVisible();
  await expect.element(row.getByText("Halloween (1978)")).toBeVisible();
  await expect.element(row.getByText("Halloween/Halloween.mkv")).not.toBeInTheDocument();
  await userEvent.click(row.getByRole("button", { name: "Details" }));
  await expect.element(row.getByText("Halloween/Halloween.mkv")).toBeVisible();

  const odd = entry(brazil.folder);
  await expect.element(odd.getByText("No film matches Brazil (1985).")).toBeVisible();
  await expect
    .element(odd.getByText("ffprobe can't read this file. Replace it, then scan again."))
    .toBeVisible();
  await expect.element(odd.getByText("No match yet.")).toBeVisible();
  expect(odd.getByRole("button", { name: "Add" }).elements()).toHaveLength(0);
  await userEvent.click(odd.getByRole("button", { name: "Details" }));
  await expect.element(odd.getByText("Invalid data found when processing input")).toBeVisible();
});

test("a folder whose Work lives in another root offers only Ignore", async () => {
  openPending();
  const row = entry(heat.folder);
  await expect
    .element(
      row.getByText(
        "Heat (1995) is in the Films library root on vault. A Work's files all live in one root. Move these files there, or ignore them.",
      ),
    )
    .toBeVisible();
  expect(
    row
      .getByRole("button")
      .elements()
      .map((button) => button.textContent),
  ).toEqual(["Ignore", "Details"]);
});

test("Add adds an entry's suggested Work and it leaves the list", async () => {
  const { sent } = openPending();
  await userEvent.click(entry(halloween.folder).getByRole("button", { name: "Add" }));
  await expect.element(page.getByText("Added Halloween (1978).")).toBeVisible();
  expect(sent.added).toEqual([[halloween.id]]);
  await expect.element(entry(halloween.folder)).not.toBeInTheDocument();
});

test("Add says which folder wasn't added and why", async () => {
  const { world } = openPending();
  await expect.element(entry(halloween.folder)).toBeVisible();
  world.entries = world.entries.map((found) =>
    found.id === halloween.id ? Object.assign({}, found, { suggestion: null }) : found,
  );
  await userEvent.click(entry(halloween.folder).getByRole("button", { name: "Add" }));
  await expect
    .element(page.getByText("Halloween wasn't added. The Import has no suggested Work."))
    .toBeVisible();
});

test("Add all adds every entry whose suggestion can join its root", async () => {
  const { sent } = openPending();
  await userEvent.click(page.getByRole("button", { name: "Add all 2" }));
  await expect.element(page.getByText("Added 2 Works.")).toBeVisible();
  expect(sent.added).toEqual([[halloween.id, severanceEntry.id]]);
  await expect.element(entry(heat.folder)).toBeVisible();
  await expect.element(entry(brazil.folder)).toBeVisible();
});

test("Change searches the source and makes the record picked the match", async () => {
  const { sent } = openPending();
  await userEvent.click(entry(halloween.folder).getByRole("button", { name: "Change" }));
  const sheet = page.getByRole("dialog", { name: "Change the match for Halloween" });
  await expect.element(sheet.getByRole("searchbox", { name: "Title" })).toHaveValue("Halloween");
  await userEvent.click(sheet.getByRole("button", { name: "Match Halloween (2018)" }));
  await expect.element(sheet).not.toBeInTheDocument();
  await expect.element(page.getByText("Halloween now matches Halloween (2018).")).toBeVisible();
  expect(sent.matched).toEqual([{ id: halloween.id, key: "424139" }]);
  expect(sent.searched).toEqual(["Halloween"]);
  await expect.element(entry(halloween.folder).getByText("Halloween (2018)")).toBeVisible();
});

test("Import into attaches a series folder's files at the episodes picked, leaving the rest", async () => {
  const { sent } = openPending();
  await userEvent.click(entry(severanceEntry.folder).getByRole("button", { name: "Import into" }));
  const sheet = page.getByRole("dialog", { name: "Import into a Show" });
  await expect.element(sheet.getByText("Severance", { exact: true })).toBeVisible();
  await expect.element(sheet.getByRole("button", { name: "Import" })).toBeDisabled();
  await userEvent.click(sheet.getByRole("combobox", { name: "Severance.mkv" }));
  await userEvent.click(page.getByRole("option", { name: "S01E02 Half Loop" }));
  await userEvent.click(sheet.getByRole("button", { name: "Import" }));
  await expect
    .element(page.getByText("Imported 1 file into Severance. 1 file still waits."))
    .toBeVisible();
  expect(sent.imported).toEqual([
    {
      id: severanceEntry.id,
      workId: severanceEntry.suggestion?.workId,
      files: [{ position: 0, itemIds: [halfLoop.id] }],
    },
  ]);
  await expect
    .element(entry(severanceEntry.folder).getByText("This show has no episode S03E14."))
    .toBeVisible();
});

test("Import into lists only the Works in the folder's root, and a film's files join its one Item", async () => {
  const { sent, world } = openPending();
  await userEvent.click(entry(brazil.folder).getByRole("button", { name: "Import into" }));
  const sheet = page.getByRole("dialog", { name: "Import into a Film" });
  await expect.element(sheet.getByRole("button", { name: "Brazil" })).toBeVisible();
  expect(sheet.getByRole("button", { name: "Heat" }).elements()).toHaveLength(0);
  await userEvent.click(sheet.getByRole("button", { name: "Brazil" }));
  await expect.element(sheet.getByText("Its 2 files join Brazil.")).toBeVisible();
  await userEvent.click(sheet.getByRole("button", { name: "Import" }));
  await expect.element(page.getByText("Imported 2 files into Brazil.")).toBeVisible();
  const [item] = world.works.find(({ label }) => label === "Brazil")?.items ?? [];
  expect(sent.imported.at(0)?.files.map(({ itemIds }) => itemIds)).toEqual([
    [item?.id],
    [item?.id],
  ]);
  await expect.element(entry(brazil.folder)).not.toBeInTheDocument();
});

test("Ignore sets a folder aside under Ignored, where Restore brings it back", async () => {
  const { sent } = openPending();
  await userEvent.click(entry(brazil.folder).getByRole("button", { name: "Ignore" }));
  await expect
    .element(
      page.getByText("Ignored Brazil (1985). A library scan lists it again if its files change."),
    )
    .toBeVisible();
  expect(sent.ignored).toEqual([[brazil.id]]);
  await expect.element(entry(brazil.folder)).not.toBeInTheDocument();

  await userEvent.click(page.getByRole("link", { name: "Ignored" }));
  await expect
    .element(
      entry(orchardLeft.folder).getByText(
        "This film was deleted from the library and its files were left on disk.",
      ),
    )
    .toBeVisible();
  expect(page.getByText(/^The last scan of/u).elements()).toHaveLength(0);
  await userEvent.click(entry(brazil.folder).getByRole("button", { name: "Restore" }));
  await expect.element(page.getByText("Brazil (1985) needs you again.")).toBeVisible();
  expect(sent.restored).toEqual([[brazil.id]]);
  await expect.element(entry(brazil.folder)).not.toBeInTheDocument();
});

test("with nothing waiting, each view says so", async () => {
  openPending({ start: { entries: [], leads: [], imports: [] } });
  await expect
    .element(
      page.getByText(
        "Nothing needs you. A download SlipStream can't import without you, and each folder a library scan can't match to a Work with confidence, waits here.",
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("link", { name: "Ignored" }));
  await expect.element(page.getByText("No folders are ignored.")).toBeVisible();
});

test("the Library counts the pending imports that need you and links to Pending imports, a pushed page on a phone", async () => {
  await asPhone();
  const { app } = openPending({
    path: "/library",
    handlers: [
      getGetCountsMockHandler({
        activity: 0,
        downloads: { done: 0, wanted: 0, allPaused: false },
        alerts: { nodes: null, health: null, tasks: null },
        pendingImports: 4,
      }),
    ],
  });
  await expect
    .element(page.getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: /Library/u }))
    .toHaveTextContent("Library4");
  await userEvent.click(page.getByRole("link", { name: /4 pending imports need you/u }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/library/pending");
  await expect
    .element(page.getByRole("banner").getByRole("link", { name: "Library" }))
    .toHaveAttribute("href", "/library");
  await expect.element(entry(halloween.folder)).toBeVisible();
});
