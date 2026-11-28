import type { ConversionsCounts } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { TypeFilter } from "../grabs/jobs";
import { SearchField } from "../ui/search-field";
import { Segmented } from "../ui/segmented";
import { Select } from "../ui/select";
import type { Filter } from "./load";

type Shown = NonNullable<Filter["band"]> | "all";

const bands: readonly {
  value: Shown;
  label: string;
  count: (counts: ConversionsCounts) => number;
}[] = [
  {
    value: "all",
    label: "All",
    count: (c) => c.failed + c.converting + c.paused + c.waiting,
  },
  { value: "attention", label: "Needs attention", count: (c) => c.failed },
  { value: "converting", label: "Converting", count: (c) => c.converting + c.paused },
  { value: "next", label: "Up next", count: (c) => c.waiting },
];

const every = "";

function NodeFilter({ node, onChange }: { node: string | undefined; onChange: Change }) {
  const nodes = useListNodesSuspense().data.nodes.filter(
    ({ id, roles }) => id === node || roles.some(({ role }) => role === "converting"),
  );
  if (nodes.length < 2 && node === undefined) {
    return null;
  }
  return (
    <Select
      label="Node"
      options={[
        { value: every, label: "Every Node" },
        ...nodes.map(({ id, name }) => ({ value: id, label: name })),
      ]}
      value={node ?? every}
      onValueChange={(value) => {
        onChange({ node: value === every ? undefined : value });
      }}
    />
  );
}

type Change = (change: Partial<Filter>) => void;

export function Toolbar({
  counts,
  search,
  draft,
  onDraft,
  onChange,
}: {
  counts: ConversionsCounts;
  search: Filter;
  draft: string;
  onDraft: (q: string) => void;
  onChange: Change;
}) {
  const types = useListContentTypesSuspense().data.contentTypes;
  return (
    <div className="my-4 flex flex-wrap items-end gap-3">
      <Segmented
        label="Show"
        options={bands.map(({ value, label, count }) => ({ value, label, count: count(counts) }))}
        value={search.band ?? "all"}
        onValueChange={(band) => {
          onChange({ band: band === "all" ? undefined : band });
        }}
      />
      <NodeFilter node={search.node} onChange={onChange} />
      {(types.length > 1 || search.type !== undefined) && (
        <TypeFilter
          types={types}
          type={search.type}
          onTypeChange={(type) => {
            onChange({ type });
          }}
        />
      )}
      <SearchField
        label="Search Conversions"
        placeholder="Search by title"
        value={draft}
        onValueChange={onDraft}
        className="ml-auto w-72 max-w-full"
      />
    </div>
  );
}
