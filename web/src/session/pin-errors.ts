import { z } from "zod";

import { isProblem } from "../api/errors";
import { type SubmitErrors, submitErrors } from "../ui/form-context";
import { whenWords } from "../ui/time-words";

const waiting = z.object({ until: z.iso.datetime({ offset: true }) });

/**
 * The alerts a PIN check answers: the Node's sentence, with the time a wait
 * ends. `wrong` replaces pin_wrong's sentence where no email was typed.
 */
export function pinErrors(error: unknown, wrong?: string): SubmitErrors {
  if (wrong !== undefined && isProblem(error, "pin_wrong")) {
    return { form: wrong, fields: {} };
  }
  if (isProblem(error, "pin_wait") || isProblem(error, "too_many_attempts")) {
    const wait = waiting.safeParse(error.body);
    if (wait.success) {
      const until = whenWords(wait.data.until, "next", { now: new Date(), zone: undefined });
      return { form: `${error.title}. Try again ${until}.`, fields: {} };
    }
  }
  return submitErrors(error);
}
