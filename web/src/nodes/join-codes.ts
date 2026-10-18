/** The open Join Codes and each code's state, which a Pair, a mint or a revoke changes. */
export const everyJoinCodeQuery = {
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && /^\/api\/v1\/join-codes(?:\/|$)/u.test(path),
};
