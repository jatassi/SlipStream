import { Search01Icon } from "@hugeicons/core-free-icons";

import { Data, dataFont } from "../ui/data";
import { Icon } from "../ui/icon";
import { optionId } from "../ui/virtual-listbox";
import { isPath } from "./browser-state";
import type { Parts } from "./parts";
import type { Browser } from "./use-browser";

export function GoField({ browser, field, listId }: Parts) {
  const { state, rows } = browser;
  const listed = rowsShown(browser);
  const activeShown = listed && state.active >= 0 && state.active < rows.length;
  return (
    <div className="relative flex items-center">
      <Icon icon={Search01Icon} className="pointer-events-none absolute left-3 size-4 text-ink-3" />
      <input
        ref={field}
        role="combobox"
        aria-label="Filter, or go to a path"
        aria-autocomplete="list"
        aria-expanded={listed}
        aria-controls={listed ? listId : undefined}
        aria-activedescendant={activeShown ? optionId(listId, state.active) : undefined}
        placeholder="Filter, or type a path"
        autoComplete="off"
        autoCapitalize="off"
        spellCheck={false}
        value={state.field}
        onChange={(event) => {
          browser.type(event.target.value);
        }}
        onKeyDown={(event) => {
          browser.onKeyDown(event, "field");
        }}
        className={`h-(--input-height) w-full min-w-0 rounded-item bg-well pr-3 pl-9 text-body text-ink-1 ring-1 ring-hairline ring-inset placeholder:text-ink-3 in-data-[density=touch]:text-input-touch ${isPath(state.field) ? dataFont : ""}`}
      />
    </div>
  );
}

const rowsShown = ({ listing, rows }: Browser) =>
  listing.problem === null && !listing.skeleton && rows.length > 0;

const shareOnly = /^\\\\[^\\/]+[\\/]?$/u;

export function Footnotes({ browser }: { browser: Browser }) {
  const { state, listing } = browser;
  return (
    <>
      {shareOnly.test(state.field) && (
        <p className="text-meta text-ink-3">
          Type a share name after the server, such as <Data>\\nas\media</Data>.
        </p>
      )}
      {listing.current?.path === "" && listing.current.service && (
        <p className="text-meta text-ink-3">
          Mapped drive letters aren&apos;t visible to SlipStream when it runs as a service. Type the
          share instead, such as <Data>\\server\share</Data>.
        </p>
      )}
    </>
  );
}
