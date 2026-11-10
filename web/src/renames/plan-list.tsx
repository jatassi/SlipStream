import { useState } from "react";

import type { FilesWorkLine } from "../api/gen/model";
import { SearchField } from "../ui/search-field";
import { useShell } from "../ui/shell-context";
import { VirtualRows } from "../ui/virtual-rows";
import { Reading } from "./reading";
import type { useReview } from "./round";
import { FileRows, WorkRow } from "./work-rows";

// A search box earns its place once the list is longer than a glance.
const searchFrom = 8;

const rowHeights = { desktop: 60, phone: 64 } as const;

type Review = ReturnType<typeof useReview>;

function PlanWork({
  line,
  review: { ticks, setTicks, opened, setOpen, rows, stateOf },
}: {
  line: FilesWorkLine;
  review: Review;
}) {
  const known = rows.get(line.workId);
  return (
    <WorkRow
      line={line}
      rows={rows}
      ticks={ticks}
      open={opened.has(line.workId)}
      onOpenChange={(open) => {
        setOpen(line.workId, open);
      }}
      onTicks={setTicks}
    >
      {known === undefined ? (
        <Reading
          label={`Reading the files of ${line.title}`}
          {...stateOf(line.workId)}
          className="mb-2 ml-7"
        />
      ) : (
        <FileRows line={line} rows={known} ticks={ticks} onTicks={setTicks} />
      )}
    </WorkRow>
  );
}

/** The plan's Works, searchable by title, each ticked and opening to its files. */
export function PlanList({ lines, review }: { lines: readonly FilesWorkLine[]; review: Review }) {
  const shell = useShell();
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();
  const shown =
    needle === "" ? lines : lines.filter((line) => line.title.toLowerCase().includes(needle));
  return (
    <>
      {lines.length > searchFrom && (
        <SearchField
          label="Search Works"
          placeholder="Title"
          value={query}
          onValueChange={setQuery}
        />
      )}
      <div
        className={`flex flex-col rounded-card bg-glass px-3 ring-1 ring-hairline ring-inset ${shell === "phone" ? "min-h-0 flex-1" : "max-h-[min(56vh,560px)]"}`}
      >
        {shown.length === 0 ? (
          <p className="py-3 text-ink-3">No Work matches &quot;{query.trim()}&quot;.</p>
        ) : (
          <VirtualRows
            label="Works"
            rows={shown}
            rowKey={(line) => line.workId}
            estimate={rowHeights[shell]}
            renderRow={(line) => <PlanWork line={line} review={review} />}
          />
        )}
      </div>
    </>
  );
}
