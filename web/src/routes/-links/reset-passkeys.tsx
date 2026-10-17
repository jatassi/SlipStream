import { useState } from "react";

import type { IdentityPasskey } from "../../api/gen/model";
import { useDeletePasskey } from "../../api/gen/query/identity/identity";
import { clock } from "../../format";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { List, ListRow } from "../../ui/list";
import type { Flow } from "./reset-pin";
import type { StepProps } from "./steps";

function PasskeyRow({
  passkey,
  onDeleted,
  onFailed,
}: {
  passkey: IdentityPasskey;
  onDeleted: () => void;
  onFailed: (failure: string) => void;
}) {
  const remove = useDeletePasskey();
  return (
    <ListRow
      title={passkey.name}
      meta={`${passkey.host} · Added ${clock(passkey.createdAt)}`}
      trailing={
        <Button
          size="small"
          look="fail"
          aria-label={`Delete ${passkey.name}`}
          busy={remove.isPending}
          onClick={() => {
            void remove.mutateAsync({ id: passkey.id }).then(onDeleted, (error: unknown) => {
              onFailed(error instanceof Error ? error.message : String(error));
            });
          }}
        >
          Delete
        </Button>
      }
    />
  );
}

// A taken Account may carry the taker's passkey, so the person sees every
// passkey and removes any they don't recognise.
export function Passkeys({ flow, next }: StepProps<Flow>) {
  const [kept, setKept] = useState(flow.passkeys);
  const [failure, setFailure] = useState<string>();
  return (
    <>
      <p>These passkeys still sign in as you. Delete any you don&apos;t recognise.</p>
      {kept.length === 0 ? (
        <p className="text-ink-3">You have no passkeys left.</p>
      ) : (
        <List label="Passkeys">
          {kept.map((passkey) => (
            <PasskeyRow
              key={passkey.id}
              passkey={passkey}
              onDeleted={() => {
                setFailure(undefined);
                setKept((rows) => rows.filter((row) => row.id !== passkey.id));
              }}
              onFailed={setFailure}
            />
          ))}
        </List>
      )}
      {failure !== undefined && <AlertLine text={failure} />}
      <Button
        look="primary"
        onClick={() => {
          next({ passkeys: kept });
        }}
      >
        Continue
      </Button>
    </>
  );
}
