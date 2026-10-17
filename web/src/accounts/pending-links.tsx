import {
  Copy01Icon,
  Delete02Icon,
  MailSend01Icon,
  MoreHorizontalIcon,
} from "@hugeicons/core-free-icons";
import { useState } from "react";

import type { IdentityPendingLink } from "../api/gen/model";
import { clock } from "../format";
import { Empty } from "../shell/empty";
import { IconButton } from "../ui/icon-button";
import { List, ListRow } from "../ui/list";
import { Menu } from "../ui/menu";
import { Section } from "../ui/section";
import { type Acting, PendingLinkDialogs } from "./pending-link-dialogs";

const metaOf = (link: IdentityPendingLink) =>
  `Sent ${clock(link.issuedAt)}${link.issuedBy === undefined ? "" : ` by ${link.issuedBy}`} · Expires ${clock(link.expiresAt)}`;

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
        <Empty>No Signup links are waiting. A link stays here until someone uses it.</Empty>
      ) : (
        <List label="Pending Signup links">
          {links.map((link) => (
            <ListRow
              key={link.id}
              title={link.email}
              meta={metaOf(link)}
              trailing={<LinkMenu link={link} onAct={setActing} />}
            />
          ))}
        </List>
      )}
      <PendingLinkDialogs acting={acting} setActing={setActing} />
    </Section>
  );
}
