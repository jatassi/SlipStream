import { FilterHorizontalIcon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { SearchField } from "../ui/search-field";
import { Segmented } from "../ui/segmented";
import { Select, type SelectOption } from "../ui/select";
import { Sheet } from "../ui/sheet";
import { useShell } from "../ui/shell-context";
import { type Row, type Tab, tabLabels, tabOrder } from "./model";
import { type Choice, useChoices } from "./toolbar-choices";

const every = "";

export type ToolbarProps = {
  readonly tab: Tab;
  readonly counts: (tab: Tab) => number;
  readonly rows: readonly Row[];
  readonly node: string | undefined;
  readonly type: string | undefined;
  readonly client: string | undefined;
  readonly q: string;
  readonly onQChange: (q: string) => void;
  readonly onChange: (change: {
    tab?: Tab;
    node?: string | undefined;
    type?: string | undefined;
    client?: string | undefined;
  }) => void;
};

const optionsOf = (choices: readonly Choice[], all: string): SelectOption<string>[] => [
  { value: every, label: all },
  ...choices.map(({ id, name }) => ({ value: id, label: name })),
];

const chosen = (value: string) => (value === every ? undefined : value);

const shows = (choices: readonly Choice[], picked: string | undefined): boolean =>
  choices.length > 1 || picked !== undefined;

function Filters({
  nodes,
  types,
  clients,
  node,
  type,
  client,
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
      {shows(clients, client) && (
        <Select
          label="Client"
          options={optionsOf(clients, "Every client")}
          value={client ?? every}
          onValueChange={(value) => {
            onChange({ client: chosen(value) });
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
  const choices = useChoices(props.rows, props);
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
    const filtering =
      shows(choices.nodes, props.node) ||
      shows(choices.types, props.type) ||
      shows(choices.clients, props.client);
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
