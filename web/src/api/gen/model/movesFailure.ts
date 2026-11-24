import type { MovesFailureCode } from "./movesFailureCode.ts";

export interface MovesFailure {
  code: MovesFailureCode;
  words: string;
}
