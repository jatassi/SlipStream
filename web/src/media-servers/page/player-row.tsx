import type { PlaybackPlayer as Player, PlaybackReport as Report } from "../../api/gen/model";
import { Chip } from "../../ui/chip";
import { ListRow } from "../../ui/list";
import { PlainTime } from "../../ui/relative-time";
import { type LeadContext, LeadLine, playerLead } from "./lead-line";
import { PlayerDetails } from "./player-details";

/** One player: how many of its plays the server transcoded, why, and what would have avoided it. */
export function PlayerRow({
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
  return (
    <ListRow
      title={player.name}
      meta={
        <>
          {`${player.product} · ${player.platform} · Last played `}
          <PlainTime at={player.lastPlayedAt} form="age" />
        </>
      }
      trailing={<Chip>{`${player.transcoded} of ${player.plays} transcoded`}</Chip>}
      detail={
        <div className="flex flex-col gap-1 pt-1 pb-1 text-meta text-ink-2">
          <LeadLine ctx={ctx} parts={playerLead(ctx, player)} />
          <PlayerDetails ctx={ctx} serverId={serverId} report={report} player={player} />
        </div>
      }
    />
  );
}
