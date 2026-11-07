import { FilterHorizontalIcon } from "@hugeicons/core-free-icons";

import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { SearchField } from "../ui/search-field";
import { Segmented } from "../ui/segmented";
import { Select, type SelectOption } from "../ui/select";
import { Sheet } from "../ui/sheet";
import { useShell } from "../ui/shell-context";
import { type Row, type Tab, tabLabels, tabOrder } from "./model";

const every = "";

type Choice = { readonly id: string; readonly name: string };

export type ToolbarProps = {
  readonly tab: Tab;
  readonly counts: (tab: Tab) => number;
  readonly rows: readonly Row[];
  readonly node: string | undefined;
  readonly type: string | undefined;
  readonly q: string;
  readonly onQChange: (q: string) => void;
  readonly onChange: (change: {
    tab?: Tab;
    node?: string | undefined;
    type?: string | undefined;
  }) => void;
};

const optionsOf = (choices: readonly Choice[], all: string): SelectOption<string>[] => [
  { value: every, label: all },
  ...choices.map(({ id, name }) => ({ value: id, label: name })),
];

const chosen = (value: string) => (value === every ? undefined : value);

// The Nodes and Content Types the Torrents fall under, plus the one chosen,
// so a link to a Node holding none still reads its name.
function choicesOf(
  rows: readonly Row[],
  of: (row: Row) => string,
  named: readonly Choice[],
  picked: string | undefined,
): Choice[] {
  const held = new Set(rows.map((row) => of(row)));
  return named.filter(({ id }) => held.has(id) || id === picked);
}

function useChoices({ rows, node, type }: ToolbarProps) {
  return {
    nodes: choicesOf(rows, (row) => row.nodeId, useListNodesSuspense().data.nodes, node),
    types: choicesOf(
      rows,
      (row) => row.contentType,
      useListContentTypesSuspense().data.contentTypes,
      type,
    ),
  };
}

const shows = (choices: readonly Choice[], picked: string | undefined): boolean =>
  choices.length > 1 || picked !== undefined;

function Filters({
  nodes,
  types,
  node,
  type,
  onChange,
}: ToolbarProps & ReturnType<typeof useChoices>) {
  return (
    <>
      {shows(nodes, node) && (
        <Select
          label="Node"
          options={optionsOf(nodes, "Every Node")}
          value={node ?? every}
          onValueChange={(value) => {
            onChange({ node: chosen(value) });
          }}
        />
      )}
      {shows(types, type) && (
        <Select
          label="Content Type"
          options={optionsOf(types, "Every Content Type")}
          value={type ?? every}
          onValueChange={(value) => {
            onChange({ type: chosen(value) });
          }}
        />
      )}
    </>
  );
}

function Search({ q, onQChange, className }: ToolbarProps & { className: string }) {
  return (
    <SearchField
      label="Search torrents"
      placeholder="Search by Work or Release"
      value={q}
      onValueChange={onQChange}
      className={className}
    />
  );
}

export function Toolbar(props: ToolbarProps) {
  const shell = useShell();
  const choices = useChoices(props);
  const tabs = (
    <Segmented
      label="Show"
      options={tabOrder[shell].map((tab) => ({
        value: tab,
        label: tabLabels[tab],
        count: props.counts(tab),
      }))}
      value={props.tab}
      onValueChange={(tab) => {
        props.onChange({ tab });
      }}
    />
  );
  if (shell === "phone") {
    const filtering = shows(choices.nodes, props.node) || shows(choices.types, props.type);
    return (
      <div className="mb-4 flex flex-col gap-3">
        {tabs}
        <div className="flex gap-2">
          <Search {...props} className="min-w-0 flex-1" />
          {filtering && (
            <Sheet title="Filters" trigger={<Button icon={FilterHorizontalIcon}>Filter</Button>}>
              <div className="flex flex-col gap-4 pb-2">
                <Filters {...props} {...choices} />
              </div>
            </Sheet>
          )}
        </div>
      </div>
    );
  }
  return (
    <div className="mb-4 flex flex-wrap items-end gap-3">
      {tabs}
      <Filters {...props} {...choices} />
      <Search {...props} className="ml-auto w-72 max-w-full" />
    </div>
  );
}
