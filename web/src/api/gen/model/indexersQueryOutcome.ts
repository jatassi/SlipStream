/**
 * Null while it runs; lost when its Node never responded
 * @nullable
 */
export type IndexersQueryOutcome = "ok" | "limited" | "failing" | "stopped" | "lost" | null;
