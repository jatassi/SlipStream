import type { ReactNode } from "react";

import type { WorksContentType } from "../api/gen/model";
import { bytes } from "../format";
import { Data } from "../ui/data";
import type { LibraryWork } from "./rows";
import { dateText, partialDateText, valueText } from "./values";
import {
  type ArrangeContext,
  type ColumnKey,
  type CoreColumn,
  columnKeys,
  isFieldKey,
  keyOfField,
  monitoredWords,
  noRoot,
  sharedFields,
} from "./view";
import { WorkStatus } from "./work-status";

export type Names = Pick<ArrangeContext, "profiles" | "roots">;

export type ColumnDef = {
  readonly key: ColumnKey;
  readonly header: string;
  /** Its share of the row, as a grid track. */
  readonly track: string;
  readonly align?: "end";
  readonly cell: (work: LibraryWork, names: Names) => ReactNode;
};

const ratingText = ({ value, scale }: NonNullable<LibraryWork["rating"]>) =>
  `${value.toFixed(scale > 10 ? 0 : 1)}/${scale}`;

const coreColumns: { readonly [K in CoreColumn]: Omit<ColumnDef, "key"> } = {
  status: {
    header: "Status",
    track: "minmax(0, 1.6fr)",
    cell: (work) => <WorkStatus work={work} />,
  },
  monitored: {
    header: "Monitored",
    track: "minmax(0, 1fr)",
    cell: (work) => monitoredWords[work.monitored],
  },
  profile: {
    header: "Profile",
    track: "minmax(0, 1fr)",
    cell: (work, { profiles }) => profiles.get(work.profileId),
  },
  root: {
    header: "Library root",
    track: "minmax(0, 1.4fr)",
    cell: (work, { roots }) => {
      const path = work.rootId === undefined ? undefined : roots.get(work.rootId);
      return path === undefined ? (
        <span className="text-ink-3">{noRoot}</span>
      ) : (
        <Data>{path}</Data>
      );
    },
  },
  size: {
    header: "Size on disk",
    track: "minmax(0, 0.9fr)",
    align: "end",
    cell: (work) => (work.files === 0 ? undefined : <Data>{bytes(work.bytes)}</Data>),
  },
  files: {
    header: "Files",
    track: "minmax(0, 0.6fr)",
    align: "end",
    cell: (work) => <span className="tabular-nums">{work.files}</span>,
  },
  added: {
    header: "Added",
    track: "minmax(0, 1fr)",
    cell: (work) => <Data>{dateText(work.added)}</Data>,
  },
  available: {
    header: "Available",
    track: "minmax(0, 1fr)",
    cell: (work) =>
      work.availableOn === undefined ? undefined : <Data>{partialDateText(work.availableOn)}</Data>,
  },
  rating: {
    header: "Rating",
    track: "minmax(0, 0.7fr)",
    align: "end",
    cell: (work) =>
      work.rating === undefined ? undefined : <Data>{ratingText(work.rating)}</Data>,
  },
};

const numeric = new Set(["number", "duration"]);

/** The columns a table can show, in the order they stand: the core ones, then each listed field. */
export function columnsFor(types: readonly WorksContentType[]): readonly ColumnDef[] {
  const fields = new Map(sharedFields(types, "column").map((field) => [field.key, field]));
  return columnKeys(types).flatMap((key): ColumnDef[] => {
    if (!isFieldKey(key)) {
      return [{ key, ...coreColumns[key] }];
    }
    const field = fields.get(keyOfField(key));
    if (field === undefined) {
      return [];
    }
    return [
      {
        key,
        header: field.label,
        track: "minmax(0, 0.8fr)",
        ...(numeric.has(field.kind) && { align: "end" }),
        cell: (work) => {
          const value = work.values.get(field.key);
          return value === undefined ? undefined : valueText(value);
        },
      },
    ];
  });
}
