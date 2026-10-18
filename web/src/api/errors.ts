import type { FieldError } from "./gen/model";

type ProblemBody = {
  status: number;
  code: string;
  title: string;
  detail?: string;
  errors?: FieldError[];
};

export class ApiError<C extends string = string> extends Error {
  override readonly name = "ApiError";
  readonly status: number;
  readonly code: C;
  readonly title: string;
  readonly detail: string | undefined;
  readonly errors: readonly FieldError[];
  /** The whole body, members past the Problem's own included, for a refusal that carries data. */
  readonly body: unknown;

  constructor(problem: ProblemBody & { code: C }, body: unknown = problem) {
    super(problem.title);
    this.body = body;
    this.status = problem.status;
    this.code = problem.code;
    this.title = problem.title;
    this.detail = problem.detail;
    this.errors = problem.errors ?? [];
  }
}

export class UnreachableError extends Error {
  override readonly name = "UnreachableError";

  constructor(options?: ErrorOptions) {
    super("This Node didn't answer", options);
  }
}

export type CodeOf<P> = P extends { code?: infer C } ? Extract<C, string> : never;

type CodesIn<E> = unknown extends E ? string : E extends ApiError<infer C> ? C : never;

export const isApiError = (error: unknown): error is ApiError => error instanceof ApiError;

export const isProblem = <E, C extends CodesIn<E>>(error: E, code: C): error is E & ApiError<C> =>
  error instanceof ApiError && error.code === code;

export function fieldErrors(error: ApiError): Record<string, string> {
  return Object.fromEntries(
    error.errors.map(({ location, message }) => [location.replace(/^body\./u, ""), message]),
  );
}

/** What a failed call tells the person: the problem's detail or title, or that the Node didn't answer. */
export function sentenceOf(error: unknown): string {
  if (isApiError(error)) {
    return error.detail ?? error.title;
  }
  if (error instanceof UnreachableError) {
    return error.message;
  }
  throw error;
}
