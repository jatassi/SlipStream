import { useEffect, useState } from "react";

import { CommandPalette, CommandSearch, type PaletteGroup } from "../../ui/command-palette";
import { addLabel, enabledTypes } from "../../works/fields";
import { useAdminCommands, useCommandContext } from "../commands";
import { useContentTypes } from "../nav";
import {
  closePalette,
  isPaletteShortcut,
  openPalette,
  togglePalette,
  usePaletteOpen,
} from "../palette-state";
import { matches } from "./match";
import { addGroup, closingThen, workOptions } from "./options";
import { useCatalogResults, useFoundWorks } from "./provider";

const label = "Search Works or run a command";

// A closing dialog, the palette's own included, stays mounted through its exit animation.
const anotherModalOpen = (): boolean =>
  document.querySelector('[role="dialog"][aria-modal="true"]:not([data-closed])') !== null;

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

function Results() {
  const [query, setQuery] = useState("");
  const context = useCommandContext();
  const contentTypes = useContentTypes();
  const catalogs = useCatalogResults(query, enabledTypes(contentTypes));
  const groups: PaletteGroup[] = [
    { label: "Works", items: workOptions(useFoundWorks(query), contentTypes, context) },
    ...catalogs
      .filter((results) => results.candidates.length > 0)
      .map((results) => addGroup(results, context)),
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
