import { HttpResponse, http } from "msw";

import type { ImportsImportEntry, ImportsImportFile } from "../api/gen/model";
import {
  ActOnImportFileBody,
  ConfirmImportsBody,
  ImportFolderBody,
  SetImportMoveFilesBody,
} from "../api/gen/zod/imports/imports.zod";
import type { ImportsWorld, Sent } from "./imports";

const problemType = { "Content-Type": "application/problem+json" };

export const refused = (status: number, code: string, title: string) =>
  HttpResponse.json({ status, code, title }, { status, headers: problemType });

/** The outcome a file action leaves a file at, as the Controller would. */
const actedOn = (file: ImportsImportFile, action: string): ImportsImportFile => {
  if (action === "confirm" || action === "force") {
    return {
      ...file,
      outcome: "placing",
      reason: null,
      recommended: null,
      actions: [],
      forces: [],
    };
  }
  if (action === "decline" || action === "notThisRelease") {
    return {
      ...file,
      outcome: "declined",
      reason: { code: "by_admin", sentence: "You chose not to import this file." },
      recommended: null,
      actions: [],
      forces: [],
    };
  }
  return file;
};

function replaced(world: ImportsWorld, next: ImportsImportEntry) {
  world.imports = world.imports.map((found) => (found.id === next.id ? next : found));
  world.changed([]);
}

/** An Import once Import all confirmed each file that waited for it. */
const confirmedOf = (entry: ImportsImportEntry): ImportsImportEntry => ({
  ...entry,
  stage: "placing",
  files: entry.files.map((file) =>
    file.recommended === "confirm" ? actedOn(file, "confirm") : file,
  ),
});

const openingOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports", async ({ request }) => {
    const { nodeId, folder } = ImportFolderBody.parse(await request.json());
    sent.opened.push({ nodeId, folder });
    const open = world.folder.filter((entry) => !world.imports.some(({ id }) => id === entry.id));
    world.imports = [...world.imports, ...open];
    world.changed([]);
    return HttpResponse.json({ imports: world.folder }, { status: open.length === 0 ? 200 : 201 });
  }),
  http.post("*/api/v1/imports/confirm", async ({ request }) => {
    const { importIds } = ConfirmImportsBody.parse(await request.json());
    sent.confirmed.push(importIds);
    const confirmed = world.imports
      .filter(({ id }) => importIds.includes(id))
      .map((entry) => confirmedOf(entry));
    for (const entry of confirmed) {
      replaced(world, entry);
    }
    return HttpResponse.json({ imports: confirmed });
  }),
  http.post("*/api/v1/imports/move-files", async ({ request }) => {
    const { importIds, on } = SetImportMoveFilesBody.parse(await request.json());
    sent.moved.push({ importIds, on });
    const named = world.imports.filter(({ id }) => importIds.includes(id));
    if (on && named.some(({ files }) => files.some(({ movable }) => !movable))) {
      return refused(
        409,
        "import_move_seeding",
        "A tracked torrent seeds one of these files. Move files can't take a file while it seeds.",
      );
    }
    for (const entry of named) {
      replaced(world, { ...entry, moveFiles: on });
    }
    return new HttpResponse(null, { status: 204 });
  }),
];

const actingOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports/:id/files/:position", async ({ params, request }) => {
    const id = String(params["id"]);
    const position = Number(params["position"]);
    const body = ActOnImportFileBody.parse(await request.json());
    const { action } = body;
    sent.acted.push({ id, position, action });
    if (body.action === "match") {
      sent.fileMatches.push(body);
    }
    const entry = world.imports.find((found) => found.id === id);
    if (entry === undefined || !entry.files.some((file) => file.position === position)) {
      return refused(404, "import_not_found", "No Import has that id");
    }
    const next = {
      ...entry,
      files: entry.files.map((file) => (file.position === position ? actedOn(file, action) : file)),
    };
    replaced(world, next);
    return HttpResponse.json(next);
  }),
];

/** The download and folder Imports' routes: open a folder, Import all, Move files and each file's actions. */
export const filesOf = (world: ImportsWorld, sent: Sent) => [
  ...openingOf(world, sent),
  ...actingOf(world, sent),
];
