import { expect, test } from "vitest";

import { types, view } from "../test/films";
import { decodeView, encodeView } from "./view-codec";
import { type LibraryView, defaultView } from "./view";

test("a view goes into the address and comes back the same, and the default view adds nothing", () => {
  const changed: LibraryView = {
    layout: "table",
    size: "large",
    columns: ["status", "size", "field.year"],
    sort: "field.year",
    reverse: true,
    filters: {
      status: { kind: "values", values: ["missing", "failed"] },
      root: { kind: "values", values: ["none"] },
      needs: { kind: "flag" },
      "field.year": { kind: "range", min: 1990, max: undefined },
      "field.genres": { kind: "values", values: ["Crime, Noir", "Drama"] },
    },
    q: "amelie",
  };
  const search = encodeView(changed, types);
  expect(search).toEqual({
    view: "table",
    size: "large",
    cols: "status,size,field.year",
    sort: "field.year",
    rev: 1,
    status: "missing,failed",
    root: "none",
    needs: 1,
    "f.year": "1990..",
    "f.genres": "Crime, Noir~Drama",
    q: "amelie",
  });
  expect(decodeView(search, types)).toEqual(changed);
  expect(encodeView(defaultView(types), types)).toEqual({});
});

test("an address naming what its types no longer declare opens on the rest", () => {
  const search = {
    view: "mosaic",
    sort: "field.genres",
    cols: "field.studio,size",
    "f.studio": "Fennel Films",
    "f.year": "2001",
  };
  expect(decodeView(search, types)).toEqual(
    view({
      columns: ["size"],
      filters: { "field.year": { kind: "range", min: 2001, max: 2001 } },
    }),
  );
});
