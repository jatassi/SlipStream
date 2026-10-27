import { Link } from "@tanstack/react-router";

import type { IndexersIndexerCatalog } from "../../api/gen/model";
import { Data } from "../../ui/data";
import { DeleteDefinition } from "./delete-definition";

const addCustom = (
  <Link to="/settings/indexers/new/custom" className="text-ink-1 underline">
    Add a custom definition
  </Link>
);

/** The definitions pasted or uploaded in Add Indexer, each deletable while no Indexer runs it. */
export function CustomDefinitions({ catalog }: { catalog: IndexersIndexerCatalog }) {
  const custom = catalog.entries.filter((entry) => entry.origin === "custom");
  const replaced = new Set(
    catalog.entries.filter((entry) => entry.replacedByCustom).map((entry) => entry.id),
  );
  if (custom.length === 0) {
    return (
      <p className="text-ink-2">
        No custom definitions. A custom definition runs a site the catalog lacks. {addCustom}
      </p>
    );
  }
  return (
    <div className="flex flex-col items-start gap-2">
      <ul
        aria-label="Custom definitions"
        className="w-full rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {custom.map((entry) => (
          <li
            key={entry.id}
            className="flex min-h-(--row-height) flex-wrap items-center gap-x-3 px-4 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]"
          >
            <span className="font-semibold text-ink-1">{entry.name}</span>
            <Data className="min-w-0 flex-1 truncate text-ink-3">{entry.id}</Data>
            {replaced.has(entry.id) && (
              <span className="text-ink-3">Runs in place of the catalog&apos;s {entry.name}</span>
            )}
            <DeleteDefinition entry={entry} replaces={replaced.has(entry.id)} />
          </li>
        ))}
      </ul>
      {addCustom}
    </div>
  );
}
