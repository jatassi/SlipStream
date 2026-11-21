import type { ReactElement } from "react";

import { ArtworkFrame } from "./artwork";
import { useShell } from "./shell-context";

/** The loaded layout a skeleton stands in for, so the page doesn't jump when the data lands. */
export type SkeletonShape = "rows" | "table" | "grid" | "facts" | "form" | "hero";

type Lines = 1 | 2;

/** The Library grid's poster sizes, each a narrowest column width. */
export type Posters = "small" | "medium" | "large";

export type SkeletonProps = { readonly label: string } & (
  | {
      readonly shape: "rows";
      /** The loaded List's rows: 1 for a title, 2 for a title and a meta line. */
      readonly lines: Lines;
      /** The caller already draws the list's glass frame, as the folder browser does. */
      readonly inFrame?: true;
    }
  | { readonly shape: "table"; readonly lines: Lines }
  | { readonly shape: "grid"; readonly posters: Posters }
  | { readonly shape: "facts" | "form" | "hero" }
);

type Fit = {
  readonly lines: Lines;
  readonly inFrame: true | undefined;
  readonly posters: Posters;
};

/** One placeholder bar, breathing between fill and fill-hover. */
export const skeletonBar = "block rounded-chip bg-fill animate-skeleton motion-reduce:animate-none";

const sheet = "rounded-card bg-glass ring-1 ring-hairline ring-inset";

const ruled =
  "relative not-first:before:absolute not-first:before:inset-x-0 not-first:before:top-0 not-first:before:h-px not-first:before:bg-hairline in-data-[density=touch]:not-first:before:left-4";

const rowHeight: { readonly [L in Lines]: string } = {
  1: "h-(--row-height)",
  2: "h-(--row-height-two)",
};

/** How many rows, cards or fields each shape draws. No caller knows its count before the data lands. */
const counts: { readonly [S in Exclude<SkeletonShape, "hero">]: number } = {
  rows: 6,
  table: 8,
  grid: 12,
  facts: 4,
  form: 3,
};

// Varied title widths, so the rows don't read as a barcode. Each is also its row's key.
const titleWidths = [
  "w-[56%]",
  "w-[40%]",
  "w-[48%]",
  "w-[32%]",
  "w-[52%]",
  "w-[44%]",
  "w-[36%]",
  "w-[50%]",
] as const;

const titles = (shape: keyof typeof counts): readonly string[] =>
  titleWidths.slice(0, counts[shape]);

const keysOf = (shape: keyof typeof counts): readonly string[] =>
  Array.from({ length: counts[shape] }, (_, index) => `${shape}-${index}`);

function Rows({ lines, inFrame }: Fit) {
  return (
    <div className={inFrame ? "" : sheet}>
      {titles("rows").map((width) => (
        <div
          key={width}
          className={`flex flex-col justify-center gap-2 px-4 ${rowHeight[lines]} ${ruled}`}
        >
          <span className={`h-3 ${width} ${skeletonBar}`} />
          {lines === 2 && <span className={`h-2.5 w-1/4 ${skeletonBar}`} />}
        </div>
      ))}
    </div>
  );
}

const tableColumns = "grid grid-cols-[2fr_1fr_1fr] items-center gap-4 px-4";

function Table({ lines }: Fit) {
  return (
    <div className={sheet}>
      <div className={`h-8 ${tableColumns}`}>
        {["name", "first", "second"].map((key) => (
          <span key={key} className={`h-2.5 w-16 ${skeletonBar}`} />
        ))}
      </div>
      {titles("table").map((width) => (
        <div
          key={width}
          className={`${tableColumns} ${rowHeight[lines]} relative before:absolute before:inset-x-0 before:top-0 before:h-px before:bg-hairline`}
        >
          <span className={`h-3 ${width} ${skeletonBar}`} />
          <span className={`h-3 w-1/2 ${skeletonBar}`} />
          <span className={`h-3 w-1/3 ${skeletonBar}`} />
        </div>
      ))}
    </div>
  );
}

const posterColumns: { readonly [P in Posters]: string } = {
  small: "grid-cols-[repeat(auto-fill,minmax(120px,1fr))]",
  medium: "grid-cols-[repeat(auto-fill,minmax(160px,1fr))]",
  large: "grid-cols-[repeat(auto-fill,minmax(240px,1fr))]",
};

function Grid({ posters }: Fit) {
  return (
    <div className={`grid gap-4 ${posterColumns[posters]}`}>
      {keysOf("grid").map((key) => (
        <div key={key} className="flex flex-col gap-2">
          <ArtworkFrame size="card" />
          <span className={`h-3 w-3/4 ${skeletonBar}`} />
          <span className={`h-2.5 w-1/3 ${skeletonBar}`} />
        </div>
      ))}
    </div>
  );
}

function Facts() {
  return (
    <div className={sheet}>
      {titles("facts").map((width) => (
        <div
          key={width}
          className={`grid h-(--row-height) grid-cols-[minmax(120px,200px)_minmax(0,1fr)] items-center gap-x-4 px-4 in-data-[density=touch]:h-(--row-height-two) in-data-[density=touch]:grid-cols-1 in-data-[density=touch]:content-center in-data-[density=touch]:gap-y-2 ${ruled}`}
        >
          <span className={`h-3 w-20 ${skeletonBar}`} />
          <span className={`h-3 ${width} ${skeletonBar}`} />
        </div>
      ))}
    </div>
  );
}

function Form() {
  return (
    <div className="flex max-w-[480px] flex-col gap-4">
      {keysOf("form").map((key) => (
        <div key={key} className="flex flex-col gap-2">
          <span className={`h-3 w-24 ${skeletonBar}`} />
          <span className="block h-(--input-height) rounded-item bg-well ring-1 ring-hairline ring-inset" />
        </div>
      ))}
    </div>
  );
}

const heroLines = ["w-1/2 max-w-80", "w-1/3 max-w-56"] as const;

function Hero(fit: Fit) {
  return (
    <div className="flex flex-col gap-7">
      <div className="flex flex-col gap-5 pt-[clamp(24px,14vw,180px)] sm:flex-row sm:items-end">
        <ArtworkFrame size="hero" />
        <div className="flex min-w-0 flex-1 flex-col gap-3">
          <span className={`h-[1lh] w-2/3 max-w-96 text-title-large ${skeletonBar}`} />
          {heroLines.map((width) => (
            <span key={width} className={`h-3 ${width} ${skeletonBar}`} />
          ))}
        </div>
      </div>
      <Rows {...fit} lines={2} />
    </div>
  );
}

const shapes: { readonly [S in SkeletonShape]: (fit: Fit) => ReactElement } = {
  rows: Rows,
  table: Table,
  grid: Grid,
  facts: Facts,
  form: Form,
  hero: Hero,
};

/**
 * A placeholder in the loaded layout's frame. It appears after 300ms, so a
 * fallback that resolves sooner never flashes, and a Table draws as rows on
 * a phone, where tables turn into lists.
 */
export function Skeleton(props: SkeletonProps) {
  const phone = useShell() === "phone";
  const shape = props.shape === "table" && phone ? "rows" : props.shape;
  const Shape = shapes[shape];
  return (
    <output
      aria-label={props.label}
      className="block animate-skeleton-in motion-reduce:[animation-duration:0s]"
    >
      <Shape
        lines={"lines" in props ? props.lines : 1}
        inFrame={"inFrame" in props ? props.inFrame : undefined}
        posters={"posters" in props ? props.posters : "medium"}
      />
    </output>
  );
}
