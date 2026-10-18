import { Add01Icon } from "@hugeicons/core-free-icons";

import { Button } from "../../../../ui/button";
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

/** With one machine, the Diagram offers the next one beside its tile. */
export function AddOnDiagram({ adding }: { adding: Adding }) {
  return (
    <div className="absolute right-4 bottom-4">
      <Button icon={Add01Icon} onClick={adding.start}>
        Add a Node
      </Button>
    </div>
  );
}
