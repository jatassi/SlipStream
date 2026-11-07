/**
 * Null while it runs; lost when its Node never answered
 * @nullable
 */
export type IndexersQueryOutcome = "ok" | "limited" | "failing" | "stopped" | "lost" | null;
