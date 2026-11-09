import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { SettingField } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { cinema, libraryRow, libraryServer } from "../test/library";
import { fileTokens, filmTokens, namingServer, namingSettings } from "../test/naming";
import { type PlannedWork, renamesServer, row } from "../test/renames";
import { renderApp } from "../test/render-app";

const heatId = "0b7f2c1e-4d5a-4e6b-8c9d-1a2b3c4d5e01";
const orchardId = "0b7f2c1e-4d5a-4e6b-8c9d-1a2b3c4d5e02";
const shelfId = "0b7f2c1e-4d5a-4e6b-8c9d-1a2b3c4d5e03";
const roninId = "0b7f2c1e-4d5a-4e6b-8c9d-1a2b3c4d5e04";
const file = (n: number) => `0c8e3d2f-5e6b-4f7c-9d0e-2b3c4d5e6f${String(n).padStart(2, "0")}`;

const heat: PlannedWork = {
  workId: heatId,
  contentType: "cinema",
  title: "Heat",
  digest: "aaaaaaaaaaaaaaaa",
  rows: [
    row(file(1), "Heat 1995/Heat 1995 - 4K.mkv", "Heat (1995)/Heat (1995) - 4K.mkv", {
      companions: ["Heat 1995 - 4K.en.srt", "Heat 1995 - 4K.nfo"],
    }),
    row(file(2), "Heat 1995/Heat 1995 - 1080p.mkv", "Heat (1995)/Heat (1995) - 1080p.mkv"),
  ],
};

const orchard: PlannedWork = {
  workId: orchardId,
  contentType: "cinema",
  title: "The Quiet Orchard",
  digest: "bbbbbbbbbbbbbbbb",
  rows: [
    row(
      file(3),
      "Quiet Orchard/Quiet Orchard - 4K.mkv",
      "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 4K.mkv",
    ),
  ],
};

const shelf: PlannedWork = {
  workId: shelfId,
  contentType: "cinema",
  title: "The Long Shelf",
  digest: "cccccccccccccccc",
  moving: 2,
  rows: [
    row(file(11), "Shelf/Shelf - 1x01.mkv", "Shelf/Shelf - 1x02.mkv", { group: 1 }),
    row(file(12), "Shelf/Shelf - 1x02.mkv", "Shelf/Shelf - 1x01.mkv", { group: 1 }),
    row(file(13), "Shelf/Shelf - 1x03.mkv", "Ronin (1998)/Ronin (1998) - 4K.mkv", {
      blocked: {
        reason: "clash",
        otherWorkId: roninId,
        detail: "Another Work's file has the same new path.",
      },
    }),
    row(file(14), "Shelf/Shelf - 1x04.mkv", "Shelf/Shelf - 1x04 Arrival.mkv", {
      blocked: {
        reason: "occupied",
        detail:
          "A file SlipStream doesn't track is at the new path. Move or delete that file, then plan again.",
      },
    }),
  ],
};

const fileKey = "cinema.naming.file.standard";

const settings: SettingField[] = namingSettings("cinema", [
  ["folder", "Film folder", "{Film Title} {(Year)}"],
  ["file.standard", "Film file", "{Film Title} {(Year)} - {Edition Tag} {Target}"],
]);

function openNaming(renames: Parameters<typeof renamesServer>[0]) {
  const naming = namingServer([
    {
      naming: { contentType: "cinema", settings, problems: [] },
      tokens: [
        { key: "cinema.naming.folder", tokens: filmTokens },
        { key: fileKey, tokens: fileTokens },
      ],
    },
  ]);
  const order: string[] = [];
  const server = renamesServer({
    ...renames,
    onConfirm: () => {
      order.push(`rename after ${naming.sent.length} saves`);
    },
  });
  renderApp({
    path: "/settings/naming",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      libraryServer([libraryRow({ id: roninId, key: "ronin", label: "Ronin" })]).handler,
      ...naming.handlers,
      ...server.handlers,
    ],
  });
  return { ...server, sent: naming.sent, order };
}

const sheet = page.getByRole("dialog", { name: "Rename files" });

async function reviewAll() {
  await userEvent.click(page.getByRole("button", { name: "Review all renames" }));
  await expect.element(sheet).toBeVisible();
}

test("the library's plan leads with what it renames, opens a Work to its paths, and renames only what stays ticked", async () => {
  const server = openNaming({ plan: [heat, orchard], moving: 128 });
  await expect.element(page.getByText("Renaming 128 files…")).toBeVisible();
  await reviewAll();
  await expect.element(sheet.getByText("This renames 3 files in 2 Works.")).toBeVisible();
  expect(server.planned).toEqual([{ scope: { kind: "library" } }]);

  await userEvent.click(sheet.getByRole("button", { name: "Files of Heat" }));
  const files = sheet.getByRole("list", { name: "Files of Heat" });
  await expect
    .element(files.getByRole("deletion").first())
    .toHaveTextContent("Heat 1995/Heat 1995 - 4K.mkv");
  await expect
    .element(files.getByRole("insertion").first())
    .toHaveTextContent("Heat (1995)/Heat (1995) - 4K.mkv");
  await expect
    .element(files.getByText("Moves with it: Heat 1995 - 4K.en.srt and Heat 1995 - 4K.nfo"))
    .toBeVisible();

  await userEvent.click(files.getByRole("checkbox", { name: "Heat 1995 - 1080p.mkv" }));
  await userEvent.click(sheet.getByRole("checkbox", { name: "The Quiet Orchard" }));
  await expect.element(sheet.getByText("This renames 1 file in 1 Work.")).toBeVisible();
  await expect.element(sheet.getByText("1 of 2 files")).toBeVisible();

  await userEvent.click(sheet.getByRole("button", { name: "Rename 1 file" }));
  await expect.element(sheet).not.toBeInTheDocument();
  expect(server.confirmed).toEqual([
    [
      { work: heatId, digest: heat.digest, leave: [file(2)] },
      { work: orchardId, digest: orchard.digest, leave: [file(3)] },
    ],
  ]);
});

test("files that trade names tick together, and a blocked file says why, naming the other Work", async () => {
  const server = openNaming({ plan: [shelf] });
  await reviewAll();
  await expect.element(sheet.getByText("This renames 2 files in 1 Work.")).toBeVisible();
  await expect
    .element(sheet.getByText("2 files can't be renamed now. Open their Works to see why."))
    .toBeVisible();
  await expect
    .element(sheet.getByText("2 files are still moving and aren't listed."))
    .toBeVisible();

  await userEvent.click(sheet.getByRole("button", { name: "Files of The Long Shelf" }));
  const files = sheet.getByRole("list", { name: "Files of The Long Shelf" });
  await expect
    .element(
      files.getByText(
        "Ronin has a file with the same new path. Add an id to the folder template so each Work gets its own folder.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      files.getByText(
        "A file SlipStream doesn't track is at the new path. Move or delete that file, then plan again.",
      ),
    )
    .toBeVisible();
  await expect.element(files.getByRole("checkbox", { name: "Shelf - 1x03.mkv" })).toBeDisabled();

  await userEvent.click(files.getByRole("checkbox", { name: "Shelf - 1x01.mkv" }));
  await expect.element(files.getByRole("checkbox", { name: "Shelf - 1x02.mkv" })).not.toBeChecked();
  await expect.element(sheet.getByText("No files are ticked to rename.")).toBeVisible();
  await expect.element(sheet.getByRole("button", { name: "Rename 0 files" })).toBeDisabled();

  await userEvent.click(files.getByRole("checkbox", { name: "Shelf - 1x02.mkv" }));
  await expect.element(files.getByRole("checkbox", { name: "Shelf - 1x01.mkv" })).toBeChecked();
  await userEvent.click(sheet.getByRole("button", { name: "Rename 2 files" }));
  await expect.element(sheet).not.toBeInTheDocument();
  expect(server.confirmed).toEqual([[{ work: shelfId, digest: shelf.digest }]]);
});

test("a save that renames files shows the draft's plan, and Save and rename saves before it renames", async () => {
  const server = openNaming({ plan: [heat, orchard] });
  const card = page.getByRole("article", { name: "Films" });
  await userEvent.click(card.getByRole("button", { name: "Details" }));
  await userEvent.fill(
    card.getByRole("combobox", { name: "Film file" }),
    "{Film Title} - {Target}",
  );
  await userEvent.click(card.getByRole("button", { name: "Save", exact: true }));

  const saving = page.getByRole("dialog", { name: "Save film naming" });
  await expect.element(saving.getByText("Saving renames 3 files in 2 Works.")).toBeVisible();
  expect(server.planned).toEqual([
    {
      scope: { kind: "type", id: "cinema" },
      draft: { contentType: "cinema", values: { [fileKey]: "{Film Title} - {Target}" } },
    },
  ]);
  expect(server.sent).toEqual([]);

  await userEvent.click(saving.getByRole("button", { name: "Save and rename 3 files" }));
  await expect.element(saving).not.toBeInTheDocument();
  expect(server.sent).toEqual([{ [fileKey]: "{Film Title} - {Target}" }]);
  expect(server.order).toEqual(["rename after 1 saves"]);
  expect(server.confirmed).toEqual([
    [
      { work: heatId, digest: heat.digest },
      { work: orchardId, digest: orchard.digest },
    ],
  ]);
});

test("Save only stores the settings, renames nothing, and the card still counts the files", async () => {
  const server = openNaming({ plan: [heat], unmatched: [{ contentType: "cinema", files: 2 }] });
  const card = page.getByRole("article", { name: "Films" });
  await expect.element(card.getByText("2 film files don't match the templates.")).toBeVisible();
  await userEvent.click(card.getByRole("button", { name: "Details" }));
  await userEvent.fill(
    card.getByRole("combobox", { name: "Film file" }),
    "{Film Title} - {Target}",
  );
  await userEvent.click(card.getByRole("button", { name: "Save", exact: true }));

  const saving = page.getByRole("dialog", { name: "Save film naming" });
  await userEvent.click(saving.getByRole("button", { name: "Save only" }));
  await expect.element(saving).not.toBeInTheDocument();
  expect(server.sent).toEqual([{ [fileKey]: "{Film Title} - {Target}" }]);
  expect(server.confirmed).toEqual([]);
  await expect.element(card.getByText("2 film files don't match the templates.")).toBeVisible();
});

test("the card's Review renames opens its type's plan, and Works that changed since come back ticked alone", async () => {
  const server = openNaming({
    plan: [heat, orchard],
    unmatched: [{ contentType: "cinema", files: 3 }],
    changed: [orchardId],
  });
  const card = page.getByRole("article", { name: "Films" });
  await userEvent.click(card.getByRole("button", { name: "Review renames" }));
  const review = page.getByRole("dialog", { name: "Rename film files" });
  await userEvent.click(review.getByRole("button", { name: "Rename 3 files" }));

  await expect
    .element(
      review.getByText("1 Work changed since you reviewed it. Check it and rename it again."),
    )
    .toBeVisible();
  await expect.element(review.getByText("This renames 1 file in 1 Work.")).toBeVisible();
  await expect.element(review.getByRole("checkbox", { name: "Heat" })).not.toBeChecked();
  expect(server.planned.map(({ scope }) => scope)).toEqual([
    { kind: "type", id: "cinema" },
    { kind: "type", id: "cinema" },
  ]);
});

test("the files the disk queue moves come off the card's count as they land", async () => {
  const server = openNaming({
    plan: [],
    unmatched: [{ contentType: "cinema", files: 3 }],
    moving: 3,
  });
  const card = page.getByRole("article", { name: "Films" });
  await expect.element(card.getByText("3 film files don't match the templates.")).toBeVisible();
  await expect.element(page.getByText("Renaming 3 files…")).toBeVisible();

  server.counts({ unmatched: [], moving: 0 }, [heatId]);
  await expect
    .element(card.getByText("Plex, Jellyfin and Emby show the files of each film as one item."))
    .toBeVisible();
  await expect.element(page.getByText(/^Renaming/u)).not.toBeInTheDocument();
});
