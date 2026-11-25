(() => {
  const toggle = document.getElementById("navigation-toggle");
  const sidebar = document.getElementById("file-navigation");
  const closeButton = document.getElementById("navigation-close");
  const backdrop = document.getElementById("navigation-backdrop");
  const main = document.getElementById("main-content");
  const mobile = window.matchMedia("(max-width: 48rem)");

  if (!toggle || !sidebar || !closeButton || !backdrop || !main) {
    return;
  }

  let open = false;
  let returnFocus = null;

  const focusableElements = () => Array.from(
    sidebar.querySelectorAll('a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'),
  ).filter((element) => !element.hasAttribute("hidden"));

  const syncAccessibility = () => {
    const hidden = mobile.matches && !open;
    sidebar.toggleAttribute("inert", hidden);
    main.inert = mobile.matches && open;
    toggle.setAttribute("aria-expanded", String(open));
    document.body.classList.toggle("nav-open", mobile.matches && open);
  };

  const setOpen = (nextOpen, restoreFocus = true) => {
    if (!mobile.matches) {
      nextOpen = false;
    }
    if (nextOpen) {
      returnFocus = document.activeElement;
    }
    open = nextOpen;
    syncAccessibility();
    if (open) {
      closeButton.focus();
    } else if (restoreFocus && returnFocus instanceof HTMLElement) {
      returnFocus.focus();
      returnFocus = null;
    }
  };

  toggle.addEventListener("click", () => setOpen(!open));
  closeButton.addEventListener("click", () => setOpen(false));
  backdrop.addEventListener("click", () => setOpen(false));

  sidebar.addEventListener("click", (event) => {
    if (mobile.matches && event.target.closest("a[href]")) {
      setOpen(false, false);
    }
  });

  document.addEventListener("keydown", (event) => {
    if (!mobile.matches || !open) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setOpen(false);
      return;
    }
    if (event.key !== "Tab") {
      return;
    }
    const elements = focusableElements();
    if (elements.length === 0) {
      event.preventDefault();
      closeButton.focus();
      return;
    }
    const first = elements[0];
    const last = elements[elements.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  const syncMode = () => {
    const focusWasInSidebar = sidebar.contains(document.activeElement);
    open = false;
    returnFocus = null;
    syncAccessibility();
    if (focusWasInSidebar) {
      main.focus();
    }
  };

  mobile.addEventListener("change", syncMode);
  syncMode();
})();
