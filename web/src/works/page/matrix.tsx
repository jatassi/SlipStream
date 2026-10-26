import type { ReactElement, ReactNode } from "react";

import type { WorksContentType } from "../../api/gen/model";
import { Button } from "../../ui/button";
import { Section } from "../../ui/section";
import { useShell } from "../../ui/shell-context";
import { CellBlock } from "./cell";
import type { Matrix, MatrixShown } from "./matrix-model";
import { RowHead } from "./row-head";

const unseenWords = "Not seen yet. It's wanted once a release of it is found.";

function MatrixTable({ matrix, shown }: { matrix: Matrix; shown: MatrixShown }) {
  return (
    <div className="overflow-x-auto rounded-card bg-glass ring-1 ring-hairline ring-inset">
      <table aria-label="Editions and Targets" className="w-full border-collapse text-left">
        <thead>
          <tr className="h-8 text-label text-ink-3">
            <th scope="col" className="w-[168px] px-4 font-medium">
              Edition
            </th>
            {matrix.targets.map((target) => (
              <th key={target.id} scope="col" className="px-4 font-medium">
                <span className="text-ink-1">{target.name}</span> {target.sourcing}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {matrix.rows.map((row) => (
            <tr
              key={row.edition}
              className="align-top shadow-[inset_0_1px_0_var(--color-hairline)]"
            >
              <th scope="row" className="w-[168px] px-4 py-3 text-left font-normal">
                <RowHead row={row} shown={shown} />
              </th>
              {row.kind === "unseen" ? (
                <td colSpan={matrix.targets.length} className="px-4 py-3 text-ink-3">
                  {unseenWords}
                </td>
              ) : (
                row.cells.map((view) => (
                  <td key={view.target.id} className="px-4 py-3">
                    <CellBlock view={view} row={row.name} shown={shown} />
                  </td>
                ))
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function MatrixCards({ matrix, shown }: { matrix: Matrix; shown: MatrixShown }) {
  return (
    <ul aria-label="Editions and Targets" className="flex flex-col gap-3">
      {matrix.rows.map((row) => (
        <li
          key={row.edition}
          aria-label={row.name}
          className="flex flex-col gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
        >
          <RowHead row={row} shown={shown} />
          {row.kind === "unseen" && <p className="text-ink-3">{unseenWords}</p>}
          {row.kind === "edition" &&
            row.cells.map((view) => (
              <div key={view.target.id} className="flex flex-col gap-1">
                <span className="text-label text-ink-3">
                  <span className="text-ink-1">{view.target.name}</span> {view.target.sourcing}
                </span>
                <CellBlock view={view} row={row.name} shown={shown} />
              </div>
            ))}
        </li>
      ))}
    </ul>
  );
}

/** The Edition by Target matrix: a table in the desktop shell, one card per Edition on phone. */
export function MatrixSection({
  type,
  matrix,
  shown,
  wantAnother,
}: {
  type: WorksContentType;
  matrix: Matrix;
  shown: MatrixShown;
  wantAnother: (trigger: ReactElement) => ReactNode;
}) {
  const phone = useShell() === "phone";
  if (type.items === "many") {
    return (
      <Section title="Editions and Targets">
        <p className="text-ink-3">The Item list arrives with series.</p>
      </Section>
    );
  }
  return (
    <Section title="Editions and Targets">
      {matrix.targets.length === 0 ? (
        <p className="text-ink-3">
          Its Profile has no Targets, so this {type.work.one.toLowerCase()} wants nothing.
        </p>
      ) : phone ? (
        <MatrixCards matrix={matrix} shown={shown} />
      ) : (
        <MatrixTable matrix={matrix} shown={shown} />
      )}
      <div className="mt-3">{wantAnother(<Button size="small">Want another Edition</Button>)}</div>
    </Section>
  );
}
