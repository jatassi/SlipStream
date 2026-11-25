import {
  Copy01Icon,
  Delete02Icon,
  MailSend01Icon,
  MoreHorizontalIcon,
} from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { IdentityPendingLink } from "../api/gen/model";
import { Empty } from "../ui/empty";
import { IconButton } from "../ui/icon-button";
import { List, ListRow } from "../ui/list";
import { Menu } from "../ui/menu";
import { RelativeTime } from "../ui/relative-time";
import { Section } from "../ui/section";
import { type Acting, PendingLinkDialogs } from "./pending-link-dialogs";

const list = new Intl.ListFormat("en", { type: "conjunction" });

function Meta({ link }: { link: IdentityPendingLink }) {
  const offered = link.mediaServers
    .filter((server) => server.present && server.shown)
    .map((server) => server.name);
  return (
    <>
      Sent <RelativeTime at={link.issuedAt} form="clock" />
      {link.issuedBy !== undefined && ` by ${link.issuedBy}`} · Expires{" "}
      <RelativeTime at={link.expiresAt} form="clock" />
      {offered.length > 0 && ` · Offers ${list.format(offered)}`}
    </>
  );
}

function LinkMenu({ link, onAct }: { link: IdentityPendingLink; onAct: (acting: Acting) => void }) {
  return (
    <Menu
      label={`Signup link for ${link.email}`}
      trigger={<IconButton icon={MoreHorizontalIcon} label={`Actions for ${link.email}`} />}
      items={[
        {
          label: "Copy link",
          icon: Copy01Icon,
          onSelect: () => {
            onAct({ kind: "copy", link });
          },
        },
        {
          label: "Send again",
          icon: MailSend01Icon,
          onSelect: () => {
            onAct({ kind: "resend", link });
          },
        },
        "separator",
        {
          label: "Withdraw",
          icon: Delete02Icon,
          tone: "danger",
          onSelect: () => {
            onAct({ kind: "withdraw", link });
          },
        },
      ]}
    />
  );
}

export function PendingLinks({ links }: { links: readonly IdentityPendingLink[] }) {
  const [acting, setActing] = useState<Acting>();
  return (
    <Section title="Pending Signup links" count={links.length}>
      {links.length === 0 ? (
        <Empty look="sheet">
          No Signup links are waiting. A link stays here until someone uses it.
        </Empty>
      ) : (
        <List label="Pending Signup links">
          {links.map((link) => (
            <ListRow
              key={link.id}
              title={link.email}
              meta={<Meta link={link} />}
              trailing={<LinkMenu link={link} onAct={setActing} />}
            />
          ))}
        </List>
      )}
      <PendingLinkDialogs acting={acting} setActing={setActing} />
    </Section>
  );
}
