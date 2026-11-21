import { CommandLineIcon, Mail01Icon, WebhookIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react";

import apprise from "./logos/apprise.svg";
import discord from "./logos/discord.svg";
import gotify from "./logos/gotify.svg";
import ntfy from "./logos/ntfy.svg";
import pushbullet from "./logos/pushbullet.svg";
import pushover from "./logos/pushover.svg";
import slack from "./logos/slack.svg";
import telegram from "./logos/telegram.svg";

type Mark = { logo: string; icon?: never } | { icon: IconSvgElement; logo?: never };

const marks: Readonly<Record<string, Mark>> = {
  discord: { logo: discord },
  telegram: { logo: telegram },
  slack: { logo: slack },
  pushover: { logo: pushover },
  gotify: { logo: gotify },
  ntfy: { logo: ntfy },
  pushbullet: { logo: pushbullet },
  apprise: { logo: apprise },
  email: { icon: Mail01Icon },
  webhook: { icon: WebhookIcon },
  script: { icon: CommandLineIcon },
};

/** A provider's logo, or an icon for one with no logo of its own, 28px square. */
export function ProviderMark({ id }: { id: string }) {
  const mark = marks[id];
  if (mark === undefined) {
    throw new Error(`no mark for the provider ${id}`);
  }
  return mark.logo === undefined ? (
    <HugeiconsIcon icon={mark.icon} size={28} aria-hidden="true" className="shrink-0 text-ink-2" />
  ) : (
    // oxlint-disable-next-line slipstream/no-restricted-syntax -- a bundled provider logo, not artwork
    <img src={mark.logo} alt="" className="size-7 shrink-0" />
  );
}
