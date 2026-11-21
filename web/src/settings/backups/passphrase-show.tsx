import { useQuery, useQueryClient } from "@tanstack/react-query";

import { sentenceOf } from "../../api/errors";
import { getBackupPassphrase } from "../../api/gen/query/backups/backups";
import { hidePassphrase, shownPassphraseKey } from "../../live/adopt-backups";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";

// No abort signal, so the read outlives a remount, and no garbage-collection
// grace, so the words go once nothing shows them.
export function useShownPassphrase() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: shownPassphraseKey,
    queryFn: () => getBackupPassphrase(),
    enabled: false,
    gcTime: 0,
  });
  return {
    shown: query.data?.passphrase,
    busy: query.isFetching,
    failure: query.error === null ? null : <AlertLine text={sentenceOf(query.error)} />,
    show: () => void query.refetch(),
    hide: () => {
      hidePassphrase(queryClient);
    },
  };
}

export function ShowButton({ passphrase }: { passphrase: ReturnType<typeof useShownPassphrase> }) {
  return (
    <>
      {passphrase.shown === undefined ? (
        <Button busy={passphrase.busy} onClick={passphrase.show}>
          Show passphrase
        </Button>
      ) : (
        <Button onClick={passphrase.hide}>Hide passphrase</Button>
      )}
      {passphrase.failure}
    </>
  );
}
