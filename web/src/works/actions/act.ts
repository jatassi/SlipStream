import { toast } from "../../ui/toast";
import { type ActionKind, type AnswerOf, type Subject, type WorkAction, entryOf } from "./actions";
import { outcomeWords } from "./outcomes";

/**
 * Runs an action on the Works in one request and toasts how it went for
 * each. A refused request throws, so the dialog or button that asked can
 * show why; the per-Work outcomes never do.
 */
export async function act<K extends ActionKind>(
  subjects: readonly Subject[],
  action: WorkAction<K>,
): Promise<AnswerOf<K>> {
  const answer = await entryOf(action).post([...new Set(subjects.map(({ id }) => id))], action);
  const { done, failed } = outcomeWords(action, answer.outcomes, subjects);
  if (done !== undefined) {
    toast(done);
  }
  for (const sentence of failed) {
    toast(sentence, { kind: "fail" });
  }
  return answer;
}
