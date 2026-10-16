/**
 * The page's own reload, the shell's one way to start the app over. It sits
 * on an object so a test can stand in for it, since `location.reload` can't
 * be replaced.
 */
export const thisPage = {
  reload() {
    location.reload();
  },
};
