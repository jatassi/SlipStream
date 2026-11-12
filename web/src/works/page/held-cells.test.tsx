import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesCellList, FilesFileList } from "../../api/gen/model";
import { downloading, importingPack } from "../../test/grabs";
import { orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { scannedFiles } from "../../test/scanned-files";
import { awayGrab, filled } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

const cutRow = () =>
  page.getByRole("table", { name: "Editions and Targets" }).getByRole("row", { name: /^Any cut/u });
const files = () => page.getByRole("list", { name: "Files", exact: true });

const [met, inFlight] = filled.cells;

function held(step: "waiting" | "placing"): FilesCellList {
  if (met?.state !== "met") {
    throw new Error("filled leads with a Met cell");
  }
  const from = "2026-10-12T09:03:00Z";
  return {
    ...filled,
    cells: [{ ...met, held: { grabId: downloading.id, step, from, until: null } }],
  };
}

function sentFromSeedbox(): FilesCellList {
  if (inFlight?.state !== "downloading") {
    throw new Error("filled ends with a downloading cell");
  }
  const from = "2026-10-12T09:03:00Z";
  const until = "2026-10-12T16:10:00Z";
  return { ...filled, cells: [{ ...inFlight, step: "sending", from, until }] };
}

test("a Met cell a waiting file holds reads Met with Waiting for you in its facts", async () => {
  const server = workServer({ startCells: held("waiting") });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves(downloading);
  await expect.element(cutRow()).toMatchTextContent(/Met.*WEBDL-1080p.*Waiting for you/u);
});

test("a Met cell a placing file holds reads Importing", async () => {
  const server = workServer({ startCells: held("placing") });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves(downloading);
  await expect.element(cutRow()).toMatchTextContent(/Met.*Importing/u);
});

test("a downloading cell names the Node its file is sent from and the Import delay it waits for", async () => {
  const server = workServer({ startCells: sentFromSeedbox() });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves(awayGrab);
  await expect.element(cutRow()).toMatchTextContent(/Sending from seedbox/u);
  await expect
    .element(cutRow())
    .toMatchTextContent(/Waiting for the import delay, until \d{1,2}:10(\s?[AP]M)?\./u);
});

test("a downloading cell whose file waits for the Admin says Waiting for you once, as the link to its Import", async () => {
  if (inFlight?.state !== "downloading") {
    throw new Error("filled ends with a downloading cell");
  }
  const from = "2026-10-12T09:03:00Z";
  const server = workServer({
    startCells: { ...filled, cells: [{ ...inFlight, step: "waiting", from }] },
  });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves({ ...importingPack, id: downloading.id });
  await expect
    .element(cutRow().getByRole("link", { name: "Waiting for you" }))
    .toHaveAttribute("href", `/library/pending/${importingPack.id}`);
  expect(
    cutRow()
      .element()
      .textContent.match(/Waiting for you/gu),
  ).toHaveLength(1);
});

function placedFiles(placedBy: "link" | "transfer"): FilesFileList {
  const [file] = scannedFiles.files;
  if (file === undefined) {
    throw new Error("scannedFiles lists a file");
  }
  const grab = { id: awayGrab.id, release: awayGrab.releaseTitle };
  return { ...scannedFiles, files: [{ ...file, placedBy, grab }] };
}

test("a file's Details say how its import placed it and the Grab it came from", async () => {
  const server = workServer({ files: placedFiles("link") });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  const file = files().getByRole("listitem").nth(0);
  await userEvent.click(file.getByRole("button", { name: "Details" }));
  await expect
    .element(
      file.getByText("Hard linked. The download and the library share the same data on disk."),
    )
    .toBeVisible();
  await expect.element(file.getByText(`Grab ${awayGrab.releaseTitle}`)).toBeVisible();
});

test("a file sent between Nodes names the Node it came from", async () => {
  const server = workServer({ files: placedFiles("transfer") });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  server.grabMoves(awayGrab);
  const file = files().getByRole("listitem").nth(0);
  await userEvent.click(file.getByRole("button", { name: "Details" }));
  await expect.element(file.getByText("SlipStream sent this file from seedbox.")).toBeVisible();
});
