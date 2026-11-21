import type { ReactNode } from "react";

/** Where the empty sits, which sets its surface. */
export type EmptyLook = "sheet" | "row" | "inline";

const surfaces: { readonly [L in EmptyLook]: string } = {
  sheet: "min-h-(--row-height) rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset",
  row: "min-h-(--row-height) px-4 py-3",
  inline: "",
};

/** What isn't here, and the way forward when there is one. In Touch the action sits under the sentence. */
export function Empty({
  look,
  action,
  children,
}: {
  readonly look: EmptyLook;
  /** The way forward: one button, a button group or a link. */
  readonly action?: ReactNode;
  /** What is missing, then what to do or what SlipStream does next (WRITING.md rule 1). */
  readonly children: ReactNode;
}) {
  return (
    <div
      className={`flex flex-wrap items-center gap-x-4 gap-y-3 in-data-[density=touch]:flex-col in-data-[density=touch]:items-start ${surfaces[look]}`}
    >
      <p className="max-w-[64ch] text-(length:--body-size) leading-(--body-line-height) text-ink-2">
        {children}
      </p>
      {action}
    </div>
  );
}
