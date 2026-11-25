import type {
  MediaserversHowToWatch as HowTo,
  MediaserversMyAccessRow as MyRow,
} from "../../api/gen/model";
import { type GuidedStep, GuidedSteps } from "../../ui/guided-steps";

const downloads: Readonly<Record<string, string>> = {
  Plex: "https://www.plex.tv/apps-devices/",
  Jellyfin: "https://jellyfin.org/downloads/",
  Emby: "https://emby.media/download.html",
};

function getApp(app: string): GuidedStep {
  const href = downloads[app];
  return {
    label: `Get the ${app} app`,
    ...(href === undefined ? {} : { link: { label: `${app}'s download page`, href } }),
  };
}

function stepsOf(row: MyRow, howTo: HowTo): GuidedStep[] {
  if (row.server.access === "plex_tv") {
    return [
      getApp(howTo.app),
      { label: `Sign in as ${howTo.signInAs}` },
      { label: `Pick ${row.server.name}` },
    ];
  }
  return [
    getApp(howTo.app),
    {
      label:
        howTo.address === undefined
          ? `Ask ${row.adminName} for the server address`
          : `Add the server ${howTo.address}`,
    },
    { label: `Choose Manual Login and enter ${howTo.signInAs}` },
  ];
}

export function HowToWatch({ row, plexPassLine }: { row: MyRow; plexPassLine: boolean }) {
  const howTo = row.howToWatch;
  if (howTo === undefined) {
    throw new Error("a row the Member can watch carries how to watch it");
  }
  return (
    <div className="flex flex-col gap-4">
      <GuidedSteps label={`How to watch ${row.server.name}`} steps={stepsOf(row, howTo)} />
      <p>Your SlipStream PIN doesn&apos;t work in {howTo.app}.</p>
      {plexPassLine && row.server.access === "plex_tv" && (
        <p>
          Away from home, you need your own Plex Pass or Remote Watch Pass to play videos in Plex.
        </p>
      )}
    </div>
  );
}
