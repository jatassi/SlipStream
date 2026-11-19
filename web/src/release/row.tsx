import { Fragment, type ReactNode, useId } from "react";

import type { GrabsReleaseFacts } from "../api/gen/model";
import { bytes, countWords } from "../format";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Defined, type Term } from "../ui/defined";
import { Joined } from "../ui/joined";
import { Quality } from "./quality";

/** A Release as every screen that lists one draws it. Each listing's own wire object fits it. */
export type Release = {
  readonly title: string;
  readonly facts: GrabsReleaseFacts;
  /** Bytes. */
  readonly size?: number | undefined;
  readonly freeleech?: boolean | undefined;
};

/** The Indexers that listed a Release and its seeders, as a search run's list reads them. */
export type Heard = { readonly indexers: readonly string[]; readonly seeders: number };

const freeleech: Term = {
  word: "Freeleech",
  definition: "Downloading it doesn't count against your ratio on its Tracker.",
};

const revisionTerm = (word: string): Term => ({
  word,
  definition: "It's a later release of the same video that fixes a problem in an earlier one.",
});

const list = new Intl.ListFormat("en", { type: "conjunction" });

const seeders = { one: "seeder", many: "seeders" };

/**
 * The Quality chip, then the revision, HDR format, audio, Release group and
 * size joined in a line, then a Freeleech chip, each left out when unknown.
 * Hidden dots join the chips to the line, so its text reads as the sentence
 * Go's Summary.Words writes. Undefined when the Release shows no fact.
 */
function factsLine(release: Release, id?: string): ReactNode {
  const { quality, revision, hdr, audio, group } = release.facts;
  const words = [
    revision !== undefined && (
      <Defined key="revision" term={revisionTerm(revision)}>
        {(word) => word}
      </Defined>
    ),
    hdr,
    audio,
    group,
    release.size !== undefined && <Data key="size">{bytes(release.size)}</Data>,
  ].filter((part) => part !== undefined && part !== false);
  const parts = [
    quality !== undefined && <Quality key="quality" quality={quality} />,
    words.length > 0 && (
      <span key="words">
        <Joined parts={words} />
      </span>
    ),
    release.freeleech === true && (
      <Defined key="freeleech" term={freeleech}>
        {(word) => <Chip>{word}</Chip>}
      </Defined>
    ),
  ].filter((part) => part !== false);
  if (parts.length === 0) {
    return undefined;
  }
  return (
    <span id={id} className="flex flex-wrap items-center gap-x-2 gap-y-1 text-ink-1">
      {parts.map((part, at) => (
        // oxlint-disable-next-line react/no-array-index-key -- the parts are fixed in place for each line
        <Fragment key={at}>
          {at > 0 && <span className="sr-only"> · </span>}
          {part}
        </Fragment>
      ))}
    </span>
  );
}

/** The title alone, as a Release that shows no fact reads. */
const bareTitle = (release: Release, id?: string) => (
  <span id={id} className="wrap-anywhere text-ink-1">
    <Data>{release.title}</Data>
  </span>
);

function named(release: Release, factsId?: string): ReactNode {
  const facts = factsLine(release, factsId);
  return facts === undefined ? (
    bareTitle(release, factsId)
  ) : (
    <span className="flex min-w-0 flex-col gap-1">
      {facts}
      <span className="text-meta wrap-anywhere text-ink-3">
        <Data>{release.title}</Data>
      </span>
    </span>
  );
}

/** The Release's facts on one line and its title in Data under them, for a table cell or a fact's value. */
export function ReleaseName({ release }: { release: Release }) {
  return named(release);
}

/**
 * One Release in a list: its facts, then what the screen says about it.
 * renderLead draws a control before it, such as a plan's tick, given the id
 * of the facts that name it. action is the row's one button, which sits
 * beside the facts and drops under them when the row is too narrow. A row
 * given heard shows the title and where it was heard under the facts.
 * Manual search keeps those under Details.
 */
export function ReleaseRow({
  release,
  heard,
  renderLead,
  action,
  children,
}: {
  release: Release;
  heard?: Heard | undefined;
  renderLead?: (factsId: string) => ReactNode;
  action?: ReactNode;
  children?: ReactNode;
}) {
  const factsId = useId();
  return (
    <li className="flex items-start gap-3 py-3">
      {renderLead?.(factsId)}
      <div className="flex min-w-0 flex-1 flex-col items-start gap-1.5">
        <div className="flex w-full flex-wrap items-start gap-x-3 gap-y-1.5">
          <span className="min-w-[min(100%,16rem)] flex-1">
            {heard === undefined
              ? (factsLine(release, factsId) ?? bareTitle(release, factsId))
              : named(release, factsId)}
          </span>
          {action}
        </div>
        {heard !== undefined && (
          <span className="text-meta text-ink-3">
            {list.format(heard.indexers)} · {countWords(heard.seeders, seeders)}
          </span>
        )}
        {children}
      </div>
    </li>
  );
}
