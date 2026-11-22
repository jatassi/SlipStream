import { Rates, moving, totalOf, useLatestTraffic } from "../../torrents/rates";
import { useMedia, wideDesktopQuery } from "../media";

/** Every Node's download and upload rates, in wide windows while bytes move. */
export function TopBarRates() {
  const traffic = useLatestTraffic();
  const wide = useMedia(wideDesktopQuery);
  const pace = traffic === undefined ? undefined : totalOf(traffic);
  if (!wide || pace === undefined || !moving(pace)) {
    return null;
  }
  return (
    <span className="text-meta text-ink-2">
      <Rates pace={pace} />
    </span>
  );
}
