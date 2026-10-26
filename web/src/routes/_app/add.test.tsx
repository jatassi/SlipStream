import { http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../api/gen/query/works/works.msw";
import { addServer, harvestId, shelfRoot } from "../../test/adding";
import { problem } from "../../test/handlers";
import { cinema, orchardId } from "../../test/library";
import { films, ultra } from "../../test/profiles";
import { renderApp } from "../../test/render-app";
import { filmsRoot } from "../../test/roots";

const sheet = () => page.getByRole("dialog", { name: "Add The Quiet Orchard" });
const notifications = () => page.getByRole("region", { name: "Notifications" });
const sentence = () => sheet().getByText(/^Adds /u);
const willStart = () => sheet().getByText("New Films will start with these.");

async function openSheet() {
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  await expect.element(sentence()).toBeVisible();
}

const quietPage = "/add?contentType=cinema&query=quiet";

test("searching puts the term in the URL and lists each candidate with its year, original title and overview", async () => {
  const server = addServer();
  const { router } = renderApp({ path: "/add?contentType=cinema", handlers: server.handlers });
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();

  await userEvent.fill(page.getByRole("searchbox", { name: "Title" }), "  quiet orchard ");
  await userEvent.click(page.getByRole("button", { name: "Search" }));

  const results = page.getByRole("list", { name: "Results" }).getByRole("listitem");
  await expect
    .element(results.first())
    .toHaveTextContent(
      "The Quiet Orchard2021 · Le Verger SilencieuxAn orchard keeper waits out a long winter.Add",
    );
  await expect.element(results.last()).toHaveTextContent("Orchard Harvest2019In the library");
  expect(router.state.location.href).toBe("/add?contentType=cinema&query=quiet+orchard");
  expect(server.searched).toEqual(["quiet orchard"]);
});

test("a candidate already in the library opens its Work page", async () => {
  const server = addServer();
  const { router } = renderApp({ path: quietPage, handlers: server.handlers });
  await userEvent.click(page.getByRole("link", { name: /^Orchard Harvest/u }));
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${harvestId}`);
});

test("the sheet opens on the preview and adds with the defaults, remembering, then opens the Work", async () => {
  const server = addServer();
  const { router } = renderApp({ path: quietPage, handlers: server.handlers });
  await openSheet();

  await expect
    .element(sentence())
    .toHaveTextContent(
      "Adds The Quiet Orchard (2021) to Films on /srv/media/films, monitored, wanting Films' 1 Target.",
    );
  await expect.element(sheet().getByText("2021 · 1h 52m")).toBeVisible();
  await expect.element(sheet().getByRole("switch", { name: "Monitor" })).toBeChecked();
  await expect
    .element(sheet().getByRole("switch", { name: "Remember these choices" }))
    .toBeChecked();
  expect(sheet().getByRole("combobox", { name: "Library root" }).query()).toBeNull();
  expect(willStart().query()).toBeNull();
  await expect
    .element(
      sheet().getByText("Adding doesn't search. To search now, use Search on its Work page."),
    )
    .toBeVisible();

  await userEvent.click(sheet().getByRole("button", { name: "Add Film" }));
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  await expect.element(notifications().getByText("Added The Quiet Orchard")).toBeVisible();
  expect(server.posted).toEqual([
    {
      contentType: "cinema",
      key: "9000001",
      profileId: films.id,
      rootId: filmsRoot.id,
      monitor: true,
      remember: true,
    },
  ]);
  expect(router.state.location.href).toBe(`/works/${orchardId}`);
});

test("a changed choice says new Works will start with it, until Remember is off", async () => {
  const server = addServer();
  renderApp({ path: quietPage, handlers: server.handlers });
  await openSheet();

  await userEvent.click(sheet().getByRole("combobox", { name: "Profile" }));
  await userEvent.click(page.getByRole("option", { name: "Ultra" }));
  await userEvent.click(sheet().getByRole("switch", { name: "Monitor" }));
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Adds The Quiet Orchard (2021) to Films on /srv/media/films, not monitored, wanting Ultra's 2 Targets.",
    );
  await expect.element(willStart()).toBeVisible();

  await userEvent.click(sheet().getByRole("switch", { name: "Remember these choices" }));
  await expect.element(willStart()).not.toBeInTheDocument();
  await userEvent.click(sheet().getByRole("button", { name: "Add Film" }));
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  expect(server.posted).toEqual([
    {
      contentType: "cinema",
      key: "9000001",
      profileId: ultra.id,
      rootId: filmsRoot.id,
      monitor: false,
      remember: false,
    },
  ]);
});

test("with two roots the sheet offers each with its free space", async () => {
  const server = addServer({ roots: [filmsRoot, shelfRoot] });
  renderApp({ path: quietPage, handlers: server.handlers });
  await openSheet();

  await userEvent.click(sheet().getByRole("combobox", { name: "Library root" }));
  expect(
    page
      .getByRole("option")
      .all()
      .map((option) => option.element().textContent),
  ).toEqual(["/srv/media/films · 1.2 TB free", "/mnt/shelf/films · Free space not read yet"]);
  await userEvent.click(page.getByRole("option", { name: /^\/mnt\/shelf/u }));
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Adds The Quiet Orchard (2021) to Films on /mnt/shelf/films, monitored, wanting Films' 1 Target.",
    );
  await expect.element(willStart()).toBeVisible();
});

test("the sheet starts unmonitored when the type adds unmonitored, and moves nothing", async () => {
  const server = addServer({ monitor: false });
  renderApp({ path: quietPage, handlers: server.handlers });
  await openSheet();
  await expect.element(sheet().getByRole("switch", { name: "Monitor" })).not.toBeChecked();
  await expect
    .element(sentence())
    .toHaveTextContent(
      "Adds The Quiet Orchard (2021) to Films on /srv/media/films, not monitored, wanting Films' 1 Target.",
    );
  expect(willStart().query()).toBeNull();
});

test("an add that finds the Work already there opens it and says so", async () => {
  const server = addServer({ created: false });
  const { router } = renderApp({ path: quietPage, handlers: server.handlers });
  await openSheet();
  await userEvent.click(sheet().getByRole("button", { name: "Add Film" }));
  await expect
    .element(notifications().getByText("The Quiet Orchard is already in the library."))
    .toBeVisible();
  expect(router.state.location.href).toBe(`/works/${orchardId}`);
});

test("a preview the catalog can't give says why in the sheet", async () => {
  const server = addServer();
  renderApp({
    path: quietPage,
    handlers: [
      http.get("*/api/v1/content-types/:contentType/catalog/:key", () =>
        problem(502, "metadata_unavailable", "TMDB isn't responding. Try again later."),
      ),
      ...server.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Add The Quiet Orchard" }));
  await expect
    .element(sheet().getByRole("alert"))
    .toHaveTextContent("TMDB isn't responding. Try again later.");
  expect(sheet().getByRole("button", { name: "Add Film" }).query()).toBeNull();
});

test("a failed Add says the server's reason in a toast", async () => {
  const server = addServer();
  renderApp({
    path: quietPage,
    handlers: [
      http.post("*/api/v1/works", () =>
        problem(
          502,
          "metadata_unavailable",
          "The metadata source didn't respond. Try again later.",
        ),
      ),
      ...server.handlers,
    ],
  });
  await openSheet();
  await userEvent.click(sheet().getByRole("button", { name: "Add Film" }));
  await expect
    .element(notifications().getByRole("dialog"))
    .toHaveTextContent("FailedThe metadata source didn't respond. Try again later.");
  await expect.element(sheet().getByRole("button", { name: "Add Film" })).toBeEnabled();
});

test.each([
  ["a disabled", "/add?contentType=cinema"],
  ["an unknown", "/add?contentType=nothing"],
])("%s Content Type has no Add page", async (_kind, path) => {
  renderApp({
    path,
    handlers: [getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] })],
  });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
  await expect.element(page.getByRole("heading", { name: /^Add/u })).not.toBeInTheDocument();
});
