import { Link } from "@tanstack/react-router";

import type {
  SubtitlesProfileLanguages,
  SubtitlesProvider,
  SubtitlesSubtitleSettings,
  WorksContentType,
} from "../api/gen/model";
import { scansLeft } from "../subtitles/scan-words";
import { leadOf, wantPhrase } from "../subtitles/settings-words";
import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";

const primaryLink =
  "inline-flex h-(--control-height) items-center rounded-capsule bg-solid px-3.5 text-button font-semibold text-on-solid in-data-[density=touch]:text-body-touch";

/** What holds first, each Storage Node's scans left once ready, then the Admin's next step. */
export function Lead({
  settings: { lead, profiles },
  providers,
  types,
  openKey,
}: {
  settings: SubtitlesSubtitleSettings;
  providers: readonly SubtitlesProvider[];
  types: readonly WorksContentType[];
  openKey: (provider: SubtitlesProvider) => void;
}) {
  const { sentences, next } = leadOf(lead, profiles.length, providers, types);
  const scans = lead.kind === "ready" ? lead.scans : [];
  return (
    <div className="flex flex-col items-start gap-3">
      <p className="text-ink-1">{sentences.join(" ")}</p>
      {scans.length > 0 && (
        <ul aria-label="Audio scans left" className="flex flex-col gap-1 text-ink-2">
          {scans.map((scan) => (
            <li key={scan.nodeId}>{scansLeft(scan.name, scan.left)}</li>
          ))}
        </ul>
      )}
      {next === "profiles" && (
        <Link to="/settings/$section" params={{ section: "profiles" }} className={primaryLink}>
          Open Profiles
        </Link>
      )}
      {typeof next === "object" && (
        <Button
          look="primary"
          onClick={() => {
            openKey(next);
          }}
        >
          {`Add ${next.name} key`}
        </Button>
      )}
    </div>
  );
}

/** Each Profile with a subtitle language, its languages read-only, linking to its editor. */
export function ProfileRows({ profiles }: { profiles: readonly SubtitlesProfileLanguages[] }) {
  return (
    <List label="Profiles with subtitle languages">
      {profiles.map((profile) => (
        <ListRow
          key={profile.id}
          title={profile.name}
          detail={
            <span className="flex flex-col text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
              {profile.languages.map((want) => (
                <span key={`${want.language}-${String(want.forced)}`}>{wantPhrase(want)}</span>
              ))}
            </span>
          }
          render={<Link to="/settings/profiles/$profileId" params={{ profileId: profile.id }} />}
        />
      ))}
    </List>
  );
}
