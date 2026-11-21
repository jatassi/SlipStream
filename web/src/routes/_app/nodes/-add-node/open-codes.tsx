import { useState } from "react";

import { sentenceOf } from "../../../../api/errors";
import type { NodesOpenCode } from "../../../../api/gen/model";
import { revokeJoinCode } from "../../../../api/gen/query/nodes/nodes";
import { Button } from "../../../../ui/button";
import { useClock } from "../../../../ui/clock";
import { List, ListRow } from "../../../../ui/list";
import { RelativeTime } from "../../../../ui/relative-time";
import { Section } from "../../../../ui/section";
import { SignalLine } from "../../../../ui/signal-line";
// oxlint-disable-next-line import/max-dependencies -- each code row dates the code and revokes it
import { whenWords } from "../../../../ui/time-words";
import { useRefreshCodes } from "./codes";

function RevokeButton({ code, label }: { code: NodesOpenCode; label: string }) {
  const refreshCodes = useRefreshCodes();
  const [revoking, setRevoking] = useState(false);
  const [failure, setFailure] = useState<string>();
  return (
    <span className="flex items-center gap-2">
      {failure !== undefined && (
        <SignalLine tone="fail" announce>
          {failure}
        </SignalLine>
      )}
      <Button
        size="small"
        busy={revoking}
        aria-label={label}
        onClick={() => {
          setRevoking(true);
          revokeJoinCode(code.id)
            .then(refreshCodes)
            .catch((error: unknown) => {
              setFailure(sentenceOf(error));
            })
            .finally(() => {
              setRevoking(false);
            });
        }}
      >
        Revoke
      </Button>
    </span>
  );
}

/** Every Join Code that may still pair a Node, so one handed out by mistake can be revoked. */
export function OpenCodes({
  codes,
  current,
}: {
  codes: readonly NodesOpenCode[];
  current: string;
}) {
  const clock = useClock();
  if (codes.length === 0) {
    return null;
  }
  return (
    <Section title="Open Join Codes" count={codes.length}>
      <List label="Open Join Codes">
        {codes.map((code) => {
          const made = (
            <>
              Made <RelativeTime at={code.createdAt} form="clock" />
            </>
          );
          const self = code.id === current;
          return (
            <ListRow
              key={code.id}
              title={self ? "This code" : made}
              meta={
                <>
                  {self && <>{made} · </>}
                  Expires <RelativeTime at={code.expiresAt} form="clock" />
                </>
              }
              trailing={
                <RevokeButton
                  code={code}
                  label={
                    self
                      ? "Revoke this code"
                      : `Revoke the code made ${whenWords(code.createdAt, "at", clock)}`
                  }
                />
              }
            />
          );
        })}
      </List>
    </Section>
  );
}
