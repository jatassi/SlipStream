import { Link } from "@tanstack/react-router";

import type {
  PlaybackBlocked as Blocked,
  PlaybackGroup as Group,
  PlaybackOutcome as Outcome,
  PlaybackPlayer as Player,
  PlaybackSuggestion as Suggestion,
  PlaybackWorkLine as WorkLine,
  WorksContentType,
} from "../../api/gen/model";
import { type Noun, countWords, nounOf } from "../../format";
import { openManualSearch } from "../../manual/sheet-state";
import { Button } from "../../ui/button";
import { causeWords, dimensionWords, factWords } from "./playback-words";

/** What a lead line reads against: the server and player it names, the report's window and the Content Types. */
export type LeadContext = {
  readonly server: string;
  readonly player: string;
  /** "in the last 30 days", or "since Nov 12" while the history starts later. */
  readonly window: string;
  /** Where the report starts, "since Oct 15". */
  readonly since: string;
  readonly plex: boolean;
  readonly types: readonly WorksContentType[];
};

export type Part =
  | { readonly kind: "text"; readonly text: string }
  | { readonly kind: "work"; readonly workId: string; readonly title: string }
  | { readonly kind: "add"; readonly suggestion: Suggestion }
  | { readonly kind: "search"; readonly workId: string; readonly itemId: string };

const text = (words: string): Part => ({ kind: "text", text: words });

const workPart = (work: WorkLine): Part => ({
  kind: "work",
  workId: work.workId,
  title: work.title,
});

const plays: Noun = { one: "play", many: "plays" };
const mixed: Noun = { one: "Work", many: "Works" };

const were = (count: number) => (count === 1 ? "was" : "were");

export function need<T>(value: T | undefined, what: string): T {
  if (value === undefined) {
    throw new Error(`a Playback report outcome has no ${what}`);
  }
  return value;
}

function worksNoun(works: readonly WorkLine[], types: readonly WorksContentType[]): Noun {
  const ids = new Set(works.map((work) => work.contentType));
  const type = ids.size === 1 ? types.find((each) => ids.has(each.id)) : undefined;
  return type === undefined
    ? mixed
    : { one: type.work.one.toLowerCase(), many: type.work.many.toLowerCase() };
}

/** What one Work lacks for the outcome's Target, and the clause's link to the sentence before it. */
export function workLine(ctx: LeadContext, work: WorkLine): { and: boolean; parts: Part[] } {
  if (work.state === "no_file") {
    return {
      and: true,
      parts: [
        workPart(work),
        text(" has no file for it yet. "),
        { kind: "search", workId: work.workId, itemId: work.itemId },
        text("."),
      ],
    };
  }
  if (work.state === "not_in_profile") {
    return {
      and: false,
      parts: [
        text(`Add it to the ${need(work.profile, "Profile").name} Profile so that `),
        workPart(work),
        text(" gets a file for it."),
      ],
    };
  }
  if (work.state === "not_proven") {
    const { fact, value } = need(work.fact, "fact");
    return {
      and: false,
      parts: [
        workPart(work),
        text(
          ` has a file for it, but ${ctx.player} hasn't direct played its ${factWords[fact](value)} yet.`,
        ),
      ],
    };
  }
  return {
    and: false,
    parts: [workPart(work), text(` has a file for it, but ${ctx.server} doesn't list it yet.`)],
  };
}

function existingParts(ctx: LeadContext, outcome: Outcome): Part[] {
  const avoid = `The ${need(outcome.target, "Target").name} Target would avoid that`;
  const works = need(outcome.works, "Works");
  const [only] = works;
  if (only === undefined || works.length > 1) {
    const noun = worksNoun(works, ctx.types);
    return [
      text(
        `${avoid}, and ${countWords(works.length, noun)} played on ${ctx.player} need a file for it. Details lists each one.`,
      ),
    ];
  }
  const line = workLine(ctx, only);
  return [text(line.and ? `${avoid}, and ` : `${avoid}. `), ...line.parts];
}

function unprovenWords(group: Group, missing: Blocked): string {
  const words = dimensionWords[missing.dimension];
  if (!("unproven" in words)) {
    throw new Error(`no Target can choose ${missing.dimension}, so it can't be unproven`);
  }
  const blocking = group.dimensions.some((each) => each.dimension === missing.dimension);
  return blocking ? words.unproven(missing.value) : `any ${words.side}`;
}

type OutcomeWords = (ctx: LeadContext, group: Group, outcome: Outcome) => Part[];

const outcomeWords: { readonly [K in Outcome["kind"]]: OutcomeWords } = {
  version: (ctx, _group, outcome) => {
    const version = need(outcome.version, "version");
    const played = `The ${version.label} version would have played directly.`;
    return [
      text(
        version.separateItem
          ? `${played} ${ctx.server} lists it as a separate item. Play that item on ${ctx.player} instead.`
          : `${played} Choose it when you play on ${ctx.player}.`,
      ),
    ];
  },
  suggest: (ctx, _group, outcome) => {
    const suggestion = need(outcome.suggestion, "suggestion");
    return [
      text(
        `A ${suggestion.name} Target would avoid that, and Details lists the plays on ${ctx.player} that prove each of its values. `,
      ),
      { kind: "add", suggestion },
    ];
  },
  existing: (ctx, _group, outcome) => existingParts(ctx, outcome),
  unproven: (ctx, group, outcome) => {
    const missing = unprovenWords(group, need(outcome.missing, "missing dimension"));
    return [
      text(
        `SlipStream can't suggest a Target yet, because ${ctx.player} hasn't direct played ${missing} ${ctx.window}. Play ${missing} on ${ctx.player} to get a suggestion.`,
      ),
    ];
  },
  no_target: (ctx, group) => {
    const words = dimensionWords[need(group.cause[0], "cause").dimension];
    return "follow" in words ? [text(words.follow(ctx.server))] : [];
  },
};

function transcodedWords(ctx: LeadContext, count: number, total: number): string {
  return `${count} of ${countWords(total, plays)} on ${ctx.player} ${were(count)} transcoded`;
}

/**
 * One outcome's line: the count of its plays, the cause and what would have
 * avoided the transcode. It never says what the server will pick, because the
 * server picks.
 */
export function leadOf(
  ctx: LeadContext,
  player: Player,
  group: Group,
  outcome: Outcome,
  count: number,
): Part[] {
  const transcoded = transcodedWords(ctx, count, player.plays);
  const unknown = group.cause.some((each) => each.dimension === "unknown");
  const noReason = ctx.plex
    ? `${transcoded}, and what the player received doesn't show why. `
    : `${transcoded}. ${ctx.server} gave no reason, and what the player received doesn't show why. `;
  const head = unknown ? noReason : `${transcoded} because ${causeWords(group.cause)}. `;
  return [text(head), ...outcomeWords[outcome.kind](ctx, group, outcome)];
}

/** A player's lead: its largest group with that group's largest outcome, and how many plays the rest hold. */
export function playerLead(ctx: LeadContext, player: Player): Part[] {
  const [largest, ...others] = player.groups;
  if (player.transcoded === 0) {
    return [
      text(
        player.plays === 1
          ? `${ctx.player}'s 1 play wasn't transcoded.`
          : `None of ${ctx.player}'s ${player.plays} plays were transcoded.`,
      ),
    ];
  }
  if (largest === undefined) {
    return [
      text(
        `${countWords(player.transcoded, plays)} on ${ctx.player} ${were(player.transcoded)} transcoded. They were of files outside SlipStream's library or at paths no Path mapping covers, so the report can't explain them. Check Path mappings.`,
      ),
    ];
  }
  const lead = leadOf(ctx, player, largest, need(largest.outcomes[0], "outcome"), largest.plays);
  const more = others.reduce((sum, group) => sum + group.plays, 0);
  const { unmatched } = player;
  const rest = [
    more === 0
      ? ""
      : ` ${more} more ${nounOf(more, plays)} on ${ctx.player} ${were(more)} transcoded for other reasons.`,
    unmatched === 0
      ? ""
      : ` ${unmatched} more ${nounOf(unmatched, plays)} on ${ctx.player} ${were(unmatched)} transcoded, and the report can't explain ${unmatched === 1 ? "it" : "them"}. Details lists the causes to check.`,
  ];
  return [...lead, ...rest.filter(Boolean).map((words) => text(words))];
}

const linkLook =
  "underline decoration-ink-3 underline-offset-4 hover:decoration-ink-1 focus-visible:decoration-ink-1";

function keyOf(part: Part): string {
  if (part.kind === "text") {
    return part.text;
  }
  return part.kind === "add" ? "add" : `${part.kind}-${part.workId}`;
}

function PartView({ ctx, part }: { ctx: LeadContext; part: Part }) {
  if (part.kind === "text") {
    return part.text;
  }
  if (part.kind === "work") {
    return (
      <Link to="/works/$workId" params={{ workId: part.workId }} className={linkLook}>
        {part.title}
      </Link>
    );
  }
  if (part.kind === "search") {
    return (
      <button
        type="button"
        className={linkLook}
        onClick={() => {
          openManualSearch({ workId: part.workId, scope: { itemIds: [part.itemId] } });
        }}
      >
        Search for one
      </button>
    );
  }
  return (
    <Button
      size="small"
      render={
        <Link
          to="/settings/targets/new"
          search={{ from: { server: ctx.server, player: ctx.player, suggestion: part.suggestion } }}
        />
      }
    >
      Add this Target
    </Button>
  );
}

/** A line of parts as text, with its Work links, Search for one and Add this Target. */
export function LeadLine({ ctx, parts }: { ctx: LeadContext; parts: readonly Part[] }) {
  return (
    <p className="text-pretty text-ink-1">
      {parts.map((part) => (
        <PartView key={keyOf(part)} ctx={ctx} part={part} />
      ))}
    </p>
  );
}
