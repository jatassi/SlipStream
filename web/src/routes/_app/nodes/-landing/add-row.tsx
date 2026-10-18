import { Add01Icon } from "@hugeicons/core-free-icons";

import { Icon } from "../../../../ui/icon";
import type { Adding } from "../-add-node/add-node";
import { rowKeys, rowShell, rowWash } from "./rows";

/** With one machine, the Roster offers the next one in its own row. */
export function AddRow({ adding }: { adding: Adding }) {
  return (
    <li className={rowShell}>
      <button
        type="button"
        {...rowKeys}
        onClick={adding.start}
        className={`flex min-h-(--row-height-two) w-full items-center gap-3 px-4 py-3 text-left ${rowWash}`}
      >
        <Icon icon={Add01Icon} className="size-4 shrink-0 text-ink-2" />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-(length:--body-size) leading-(--body-line-height) text-ink-1">
            Add a Node
          </span>
          <span className="text-meta text-ink-3">
            Pair another machine to search, download or serve from it.
          </span>
        </span>
      </button>
    </li>
  );
}

/** With one machine, the Diagram offers the next one beside its tile, in the Roster row's words. */
export function AddOnDiagram({ adding }: { adding: Adding }) {
  return (
    <button
      type="button"
      onClick={adding.start}
      className="absolute right-4 bottom-4 flex max-w-72 items-center gap-3 rounded-card bg-popover px-4 py-3 text-left shadow-float ring-1 ring-hairline backdrop-blur-[30px] backdrop-saturate-[1.7] transition-colors duration-(--duration-control) ease-standard hover:bg-fill focus-visible:bg-fill"
    >
      <Icon icon={Add01Icon} className="size-4 shrink-0 text-ink-2" />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="text-(length:--body-size) leading-(--body-line-height) text-ink-1">
          Add a Node
        </span>
        <span className="text-meta text-ink-3">
          Pair another machine to search, download or serve from it.
        </span>
      </span>
    </button>
  );
}
