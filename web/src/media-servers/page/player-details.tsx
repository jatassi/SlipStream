import type {
  PlaybackBlocked as Blocked,
  PlaybackEvidence as Evidence,
  PlaybackGroup as Group,
  PlaybackOutcome as Outcome,
  PlaybackPlayer as Player,
  PlaybackReport as Report,
} from "../../api/gen/model";
import { countWords, durationWords } from "../../format";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { Fact, Facts } from "../../ui/facts";
import { RelativeTime } from "../../ui/relative-time";
import { type LeadContext, LeadLine, leadOf, workLine } from "./lead-line";
import { causeWords, factWords, hdrWord } from "./playback-words";
import { PlaysDialog } from "./plays-dialog";

const plays = { one: "play", many: "plays" };

const list = new Intl.ListFormat("en", { type: "conjunction" });

const valueWords = (blocked: Blocked) =>
  blocked.dimension === "hdr" ? hdrWord(blocked.value) : blocked.value;

function EvidenceList({ ctx, evidence }: { ctx: LeadContext; evidence: readonly Evidence[] }) {
  return (
    <div className="flex flex-col gap-1">
      <p>{`Played directly on ${ctx.player}`}</p>
      <Facts look="well">
        {evidence.map((each) => (
          <Fact key={`${each.fact}-${each.value}`} label={factWords[each.fact](each.value)}>
            {`${each.title} · `}
            <RelativeTime at={each.startedAt} form="clock" />
          </Fact>
        ))}
      </Facts>
    </div>
  );
}

function OutcomeDetail({
  ctx,
  player,
  group,
  outcome,
}: {
  ctx: LeadContext;
  player: Player;
  group: Group;
  outcome: Outcome;
}) {
  const { suggestion, works = [], evidence = [] } = outcome;
  return (
    <div className="flex flex-col gap-2">
      <LeadLine ctx={ctx} parts={leadOf(ctx, player, group, outcome, outcome.plays)} />
      {works.length > 1 && (
        <ul className="flex list-disc flex-col gap-1 pl-5">
          {works.map((work) => (
            <li key={work.workId}>
              <LeadLine ctx={ctx} parts={workLine(ctx, work).parts} />
            </li>
          ))}
        </ul>
      )}
      {suggestion !== undefined && suggestion.blocked.length > 0 && (
        <p>{`Left out because they blocked direct play: ${list.format(suggestion.blocked.map(valueWords))}.`}</p>
      )}
      {suggestion !== undefined && suggestion.unproven.length > 0 && (
        <p>{`Left out because ${ctx.player} hasn't direct played them ${ctx.window}: ${list.format(suggestion.unproven.map(valueWords))}.`}</p>
      )}
      {evidence.length > 0 && <EvidenceList ctx={ctx} evidence={evidence} />}
    </div>
  );
}

function GroupDetail({ ctx, player, group }: { ctx: LeadContext; player: Player; group: Group }) {
  const named = new Set(group.cause.map((each) => `${each.dimension}:${each.value}`));
  const rest = group.dimensions.filter((each) => !named.has(`${each.dimension}:${each.value}`));
  return (
    <section className="flex flex-col gap-2">
      {group.outcomes.map((outcome) => (
        <OutcomeDetail
          key={`${outcome.kind}-${outcome.version?.fileId ?? outcome.suggestion?.name ?? outcome.target?.id ?? ""}`}
          ctx={ctx}
          player={player}
          group={group}
          outcome={outcome}
        />
      ))}
      {rest.length > 0 && <p>{`Direct play was also blocked because ${causeWords(rest)}.`}</p>}
      {group.reasons.length > 0 && (
        <p>
          {`Reasons from ${ctx.server}: `}
          <Data className="[overflow-wrap:anywhere]">{group.reasons.join(", ")}</Data>
        </p>
      )}
      {group.targets.length > 0 && (
        <p>{`The transcoded files met these Targets when they played: ${list.format(group.targets.map(({ name }) => name))}.`}</p>
      )}
    </section>
  );
}

const choosing = new Set(["suggest", "existing"]);

/** The machinery behind a player's lead: counts, every group and outcome, evidence, the window and the player's plays. */
export function PlayerDetails({
  ctx,
  serverId,
  report,
  player,
}: {
  ctx: LeadContext;
  serverId: string;
  report: Report;
  player: Player;
}) {
  const outcomes = player.groups.flatMap((group) => group.outcomes);
  return (
    <Details>
      <div className="flex flex-col gap-3 pt-1 text-meta">
        <p>{`Direct play ${player.directPlays} · Direct stream ${player.directStreams} · Transcode ${player.transcoded}`}</p>
        {player.groups.map((group) => (
          <GroupDetail
            key={group.dimensions.map((each) => `${each.dimension}:${each.value}`).join(",")}
            ctx={ctx}
            player={player}
            group={group}
          />
        ))}
        {outcomes.some((outcome) => choosing.has(outcome.kind)) && (
          <p>
            {
              "A Target can't choose a file's container, video profile, bit depth or bitrate. A file it accepts can still be transcoded for one of those."
            }
          </p>
        )}
        {player.unmatched > 0 && (
          <p>{`${countWords(player.unmatched, plays)} ${player.unmatched === 1 ? "was" : "were"} of files outside SlipStream's library, at paths no Path mapping covers, or with media info SlipStream couldn't read. Check Path mappings.`}</p>
        )}
        {report.plex && (
          <p>Plex gives no reasons. SlipStream compares each file with what the player received.</p>
        )}
        <p>{`The report covers plays ${ctx.since}. SlipStream keeps play history for ${durationWords(report.window)}. Change it in Settings > System > Media Servers.`}</p>
        <div>
          <PlaysDialog serverId={serverId} player={player.key} name={player.name} />
        </div>
      </div>
    </Details>
  );
}
