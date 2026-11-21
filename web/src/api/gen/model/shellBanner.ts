export interface ShellBanner {
  /** The name of each Node whose torrents Reconciling waits on */
  nodes: string[];
  /** Whether the old Controller's folders still need finding. The shell opens that step until it is saved */
  rehome: boolean;
  /**
   * How many library roots Reconciling waits on to finish scanning
   * @minimum 0
   */
  roots: number;
}
