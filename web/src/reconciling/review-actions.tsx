import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { ReconcilingGroup, ReconcilingReviewed } from "../api/gen/model";
import {
  forgetReconcilingTorrents,
  readdReconcilingTorrents,
  useGetReconcilingSuspense,
} from "../api/gen/query/reconciling/reconciling";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { ListRow } from "../ui/list";
import { toast } from "../ui/toast";
import { groupWords, theseWords, torrentsWords } from "./words";

const forgetTorrents = async (
  count: number,
  body: Parameters<typeof forgetReconcilingTorrents>[0],
) => {
  await forgetReconcilingTorrents(body);
  toast(`Forgot ${torrentsWords(count)}.`);
};

function useAction() {
  const reconciling = useGetReconcilingSuspense().data.restore !== null;
  const readd = async (count: number, body: Parameters<typeof readdReconcilingTorrents>[0]) => {
    await readdReconcilingTorrents(body);
    toast(
      reconciling
        ? `Re-adding ${torrentsWords(count)} once Reconciling finishes.`
        : `Re-adding ${torrentsWords(count)}.`,
    );
  };
  return { readd, forget: forgetTorrents };
}

function Press({ label, action }: { label: string; action: () => Promise<void> }) {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      size="small"
      busy={busy}
      onClick={() => {
        setBusy(true);
        action()
          .catch((error: unknown) => {
            toast(sentenceOf(error), { kind: "fail" });
          })
          .finally(() => {
            setBusy(false);
          });
      }}
    >
      {label}
    </Button>
  );
}

export function TorrentRow({ torrent, node }: { torrent: ReconcilingReviewed; node: string }) {
  const { readd, forget } = useAction();
  const meta =
    torrent.readd === null
      ? `Was on ${node}`
      : `Was on ${node} · Re-adds once Reconciling finishes`;
  return (
    <ListRow
      title={torrent.name}
      meta={meta}
      trailing={
        torrent.readd === null && (
          <span className="flex gap-2">
            <Press label="Re-add" action={() => readd(1, { ids: [torrent.id] })} />
            {torrent.reason === "not_held" && (
              <Press label="Forget" action={() => forget(1, { ids: [torrent.id] })} />
            )}
          </span>
        )
      }
    />
  );
}

function ForgetGroup({ group }: { group: ReconcilingGroup }) {
  const { forget } = useAction();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
        }}
      >
        {`Forget ${theseWords(group.count)}`}
      </Button>
      <ConfirmDialog
        title={`Forget ${torrentsWords(group.count)}`}
        confirmLabel="Forget"
        look="fail"
        open={open}
        onOpenChange={setOpen}
        onConfirm={() => forget(group.count, { reason: group.reason, found: group.found })}
      >
        <p>
          SlipStream removes their records and will look for their files again. If a Node later has
          one of them, SlipStream takes it off this list.
        </p>
      </ConfirmDialog>
    </>
  );
}

export function GroupAction({ group }: { group: ReconcilingGroup }) {
  const { readd } = useAction();
  const { readdAll } = groupWords(group);
  if (readdAll) {
    return (
      <Press
        label={`Re-add ${theseWords(group.count)}`}
        action={() => readd(group.count, { reason: group.reason, found: group.found })}
      />
    );
  }
  return group.reason === "not_held" && group.found === "no_files" ? (
    <ForgetGroup group={group} />
  ) : null;
}
