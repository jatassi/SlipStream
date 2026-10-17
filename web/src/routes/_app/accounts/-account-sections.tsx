import type { ReactNode } from "react";

import { pinSentence } from "../../../accounts/pin-state";
import type { IdentityAccountDetail } from "../../../api/gen/model";
import { clock } from "../../../format";
import { Empty } from "../../../shell/empty";
import { Button } from "../../../ui/button";
import { List, ListRow } from "../../../ui/list";
import { Section } from "../../../ui/section";
import type { Asking } from "./-confirmations";

type Props = { detail: IdentityAccountDetail; onAsk: (asking: Asking) => void };

function AskButton({
  asking,
  label,
  look = "default",
  onAsk,
}: {
  asking: Asking;
  label: string;
  look?: "default" | "fail";
  onAsk: Props["onAsk"];
}) {
  return (
    <Button
      size="small"
      look={look}
      onClick={() => {
        onAsk(asking);
      }}
    >
      {label}
    </Button>
  );
}

const Fact = ({ term, value, action }: { term: string; value: string; action?: ReactNode }) => (
  <ListRow title={value} meta={term} trailing={action} />
);

function Profile({ detail: { account }, onAsk }: Props) {
  return (
    <Section title="Profile">
      <List label="Profile">
        <Fact term="Name" value={account.displayName} />
        <Fact
          term="Email"
          value={account.email}
          action={<AskButton asking="email" label="Change email" onAsk={onAsk} />}
        />
        <Fact
          term="Role"
          value={account.admin ? "Admin" : "Member"}
          action={
            <AskButton
              asking="role"
              label={account.admin ? "Make Member" : "Make Admin"}
              onAsk={onAsk}
            />
          }
        />
        <Fact
          term="Sign-in"
          value={account.disabled ? "Turned off" : "On"}
          action={
            <AskButton
              asking="access"
              label={account.disabled ? "Turn on" : "Turn off"}
              onAsk={onAsk}
            />
          }
        />
        <Fact
          term="Last sign-in"
          value={account.lastSignInAt === undefined ? "Never" : clock(account.lastSignInAt)}
        />
      </List>
    </Section>
  );
}

function Pin({ detail: { account }, onAsk }: Props) {
  return (
    <Section title="PIN">
      <List label="PIN">
        <ListRow
          title={pinSentence(account.pin)}
          trailing={<AskButton asking="reset" label="Send a Reset link" onAsk={onAsk} />}
        />
      </List>
    </Section>
  );
}

function Passkeys({ detail: { passkeys } }: Props) {
  return (
    <Section title="Passkeys" count={passkeys.length}>
      {passkeys.length === 0 ? (
        <Empty>No passkeys. This Account signs in with its PIN.</Empty>
      ) : (
        <List label="Passkeys">
          {passkeys.map((passkey) => (
            <ListRow
              key={passkey.id}
              title={passkey.name}
              meta={`${passkey.host} · Added ${clock(passkey.createdAt)}${passkey.lastUsedAt === undefined ? "" : ` · Last used ${clock(passkey.lastUsedAt)}`}`}
            />
          ))}
        </List>
      )}
    </Section>
  );
}

function Sessions({ detail: { sessions }, onAsk }: Props) {
  return (
    <Section title="Sessions" count={sessions.length}>
      {sessions.length === 0 ? (
        <Empty>Not signed in anywhere.</Empty>
      ) : (
        <div className="flex flex-col items-start gap-3">
          <div className="w-full">
            <List label="Sessions">
              {sessions.map((session) => (
                <ListRow
                  key={session.id}
                  title={session.device}
                  meta={`${session.address} · Last seen ${clock(session.lastSeenAt)}`}
                  trailing={session.current ? "This device" : undefined}
                />
              ))}
            </List>
          </div>
          <AskButton asking="sign-out-everywhere" label="Sign out everywhere" onAsk={onAsk} />
        </div>
      )}
    </Section>
  );
}

export function AccountSections(props: Props) {
  return (
    <div className="max-w-[760px]">
      <Profile {...props} />
      <Pin {...props} />
      <Passkeys {...props} />
      <Sessions {...props} />
      <Section title="Delete">
        <AskButton asking="delete" label="Delete Account" look="fail" onAsk={props.onAsk} />
      </Section>
    </div>
  );
}
