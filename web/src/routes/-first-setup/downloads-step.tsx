import { useListDownloadClientsSuspense } from "../../api/gen/query/downloadclients/downloadclients";
import { useListVolumesSuspense } from "../../api/gen/query/roots/roots";
import { DownloadsRow } from "./downloads-row";
import { downloadsApart } from "./drives";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

export function DownloadsStep({ nav }: StepBodyProps) {
  const { cards } = useListDownloadClientsSuspense().data;
  const volumes = useListVolumesSuspense().data;
  return (
    <>
      <ul
        aria-label="Downloading Nodes"
        className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {cards.map((card) => (
          <DownloadsRow key={card.nodeId} card={card} apart={downloadsApart(card, volumes)} />
        ))}
      </ul>
      <StepFooter nav={nav} />
    </>
  );
}
