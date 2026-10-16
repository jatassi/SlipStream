export interface SystemRestarting {
  /** The port the stopping process serves on */
  boundPort: number;
  /** The port the successor binds, when http.listen waits for this restart */
  nextPort?: number;
  /** When the stopping process started; its successor's startedAt is later */
  startedAt: string;
}
