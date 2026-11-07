import { z } from "zod";

import { isProblem, sentenceOf } from "../../api/errors";
import type { IndexersChallengeSolver } from "../../api/gen/model";
import { submitErrors } from "../../ui/form";

export const solverSchema = z.object({
  address: z
    .string()
    .trim()
    .min(1, "Enter the solver's address")
    .max(2048, "Use 2,048 characters or fewer"),
  timeout: z
    .string()
    .trim()
    .regex(/^\d{1,3}$/u, "Enter a number from 10 to 180")
    .refine(
      (value) => Number(value) >= 10 && Number(value) <= 180,
      "Enter a number from 10 to 180",
    ),
});

export type SolverValues = z.input<typeof solverSchema>;

export const valuesOf = (solver: IndexersChallengeSolver): SolverValues => ({
  address: solver.address ?? "",
  timeout: String(solver.timeout),
});

/** The Node can't reach the address: its sentence names both, so it shows under the address. */
export function refusal(error: unknown) {
  return isProblem(error, "challenge_solver_unreachable")
    ? { fields: { address: sentenceOf(error) } }
    : submitErrors(error);
}
