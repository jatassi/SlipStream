import {
  Eraser01Icon,
  FilterHorizontalIcon,
  PauseIcon,
  PlayIcon,
} from "@hugeicons/core-free-icons";

import { Button } from "../../../../ui/button";
import { SearchField } from "../../../../ui/search-field";
import { Segmented } from "../../../../ui/segmented";
import { Select } from "../../../../ui/select";
import { Sheet } from "../../../../ui/sheet";
import { useShell } from "../../../../ui/shell-context";
import { DownloadLogs } from "../-download";
import type { Level, LogEvent } from "./buffer";
import type { LogFilter } from "./filter";

const levelOptions: readonly { value: Level; label: string }[] = [
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "error", label: "Error" },
];

const everySubsystem = "";

type FiltersProps = {
  filter: LogFilter;
  onFilterChange: (filter: LogFilter) => void;
  subsystems: readonly string[];
};

function Filters({ filter, onFilterChange, subsystems }: FiltersProps) {
  return (
    <>
      <Segmented
        label="Minimum level"
        options={levelOptions}
        value={filter.min}
        onValueChange={(min) => {
          onFilterChange({ ...filter, min });
        }}
      />
      <Select
        label="Subsystem"
        options={[
          { value: everySubsystem, label: "All subsystems" },
          ...subsystems.map((name) => ({ value: name, label: name })),
        ]}
        value={filter.subsystem ?? everySubsystem}
        onValueChange={(subsystem) => {
          onFilterChange({ ...filter, subsystem: subsystem === everySubsystem ? null : subsystem });
        }}
      />
      <SearchField
        label="Search the log"
        placeholder="Search the log"
        value={filter.search}
        onValueChange={(search) => {
          onFilterChange({ ...filter, search });
        }}
      />
    </>
  );
}

function Actions({
  nodeId,
  paused,
  dispatch,
}: {
  nodeId: string;
  paused: boolean;
  dispatch: (event: LogEvent) => void;
}) {
  return (
    <>
      <Button
        icon={paused ? PlayIcon : PauseIcon}
        onClick={() => {
          dispatch({ type: paused ? "resume" : "pause" });
        }}
      >
        {paused ? "Resume" : "Pause"}
      </Button>
      <Button
        icon={Eraser01Icon}
        onClick={() => {
          dispatch({ type: "clear" });
        }}
      >
        Clear
      </Button>
      <DownloadLogs nodeId={nodeId} />
    </>
  );
}

/** The viewer's controls. On phone the filters wait in a sheet behind Filter. */
export function LogBar({
  nodeId,
  paused,
  dispatch,
  ...filters
}: FiltersProps & {
  nodeId: string;
  paused: boolean;
  dispatch: (event: LogEvent) => void;
}) {
  const actions = <Actions nodeId={nodeId} paused={paused} dispatch={dispatch} />;
  if (useShell() === "phone") {
    return (
      <div className="flex flex-wrap gap-2">
        <Sheet title="Filters" trigger={<Button icon={FilterHorizontalIcon}>Filter</Button>}>
          <div className="flex flex-col gap-4 pb-2">
            <Filters {...filters} />
          </div>
        </Sheet>
        {actions}
      </div>
    );
  }
  return (
    <div className="flex flex-wrap items-end gap-3">
      <Filters {...filters} />
      <div className="ml-auto flex gap-2">{actions}</div>
    </div>
  );
}
