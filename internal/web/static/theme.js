// Applies the saved or system color theme before the first paint.
(() => {
  let theme = null;
  try {
    theme = window.localStorage.getItem("mavis.theme.v1");
  } catch {
    // Without storage the system preference decides.
  }
  if (theme !== "light" && theme !== "dark") {
    theme = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  document.documentElement.dataset.theme = theme;
})();
