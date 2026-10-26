import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../api/gen/query/works/works.msw";
import { done, firstSetup, stepHeading, todo } from "../../test/first-setup";
import { cinema } from "../../test/library";
import { nodesAre } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { zines } from "../../test/roots-page";
import { filmsRoot, media, notStorage, serveRoots } from "../../test/roots";

function open(roots = [filmsRoot]) {
  const server = firstSetup([done("address"), todo("roots"), done("targets")]);
  const served = serveRoots({
    roots,
    changed: (rootIds) => {
      server.send({ event: "roots.changed", data: { rootIds } });
    },
  });
  renderApp({
    path: "/setup/roots",
    handlers: [
      ...server.handlers,
      nodesAre(media, notStorage),
      getListContentTypesMockHandler({
        contentTypes: [cinema, zines, { ...zines, id: "off", name: "Off", enabled: false }],
      }),
    ],
  });
  return { server, served };
}

const row = (name: string) =>
  page.getByRole("list", { name: "Library roots" }).getByRole("listitem").filter({ hasText: name });

test("the roots step shows each enabled type's folder, or Add a folder, and says what skipping costs", async () => {
  open();
  await expect.element(stepHeading("Library roots")).toBeVisible();
  await expect.element(row("Films")).toHaveTextContent("Films/srv/media/films on home");
  await expect.element(row("Zines")).toHaveTextContent("ZinesNo folder yetAdd a folder");
  await expect.element(row("Off")).not.toBeInTheDocument();
  await expect
    .element(page.getByText("Works you add wait for a root and are placed when you add one."))
    .toBeVisible();
});

test("Add a folder opens Add a library root with the type chosen and the only Storage Node picked", async () => {
  const { served } = open();
  await userEvent.click(row("Zines").getByRole("button", { name: "Add a folder" }));
  const dialog = page.getByRole("dialog", { name: "Add a library root" });
  await expect
    .element(dialog.getByRole("combobox", { name: "Content Type" }))
    .toHaveTextContent("Zines");
  await expect.element(dialog.getByRole("combobox", { name: "Node" })).not.toBeInTheDocument();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Folder" }), "/srv/media/zines");
  await expect
    .element(dialog.getByText("This folder is on /srv/media, 1.2 TB free."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Add" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(served.sent.created).toEqual([
    { contentType: "zine", nodeId: media.id, path: "/srv/media/zines", makeDefault: false },
  ]);
  await expect.element(row("Zines")).toHaveTextContent("Zines/srv/media/zines on home");
  await expect
    .element(page.getByText("Works you add wait for a root and are placed when you add one."))
    .not.toBeInTheDocument();
});
