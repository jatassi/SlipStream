import { type RefObject, useId, useRef } from "react";

import { Dialog } from "../ui/dialog";
import { useShell } from "../ui/shell-context";
import { Sheet } from "../ui/sheet";
import { DesktopBody, PhoneBody } from "./layouts";
import type { Parts } from "./parts";
import { type BrowserProps, useBrowser } from "./use-browser";
import type { FolderCheck } from "./verdict";

export type FolderBrowserProps = {
  readonly nodeId: string;
  readonly nodeName: string;
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  /** "" opens on Places. */
  readonly startPath: string;
  readonly check: FolderCheck;
  /** The main button's verb: "Use" reads "Use Anime". */
  readonly confirmVerb: string;
  readonly allowNewFolder: boolean;
  readonly onPick: (path: string) => void;
};

const rowHeights = { desktop: { one: 44, two: 60 }, phone: { one: 54, two: 64 } } as const;

/**
 * A one-pane drill-down through one Node's folders, read by that Node. A row
 * opens its folder; the main button uses the folder on show.
 */
export function FolderBrowser({ open, onOpenChange, ...props }: FolderBrowserProps) {
  const shell = useShell();
  const field = useRef<HTMLInputElement>(null);
  const title = `Choose a folder on ${props.nodeName}`;
  const body = (
    <Body
      {...props}
      field={field}
      close={() => {
        onOpenChange(false);
      }}
    />
  );
  return shell === "phone" ? (
    <Sheet label={title} fill open={open} onOpenChange={onOpenChange}>
      {body}
    </Sheet>
  ) : (
    <Dialog title={title} size={680} open={open} onOpenChange={onOpenChange} initialFocus={field}>
      {body}
    </Dialog>
  );
}

type BodyProps = Omit<FolderBrowserProps, "open" | "onOpenChange"> & {
  field: RefObject<HTMLInputElement | null>;
  close: () => void;
};

function Body({ confirmVerb, allowNewFolder, field, ...props }: BodyProps) {
  const shell = useShell();
  const list = useRef<HTMLDivElement>(null);
  const browser = useBrowser({
    ...props,
    field,
    refocus: shell === "desktop",
    page: () => Math.floor((list.current?.clientHeight ?? 0) / rowHeights[shell].one),
  } satisfies BrowserProps);
  const listId = useId();
  const verdictId = useId();
  const parts: Parts = {
    browser,
    field,
    list,
    listId,
    verdictId,
    heights: rowHeights[shell],
    confirmVerb,
    allowNewFolder,
    ...props,
  };
  return shell === "phone" ? <PhoneBody {...parts} /> : <DesktopBody {...parts} />;
}
