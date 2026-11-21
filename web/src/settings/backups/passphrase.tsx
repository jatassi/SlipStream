import type { BackupsPassphraseState } from "../../api/gen/model";
import { saveBackupPassphrase } from "../../api/gen/query/backups/backups";
import { dateOf } from "../../format";
import { useAttempt } from "../../targets/attempt";
import { Button } from "../../ui/button";
import { ReadOnlyField } from "../../ui/read-only-field";
import { Section } from "../../ui/section";
import { downloadSheet } from "./downloads";
import { ReplaceButton } from "./passphrase-replace";
import { ShowButton, useShownPassphrase } from "./passphrase-show";

export const passphraseAbout =
  "SlipStream keeps a copy and uses it for every scheduled Backup. You need this passphrase only to restore a Backup on another machine. Store it in a password manager or print it.";

export function ShownPassphrase({ passphrase }: { passphrase: string }) {
  return (
    <div className="flex flex-col gap-2">
      <ReadOnlyField label="Backup passphrase" value={passphrase} />
      <p className="text-ink-2">{passphraseAbout}</p>
    </div>
  );
}

const savedWords = (state: BackupsPassphraseState) =>
  state.savedAt === null
    ? "You haven't confirmed that you saved this passphrase."
    : `You confirmed that you saved this passphrase on ${dateOf(state.savedAt)}.`;

export function PassphraseSection({ state }: { state: BackupsPassphraseState }) {
  const passphrase = useShownPassphrase();
  const { shown } = passphrase;
  const download = useAttempt(downloadSheet);
  const saved = useAttempt(saveBackupPassphrase);
  return (
    <Section title="Backup passphrase">
      <div className="flex flex-col items-start gap-3">
        <p className="text-ink-2">{savedWords(state)}</p>
        {shown !== undefined && (
          <div className="self-stretch">
            <ShownPassphrase passphrase={shown} />
          </div>
        )}
        {download.failure}
        {saved.failure}
        <div className="flex flex-wrap items-center gap-2">
          <ShowButton passphrase={passphrase} />
          <Button busy={download.busy} onClick={download.run}>
            Download recovery sheet
          </Button>
          {state.savedAt === null && (
            <Button busy={saved.busy} onClick={saved.run}>
              I saved it
            </Button>
          )}
          <ReplaceButton replaced={passphrase.hide} />
        </div>
      </div>
    </Section>
  );
}
