import { useEffect, useState } from "react";

import type { WorksContentType } from "../../api/gen/model";
import {
  CommandPalette,
  CommandSearch,
  type PaletteGroup,
  type PaletteOption,
} from "../../ui/command-palette";
import { addLabel } from "../../works/fields";
import { type CommandContext, useAdminCommands, useCommandContext } from "../commands";
import { contentTypeIcons } from "../icons";
import { useContentTypes } from "../nav";
import {
  closePalette,
  isPaletteShortcut,
  openPalette,
  togglePalette,
  usePaletteOpen,
} from "../palette-state";
import { matches } from "./match";
import { type FoundWork, libraryProvider } from "./provider";

const label = "Search Works or run a command";

const anotherModalOpen = (): boolean =>
  document.querySelector('[role="dialog"][aria-modal="true"]') !== null;

export function Palette() {
  const open = usePaletteOpen();
  // A palette left open when the session ends must not greet the next one.
  useEffect(() => closePalette, []);
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!isPaletteShortcut(event) || (!open && anotherModalOpen())) {
        return;
      }
      event.preventDefault();
      togglePalette();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);
  return (
    <CommandPalette
      label={label}
      open={open}
      onOpenChange={(next) => {
        (next ? openPalette : closePalette)();
      }}
    >
      <Results />
    </CommandPalette>
  );
}

const closingThen = (run: () => void) => () => {
  closePalette();
  run();
};

function workOptions(
  works: readonly FoundWork[],
  contentTypes: readonly WorksContentType[],
  { navigate }: CommandContext,
): PaletteOption[] {
  const types = new Map(contentTypes.map((type) => [type.id, type]));
  return works.flatMap((work) => {
    const type = types.get(work.contentType);
    return type === undefined
      ? []
      : [
          {
            id: work.id,
            label: work.label,
            meta: type.work.one,
            icon: contentTypeIcons[type.icon],
            choose: closingThen(() => {
              void navigate({ to: "/works/$workId", params: { workId: work.id } });
            }),
          },
        ];
  });
}

function Results() {
  const [query, setQuery] = useState("");
  const context = useCommandContext();
  const contentTypes = useContentTypes();
  const groups: PaletteGroup[] = [
    { label: "Works", items: workOptions(libraryProvider.useWorks(query), contentTypes, context) },
    {
      label: "Commands",
      items: useAdminCommands()
        .filter((command) => matches(query, command.label))
        .map((command) => ({
          id: command.id,
          label: command.label,
          meta: command.meta,
          icon: command.icon,
          choose: closingThen(() => {
            command.run(context);
          }),
        })),
    },
  ].filter((group) => group.items.length > 0);

  const firstType = contentTypes.find((type) => type.enabled);
  const example = firstType === undefined ? "Go to Library" : addLabel(firstType);
  return (
    <CommandSearch
      label={label}
      query={query}
      onQueryChange={setQuery}
      groups={groups}
      empty={`Nothing matches "${query.trim()}". Try a Work's name, or a command such as ${example}.`}
    />
  );
}
