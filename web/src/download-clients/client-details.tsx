import type {
  DownloadclientsDaemon as Daemon,
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import { Details } from "../ui/details";
import { ClientFoldersField } from "./client-folders-field";
import type { ClientValues } from "./client-form";
import { DaemonField, onlyDaemon } from "./daemon-field";
import { Timings } from "./timings";
import type { ClientForm } from "./use-client-sheet";

const labelHints: { readonly [L in Kind["label"]]: string } = {
  any: "",
  no_comma: " Use a label without a comma.",
  lower_word: " Use a-z, 0-9, _ and - only.",
};

function summaryOf(values: ClientValues, kind: Kind, node: string): string {
  const folder = values.folder.trim();
  const saves = `SlipStream labels its torrents ${values.label.trim()}, and ${kind.name} saves them in ${folder === "" ? "its own download folder" : folder}.`;
  const pairs = values.clientFolders.length;
  if (pairs === 0) {
    return `${saves} ${node} opens the paths ${kind.name} reports as they are.`;
  }
  const folders = pairs === 1 ? "1 Client folder" : `${pairs} Client folders`;
  return `${saves} ${node} opens ${kind.name}'s paths through ${folders}.`;
}

/** The label SlipStream sets and the folder the client saves to. */
function LabelAndFolder({ form, kind }: { form: ClientForm; kind: Kind }) {
  return (
    <>
      <form.AppField name="label">
        {(field) => (
          <field.TextField
            label="Label"
            autoComplete="off"
            spellCheck={false}
            description={`SlipStream sets it on every torrent it adds to ${kind.name}.${labelHints[kind.label]}`}
          />
        )}
      </form.AppField>
      <form.AppField name="folder">
        {(field) => (
          <field.TextField
            label="Download folder"
            data
            spellCheck={false}
            description={`The folder as ${kind.name} names it. Leave it empty for ${kind.name}'s own download folder. SlipStream adds a folder for each Content Type inside it.`}
          />
        )}
      </form.AppField>
    </>
  );
}

const timingsLater =
  "Check every and Import delay start at their defaults. Change them once the client is saved.";

/** The machinery under Details: label, download folder, Client folders, Deluge's daemon and the timings. */
export function ClientDetails({
  form,
  kind,
  node,
  saved,
  daemons,
  open,
  onOpenChange,
}: {
  form: ClientForm;
  kind: Kind;
  node: string;
  saved: External | undefined;
  daemons: readonly Daemon[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <form.Subscribe selector={(state) => state.values}>
      {(values) => (
        <Details
          fields
          open={open}
          onOpenChange={onOpenChange}
          summary={summaryOf(values, kind, node)}
        >
          <div className="mt-2 flex flex-col gap-4">
            <LabelAndFolder form={form} kind={kind} />
            <ClientFoldersField form={form} kind={kind} node={node} />
            {kind.daemon && (daemons.length > 1 || values.daemon !== onlyDaemon) && (
              <DaemonField form={form} daemons={daemons} />
            )}
            {saved === undefined ? (
              <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
                {timingsLater}
              </p>
            ) : (
              <Timings client={saved} />
            )}
          </div>
        </Details>
      )}
    </form.Subscribe>
  );
}
