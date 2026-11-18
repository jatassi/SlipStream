import { Tick02Icon } from "@hugeicons/core-free-icons";
import { useState } from "react";

import { DialogActions } from "../account/dialog-actions";
import { sentenceOf } from "../api/errors";
import type {
  SubtitlesCandidate,
  SubtitlesCandidateReasonsItem,
  SubtitlesCandidates,
  SubtitlesOutcome,
  SubtitlesProvider,
} from "../api/gen/model";
import { pickSubtitleCandidate } from "../api/gen/query/subtitles/subtitles";
import { useNow } from "../tasks/use-now";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Chip } from "../ui/chip";
// oxlint-disable-next-line import/max-dependencies -- the sheet reads the search, picks and follows the attempt it starts
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Icon } from "../ui/icon";
import { Skeleton } from "../ui/skeleton";
import { Spinner } from "../ui/spinner";
import type { SubtitleSubject } from "./search-host";
import { AttemptLine, useAttemptLine } from "./attempt";
import { useCandidates } from "./work-api";
import { matchWords, outcomeWords, reasonWords, triedWords } from "./sheet-words";
import { providerName } from "./work-words";

const steps = Object.keys(reasonWords).filter((step): step is SubtitlesCandidateReasonsItem =>
  Object.hasOwn(reasonWords, step),
);

/** How each provider's search ended, where it found nothing or couldn't search. */
function Notes({
  outcomes,
  providers,
}: {
  outcomes: readonly SubtitlesOutcome[];
  providers: readonly SubtitlesProvider[];
}) {
  const clock = { now: useNow(60_000), zone: undefined };
  return outcomes
    .filter(({ kind }) => kind !== "ok")
    .map((outcome) => (
      <p key={outcome.provider} className="text-meta text-ink-3">
        {outcomeWords(outcome, providers, clock)}
      </p>
    ));
}

function Steps({ reasons }: { reasons: readonly SubtitlesCandidateReasonsItem[] }) {
  return (
    <ul aria-label="Ranking steps" className="flex flex-wrap gap-x-4 gap-y-1 text-meta">
      {steps.map((step) => {
        const holds = reasons.includes(step);
        return (
          <li
            key={step}
            className={`flex items-center gap-1 ${holds ? "text-ink-1" : "text-ink-3"}`}
          >
            {holds && <Icon icon={Tick02Icon} className="size-3.5" />}
            <span>
              {reasonWords[step]}
              {!holds && <span className="sr-only">, doesn&apos;t hold</span>}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

type CardProps = {
  readonly candidate: SubtitlesCandidate;
  readonly best: boolean;
  readonly providers: readonly SubtitlesProvider[];
  readonly busy: boolean;
  readonly locked: boolean;
  readonly onPick: () => void;
};

function Card({ candidate, best, providers, busy, locked, onPick }: CardProps) {
  const now = useNow(60_000);
  const { tried } = candidate;
  return (
    <li className="flex flex-col gap-2 px-4 py-3">
      <div className="flex min-w-0 items-start gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 items-center gap-2">
            {best && <Chip look="outline">Best</Chip>}
            <Data className="truncate text-ink-1">{candidate.release}</Data>
          </span>
          <span className="text-meta text-ink-3">
            {providerName(providers, candidate.provider)}
            {tried !== null && ` · ${triedWords(tried, now)}`}
            {tried !== null && tried.match !== null && ` · ${matchWords(tried.match)}`}
          </span>
        </div>
        <Button
          size="small"
          busy={busy}
          disabled={candidate.disabled || locked}
          aria-label={`Pick ${candidate.release} from ${providerName(providers, candidate.provider)}`}
          onClick={onPick}
        >
          Pick
        </Button>
      </div>
      <Steps reasons={candidate.reasons} />
    </li>
  );
}

function usePick(subject: SubtitleSubject, searchId: string, onClose: () => void) {
  const [picked, setPicked] = useState<{ key: string; attemptId: string }>();
  const [failure, setFailure] = useState<string>();
  const [sending, setSending] = useState<string>();
  const line = useAttemptLine(subject, picked?.attemptId, onClose);
  const pick = async (key: string) => {
    setSending(key);
    setFailure(undefined);
    try {
      const { attemptId } = await pickSubtitleCandidate(
        subject.fileId,
        subject.want,
        searchId,
        key,
      );
      setPicked({ key, attemptId });
    } catch (error) {
      setFailure(sentenceOf(error));
    } finally {
      setSending(undefined);
    }
  };
  const running = picked !== undefined && line.attempt?.outcome === null;
  return { picked, failure, sending, line, running, pick };
}

function CandidateList({
  label,
  candidates,
  subject,
  pick,
  firstIsBest,
}: {
  label: string;
  candidates: readonly SubtitlesCandidate[];
  subject: SubtitleSubject;
  pick: ReturnType<typeof usePick>;
  firstIsBest: boolean;
}) {
  return (
    <ul
      aria-label={label}
      className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {candidates.map((candidate, index) => (
        <Card
          key={candidate.key}
          candidate={candidate}
          best={firstIsBest && index === 0}
          providers={subject.providers}
          busy={pick.sending === candidate.key}
          locked={pick.sending !== undefined || pick.running}
          onPick={() => void pick.pick(candidate.key)}
        />
      ))}
    </ul>
  );
}

/** The candidates not tried yet, ranked, then those tried before, folded. */
function Ranked({
  candidates,
  subject,
  pick,
}: {
  candidates: readonly SubtitlesCandidate[];
  subject: SubtitleSubject;
  pick: ReturnType<typeof usePick>;
}) {
  const fresh = candidates.filter((candidate) => candidate.tried === null);
  const tried = candidates.filter((candidate) => candidate.tried !== null);
  return (
    <>
      {fresh.length > 0 && (
        <CandidateList
          label="Candidates"
          candidates={fresh}
          subject={subject}
          pick={pick}
          firstIsBest
        />
      )}
      {tried.length > 0 && (
        <Details label="Tried before">
          <CandidateList
            label="Tried before"
            candidates={tried}
            subject={subject}
            pick={pick}
            firstIsBest={false}
          />
        </Details>
      )}
    </>
  );
}

/** Whether the providers still search, how each one's search ended, and an empty result. */
function SearchLines({
  read: { candidates, outcomes, state },
  providers,
}: {
  read: SubtitlesCandidates;
  providers: readonly SubtitlesProvider[];
}) {
  return (
    <>
      {state === "searching" && (
        <output className="flex items-center gap-2 text-ink-2">
          <Spinner />
          Searching the Subtitle Providers…
        </output>
      )}
      <Notes outcomes={outcomes} providers={providers} />
      {state === "done" && candidates.length === 0 && (
        <p className="text-ink-2">No subtitle was found for this file.</p>
      )}
    </>
  );
}

/**
 * Manual subtitle search's ranked candidates, the first tagged Best, each
 * with the ranking steps it meets. Earlier tries fold into Tried before.
 * A pick shows its steps until its subtitle is added or fails.
 */
export function CandidatesBody({
  subject,
  searchId,
  onClose,
}: {
  subject: SubtitleSubject;
  searchId: string;
  onClose: () => void;
}) {
  const read = useCandidates(subject.fileId, subject.want, searchId);
  const pick = usePick(subject, searchId, onClose);
  if (read.error !== null) {
    return <AlertLine text={sentenceOf(read.error)} />;
  }
  if (read.data === undefined) {
    return <Skeleton label="Loading the candidates" />;
  }
  const { candidates } = read.data;
  return (
    <div className="flex flex-col gap-4">
      <SearchLines read={read.data} providers={subject.providers} />
      {pick.failure !== undefined && <AlertLine text={pick.failure} />}
      <AttemptLine line={pick.line} providers={subject.providers} />
      <Ranked candidates={candidates} subject={subject} pick={pick} />
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </div>
  );
}
