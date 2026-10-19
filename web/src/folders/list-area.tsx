import { countWords } from "../format";
import { isPath } from "./browser-state";
import { FolderRows } from "./folder-rows";
import { NewFolderRow, Quiet, SkeletonRows } from "./list-parts";
import type { Parts } from "./parts";
import { ProblemCard } from "./problem-card";
import type { Browser } from "./use-browser";

const folders = { one: "folder", many: "folders" };
const places = { one: "place", many: "places" };

function emptyWords(browser: Browser): string | null {
  const { state, listing } = browser;
  if (isPath(state.field)) {
    return "Press Enter to go to this path.";
  }
  if (listing.shown === undefined) {
    return null;
  }
  if (state.field !== "") {
    return `Nothing here matches ${state.field}.`;
  }
  return listing.shown.path === ""
    ? "This Node shows no drives."
    : "This folder has no folders inside.";
}

export function ListArea(parts: Parts & { className: string }) {
  const { browser, list, listId, heights, nodeName, className } = parts;
  const { state, listing, rows } = browser;
  const announced = announcement(browser, nodeName);
  return (
    <div
      ref={list}
      aria-busy={listing.loading}
      className={`flex flex-col overflow-hidden rounded-card bg-glass ring-1 ring-hairline ring-inset ${className}`}
    >
      {state.creating !== null && (
        <NewFolderRow
          folder={state.creating}
          busy={browser.making}
          onName={browser.nameFolder}
          onKeyDown={(event) => {
            browser.onKeyDown(event, "folderName");
          }}
          onCreate={browser.create}
          onCancel={browser.cancelNewFolder}
        />
      )}
      {listing.problem === null ? (
        listing.skeleton ? (
          <SkeletonRows />
        ) : rows.length === 0 ? (
          <Quiet>{emptyWords(browser)}</Quiet>
        ) : (
          <FolderRows
            id={listId}
            rows={rows}
            active={state.active}
            estimate={rows[0]?.kind === "place" ? heights.two : heights.one}
            onChoose={browser.choose}
          />
        )
      ) : (
        <ProblemCard
          problem={listing.problem}
          nodeName={nodeName}
          onUp={state.path === "" ? undefined : browser.up}
          onRetry={browser.retry}
        />
      )}
      <p aria-live="polite" className="sr-only">
        {announced}
      </p>
    </div>
  );
}

function announcement({ listing, name }: Browser, nodeName: string): string {
  const { current } = listing;
  if (current === undefined) {
    return "";
  }
  return current.path === ""
    ? `${nodeName}, ${countWords(current.places.length, places)}`
    : `${name}, ${countWords(current.folders.length, folders)}`;
}
