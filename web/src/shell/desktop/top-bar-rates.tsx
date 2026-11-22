import { Rates, totalOf, useLatestTraffic } from "../../torrents/rates";
import { useMedia, wideDesktopQuery } from "../media";

/** Every Node's download and upload rates, in wide windows once a sample has arrived. */
export function TopBarRates() {
  const traffic = useLatestTraffic();
  const wide = useMedia(wideDesktopQuery);
  if (!wide || traffic === undefined) {
    return null;
  }
  return (
    <span className="text-meta text-ink-2">
      <Rates pace={totalOf(traffic)} />
    </span>
  );
}
