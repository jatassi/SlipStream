import { useState } from "react";

import { sentenceOf } from "../../../../api/errors";
import type { NodesOpenCode } from "../../../../api/gen/model";
import { revokeJoinCode } from "../../../../api/gen/query/nodes/nodes";
import { clock } from "../../../../format";
import { AlertLine } from "../../../../ui/alert-line";
import { Button } from "../../../../ui/button";
import { List, ListRow } from "../../../../ui/list";
import { Section } from "../../../../ui/section";
import { useRefreshCodes } from "./codes";

function RevokeButton({ code, label }: { code: NodesOpenCode; label: string }) {
  const refreshCodes = useRefreshCodes();
  const [revoking, setRevoking] = useState(false);
  const [failure, setFailure] = useState<string>();
  return (
    <span className="flex items-center gap-2">
      {failure !== undefined && <AlertLine text={failure} />}
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
  if (codes.length === 0) {
    return null;
  }
  return (
    <Section title="Open Join Codes" count={codes.length}>
      <List label="Open Join Codes">
        {codes.map((code) => {
          const made = `Made ${clock(code.createdAt)}`;
          const self = code.id === current;
          return (
            <ListRow
              key={code.id}
              title={self ? "This code" : made}
              meta={`${self ? `${made} · ` : ""}Expires ${clock(code.expiresAt)}`}
              trailing={
                <RevokeButton
                  code={code}
                  label={
                    self ? "Revoke this code" : `Revoke the code made ${clock(code.createdAt)}`
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
