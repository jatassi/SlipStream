export interface GrabsGrabbing {
  /**
   * The Releases grabbed now. 0 when nothing waited, because the wait already ended
   * @minimum 0
   */
  grabbing: number;
}
