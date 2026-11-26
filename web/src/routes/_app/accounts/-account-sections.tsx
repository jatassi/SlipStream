import { pinSentence } from "../../../accounts/pin-state";
import type { IdentityAccountDetail } from "../../../api/gen/model";
import { Button } from "../../../ui/button";
import { useClock } from "../../../ui/clock";
import { Empty } from "../../../ui/empty";
import { Fact, Facts } from "../../../ui/facts";
import { List, ListRow } from "../../../ui/list";
import { RelativeTime } from "../../../ui/relative-time";
import { Section } from "../../../ui/section";
import type { Asking } from "./-confirmations";
// oxlint-disable-next-line import/max-dependencies -- the Account page lays out its Profile, PIN, Passkeys, Sessions, Media Servers and Delete sections
import { MediaServers } from "./-media-servers";

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

function Profile({ detail: { account }, onAsk }: Props) {
  const ask = (asking: Asking, label: string) => ({
    label,
    onClick: () => {
      onAsk(asking);
    },
  });
  return (
    <Section title="Profile">
      <Facts look="sheet">
        <Fact label="Name">{account.displayName}</Fact>
        <Fact label="Email" action={ask("email", "Change email")}>
          {account.email}
        </Fact>
        <Fact label="Role" action={ask("role", account.admin ? "Make Member" : "Make Admin")}>
          {account.admin ? "Admin" : "Member"}
        </Fact>
        <Fact label="Sign-in" action={ask("access", account.disabled ? "Turn on" : "Turn off")}>
          {account.disabled ? "Turned off" : "On"}
        </Fact>
        <Fact label="Last sign-in">
          {account.lastSignInAt === undefined ? (
            "Never"
          ) : (
            <RelativeTime at={account.lastSignInAt} form="age" />
          )}
        </Fact>
      </Facts>
    </Section>
  );
}

function Pin({ detail: { account }, onAsk }: Props) {
  const clock = useClock();
  return (
    <Section title="PIN">
      <List label="PIN">
        <ListRow
          title={pinSentence(account.pin, clock)}
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
        <Empty look="sheet">No passkeys. This Account signs in with its PIN.</Empty>
      ) : (
        <List label="Passkeys">
          {passkeys.map((passkey) => (
            <ListRow
              key={passkey.id}
              title={passkey.name}
              meta={
                <>
                  {passkey.host} · Added <RelativeTime at={passkey.createdAt} form="clock" />
                  {passkey.lastUsedAt !== undefined && (
                    <>
                      {" · Last used "}
                      <RelativeTime at={passkey.lastUsedAt} form="age" />
                    </>
                  )}
                </>
              }
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
        <Empty look="sheet">Not signed in anywhere.</Empty>
      ) : (
        <div className="flex flex-col items-start gap-3">
          <div className="w-full">
            <List label="Sessions">
              {sessions.map((session) => (
                <ListRow
                  key={session.id}
                  title={session.device}
                  meta={
                    <>
                      {session.address} · Last seen{" "}
                      <RelativeTime at={session.lastSeenAt} form="age" />
                    </>
                  }
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
      <MediaServers account={props.detail.account} />
      <Section title="Delete">
        <AskButton asking="delete" label="Delete Account" look="fail" onAsk={props.onAsk} />
      </Section>
    </div>
  );
}
