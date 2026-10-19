import { expect, test } from "vitest";

import {
  type BrowserAction,
  type BrowserState,
  browse,
  isPath,
  startBrowsing,
} from "./browser-state";

const run = (state: BrowserState, ...actions: readonly BrowserAction[]) =>
  actions.reduce((next, action) => browse(next, action), state);

const tv = String.raw`D:\Media\TV`;
const anime = String.raw`D:\Media\TV\Anime`;
const media = String.raw`D:\Media`;
const shows = [String.raw`D:\Media\TV\Akira`, anime, String.raw`D:\Media\TV\Bebop`];

test("opening a folder clears the field and makes the first row active once it lists", () => {
  const state = run(
    startBrowsing(media, false),
    { type: "type", text: "tv" },
    { type: "open", path: tv },
  );
  expect(state).toEqual(startBrowsing(tv, false));
  expect(browse(state, { type: "listed", path: tv, paths: shows }).active).toBe(0);
});

test("going up makes the folder you came from active once the parent lists", () => {
  const state = run(
    startBrowsing(anime, false),
    { type: "open", path: tv, cameFrom: anime },
    { type: "listed", path: tv, paths: shows },
  );
  expect(state.active).toBe(1);
  expect(state.cameFrom).toBeNull();
});

test("a listing for a folder the browser has left changes nothing", () => {
  const state = run(startBrowsing(anime, false), { type: "open", path: tv, cameFrom: anime });
  expect(browse(state, { type: "listed", path: anime, paths: [] })).toBe(state);
});

test("a folder you came from that no longer lists falls back to the first row", () => {
  const state = run(
    startBrowsing(anime, false),
    { type: "open", path: tv, cameFrom: anime },
    { type: "listed", path: tv, paths: [shows[0] ?? ""] },
  );
  expect(state.active).toBe(0);
});

test("an empty listing has no active row", () => {
  expect(browse(startBrowsing(tv, false), { type: "listed", path: tv, paths: [] }).active).toBe(-1);
});

test("a relisting keeps the active row while it still exists", () => {
  const listed = run(
    startBrowsing(tv, false),
    { type: "listed", path: tv, paths: shows },
    { type: "move", to: "last", count: 3, page: 10 },
  );
  expect(browse(listed, { type: "listed", path: tv, paths: shows }).active).toBe(2);
  expect(browse(listed, { type: "listed", path: tv, paths: shows.slice(0, 2) }).active).toBe(0);
});

test("moves go by index and stop at the ends", () => {
  const state = run(startBrowsing(tv, false), { type: "listed", path: tv, paths: shows });
  expect(browse(state, { type: "move", to: "next", count: 3, page: 10 }).active).toBe(1);
  expect(browse(state, { type: "move", to: "prev", count: 3, page: 10 }).active).toBe(0);
  expect(browse(state, { type: "move", to: "pageDown", count: 3, page: 10 }).active).toBe(2);
});

test("typing makes the first match active and forgets where you came from", () => {
  const state = run(
    startBrowsing(tv, false),
    { type: "listed", path: tv, paths: shows },
    { type: "move", to: "last", count: 3, page: 10 },
    { type: "type", text: "an" },
  );
  expect(state).toMatchObject({ field: "an", active: 0, cameFrom: null });
});

test("clearing empties the field and returns to the first row", () => {
  expect(
    run(startBrowsing(tv, false), { type: "type", text: "an" }, { type: "clear" }),
  ).toMatchObject({
    field: "",
    active: 0,
  });
});

test("Show hidden toggles and survives opening a folder", () => {
  const state = run(
    startBrowsing(tv, false),
    { type: "toggleHidden" },
    { type: "open", path: anime },
  );
  expect(state.showHidden).toBe(true);
  expect(browse(state, { type: "toggleHidden" }).showHidden).toBe(false);
});

test("New folder adds a named row, clears the filter, and keeps a refusal until renamed", () => {
  const opened = run(startBrowsing(tv, false), { type: "type", text: "an" }, { type: "newFolder" });
  expect(opened).toMatchObject({ field: "", creating: { name: "New folder", error: null } });

  const refused = run(
    opened,
    { type: "nameFolder", name: "CON" },
    { type: "folderRefused", error: "Windows reserves the name CON" },
  );
  expect(refused.creating).toEqual({ name: "CON", error: "Windows reserves the name CON" });
  expect(browse(refused, { type: "nameFolder", name: "Movies" }).creating).toEqual({
    name: "Movies",
    error: null,
  });
  expect(browse(refused, { type: "cancelNewFolder" }).creating).toBeNull();
});

test("a second New folder keeps the row being named", () => {
  const naming = run(
    startBrowsing(tv, false),
    { type: "newFolder" },
    { type: "nameFolder", name: "Movies" },
  );
  expect(browse(naming, { type: "newFolder" })).toBe(naming);
});

test("Places has no New folder, and naming without a row changes nothing", () => {
  const places = startBrowsing("", false);
  expect(browse(places, { type: "newFolder" })).toBe(places);
  expect(browse(places, { type: "nameFolder", name: "Movies" })).toBe(places);
  expect(browse(places, { type: "folderRefused", error: "no" })).toBe(places);
});

test("opening the new folder closes its row", () => {
  const state = run(startBrowsing(tv, false), { type: "newFolder" }, { type: "open", path: anime });
  expect(state.creating).toBeNull();
});

test.each([
  ["/", true],
  ["/mnt/media", true],
  ["media/", true],
  [String.raw`\\nas\media`, true],
  [String.raw`\\nas`, true],
  [String.raw`D:\Media`, true],
  ["D:", true],
  ["c:", true],
  ["Anime", false],
  ["", false],
  ["Season 1: Pilot", false],
  ["~", false],
])("%j is a path: %s", (text, want) => {
  expect(isPath(text)).toBe(want);
});
