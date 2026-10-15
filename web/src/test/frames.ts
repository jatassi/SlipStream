// Waits real animation frames, which a fake clock leaves running.
export function frames(count: number, onFrame: (frame: number) => void = () => {}) {
  return new Promise<void>((resolve) => {
    let frame = 0;
    const tick = () => {
      frame += 1;
      onFrame(frame);
      if (frame === count) {
        resolve();
      } else {
        requestAnimationFrame(tick);
      }
    };
    requestAnimationFrame(tick);
  });
}
