import { ArrowUpRight01Icon, PlayIcon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import type { ReactNode } from "react";

import type {
  FilesItemStatus,
  WorksAvailability,
  WorksContentType,
  WorksFieldSpec,
  WorksItem,
  WorksPerson,
  WorksRating,
  WorksWebLink,
  WorksWorkView,
} from "../../api/gen/model";
import { clock } from "../../format";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { Icon } from "../../ui/icon";
import { Section } from "../../ui/section";
import { type Placed, codeOf, placedOf } from "../fields-by-tier";
// oxlint-disable-next-line import/max-dependencies -- About shows every place a type puts a field, with its art and links
import { aboutText, dayWords, ratingValue } from "./fields-format";
import { OpenIn } from "./open-in";

type Holder = Pick<WorksWorkView, "fields" | "artwork">;

/** The Work's values of the fields its type shows in a place, in declared order. */
export const placed = (type: WorksContentType, work: WorksWorkView, place: Place) =>
  placedOf(type, "work", work.fields, place);

type Place = WorksFieldSpec["in"][number];

const textOf = (holder: Holder, key: string | null): string | undefined => {
  const value = holder.fields.find((field) => field.key === key);
  return value?.kind === "text" ? value.text : undefined;
};

/** "R (United States)" with its `With` field, and the `Art` kind's image beside it. */
function AboutValue({ holder, item: { spec, value } }: { holder: Holder; item: Placed }) {
  const bracket = textOf(holder, spec.with);
  const art = holder.artwork.find(({ kind }) => kind === spec.art);
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      {art !== undefined && (
        <img
          src={`${art.url}?size=small`}
          alt=""
          className="h-5 max-w-24 object-contain"
          loading="lazy"
        />
      )}
      {bracket === undefined ? aboutText(value) : `${aboutText(value)} (${bracket})`}
    </span>
  );
}

/** Fields as labels and values, a Work's or an Item's. */
export function KeyValues({ holder, items }: { holder: Holder; items: readonly Placed[] }) {
  return (
    <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-6 gap-y-2">
      {items.map((item) => (
        <div key={item.spec.key} className="contents">
          <dt className="text-ink-3">{item.spec.label}</dt>
          <dd className="text-ink-1">
            <AboutValue holder={holder} item={item} />
          </dd>
        </div>
      ))}
    </dl>
  );
}

function Person({ person }: { person: WorksPerson }) {
  return (
    <li className="flex w-24 shrink-0 flex-col gap-1.5">
      <span className="block aspect-2/3 w-24 overflow-hidden rounded-item bg-glass-raised ring-1 ring-hairline ring-inset">
        {person.photoUrl !== null && (
          <img
            src={`${person.photoUrl}?size=small`}
            alt=""
            loading="lazy"
            className="size-full object-cover"
          />
        )}
      </span>
      <span className="text-meta font-semibold text-ink-1">{person.name}</span>
      {person.role !== "" && <span className="text-meta text-ink-3">{person.role}</span>}
    </li>
  );
}

// People keep the source's order, grouped by the department each first appears in.
function departments(people: readonly WorksPerson[]): [string, WorksPerson[]][] {
  const groups = new Map<string, WorksPerson[]>();
  for (const person of people) {
    groups.set(person.department, [...(groups.get(person.department) ?? []), person]);
  }
  return [...groups];
}

function Credits({ items }: { items: readonly Placed[] }) {
  return items.flatMap(({ spec, value }) =>
    value.kind === "people"
      ? departments(value.people).map(([department, people]) => (
          <div key={`${spec.key} ${department}`} className="flex flex-col gap-2">
            <h3 className="text-label text-ink-3">{department === "" ? spec.label : department}</h3>
            <ul
              aria-label={department === "" ? spec.label : department}
              className="flex gap-3 overflow-x-auto pb-1"
            >
              {people.map((person) => (
                <Person key={`${person.name} ${person.role}`} person={person} />
              ))}
            </ul>
          </div>
        ))
      : [],
  );
}

const votesOf = (votes: number) => `${votes.toLocaleString()} ${votes === 1 ? "vote" : "votes"}`;

export function Ratings({ ratings }: { ratings: readonly WorksRating[] }) {
  return (
    <ul aria-label="Ratings" className="flex flex-col gap-1.5">
      {ratings.map((rating) => (
        <li key={rating.source} className="flex flex-wrap items-baseline gap-x-2">
          <span className="font-semibold text-ink-1">{rating.label}</span>
          <Data className="text-ink-1">{ratingValue(rating)}</Data>
          <span className="text-meta text-ink-3">
            {rating.votes !== null && `${votesOf(rating.votes)} · `}fetched{" "}
            <Data>{clock(rating.fetchedAt)}</Data>
          </span>
        </li>
      ))}
    </ul>
  );
}

const linkIcons: { readonly [K in WorksWebLink["kind"]]: IconSvgElement } = {
  page: ArrowUpRight01Icon,
  trailer: PlayIcon,
};

function Links({ links }: { links: readonly WorksWebLink[] }) {
  return (
    <ul aria-label="Links" className="flex flex-wrap gap-2">
      {links.map((link) => (
        <li key={link.key}>
          <a
            href={link.url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-(--control-height) items-center gap-2 rounded-capsule bg-fill px-3.5 text-button font-semibold text-ink-1 ring-1 ring-hairline-strong ring-inset hover:bg-fill-hover"
          >
            <Icon icon={linkIcons[link.kind]} className="size-4" />
            {link.label}
          </a>
        </li>
      ))}
    </ul>
  );
}

function Part({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-label text-ink-3">{title}</h3>
      {children}
    </div>
  );
}

/** "Counts as out on April 16, 2021, Digital release in United States." */
function countsAsOut({ date, basis }: WorksAvailability): string {
  return basis === ""
    ? `Counts as out on ${dayWords(date)}.`
    : `Counts as out on ${dayWords(date)}, ${basis}.`;
}

type Dated = { readonly item: WorksItem; readonly date: string };

function nextAndLatest(
  work: WorksWorkView,
  statuses: readonly FilesItemStatus[],
): { next: Dated | undefined; latest: Dated | undefined } {
  const released = new Map(statuses.map((s) => [s.itemId, s.released]));
  const dated = work.items.flatMap((item) =>
    item.withdrawnAt === null && !item.aside && item.availability !== null
      ? [{ item, date: item.availability.date }]
      : [],
  );
  return {
    next: dated
      .filter((at) => released.get(at.item.id) === false)
      .toSorted((a, b) => a.date.localeCompare(b.date))[0],
    latest: dated
      .filter((at) => released.get(at.item.id) === true)
      .toSorted((a, b) => b.date.localeCompare(a.date))[0],
  };
}

/**
 * A one-Item Work's day it counts as out, or for many Items the next
 * Item to come out and the latest one out.
 */
function availabilityOf(
  type: WorksContentType,
  work: WorksWorkView,
  statuses: readonly FilesItemStatus[],
): string[] {
  if (type.items === "one") {
    const availability = work.items[0]?.availability;
    return availability === undefined || availability === null ? [] : [countsAsOut(availability)];
  }
  const { next, latest } = nextAndLatest(work, statuses);
  const noun = type.item.one.toLowerCase();
  return [
    ...(latest === undefined
      ? []
      : [
          `The latest ${noun}, ${codeOf(type, latest.item)}, counted as out on ${dayWords(latest.date)}.`,
        ]),
    ...(next === undefined
      ? []
      : [`The next ${noun}, ${codeOf(type, next.item)}, counts as out on ${dayWords(next.date)}.`]),
  ];
}

/** About the Work: its tagline and what its type places there, its credits, ratings, links and Open in links, then Details. */
export function About({
  type,
  work,
  statuses,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  statuses: readonly FilesItemStatus[];
}) {
  const tagline = placed(type, work, "tagline").at(0);
  const credits = placed(type, work, "credits");
  const details = placed(type, work, "details");
  const availability = availabilityOf(type, work, statuses);
  const record = `${type.source} lists this ${type.work.one.toLowerCase()} as ${work.key}.`;
  return (
    <Section title="About">
      <div className="flex flex-col gap-5">
        {tagline !== undefined && (
          <p className="text-card-title text-ink-1">{aboutText(tagline.value)}</p>
        )}
        <KeyValues holder={work} items={placed(type, work, "about")} />
        {availability.map((line) => (
          <p key={line} className="text-ink-2">
            {line}
          </p>
        ))}
        {credits.length > 0 && <Credits items={credits} />}
        {work.ratings.length > 0 && (
          <Part title="Ratings">
            <Ratings ratings={work.ratings} />
          </Part>
        )}
        {work.links.length > 0 && (
          <Part title="Links">
            <Links links={work.links} />
          </Part>
        )}
        <OpenIn links={work.openIn} />
        {details.length === 0 ? (
          <p className="text-ink-2">{record}</p>
        ) : (
          <Details summary={record}>
            <KeyValues holder={work} items={details} />
          </Details>
        )}
      </div>
    </Section>
  );
}
