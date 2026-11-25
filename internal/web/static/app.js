(() => {
  const toggle = document.getElementById("navigation-toggle");
  const sidebar = document.getElementById("file-navigation");
  const closeButton = document.getElementById("navigation-close");
  const backdrop = document.getElementById("navigation-backdrop");
  const main = document.getElementById("main-content");
  const mobile = window.matchMedia("(max-width: 48rem)");
  const preferenceMemory = new Map();

  const readPreference = (key) => {
    if (preferenceMemory.has(key)) {
      return preferenceMemory.get(key);
    }
    try {
      const value = window.localStorage.getItem(key);
      if (value !== null) {
        preferenceMemory.set(key, value);
      }
      return value;
    } catch {
      return null;
    }
  };

  const writePreference = (key, value) => {
    preferenceMemory.set(key, value);
    try {
      window.localStorage.setItem(key, value);
    } catch {
      // The in-memory copy keeps preferences working when storage is unavailable.
    }
  };

  if (!toggle || !sidebar || !closeButton || !backdrop || !main) {
    return;
  }

  let open = false;
  let returnFocus = null;

  const focusableElements = () => Array.from(
    sidebar.querySelectorAll('a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'),
  ).filter((element) => !element.closest("[hidden]"));

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

  toggle.addEventListener("click", () => {
    if (!open) {
      toggle.focus();
    }
    setOpen(!open);
  });
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

  const resizer = document.getElementById("sidebar-resizer");
  if (resizer) {
    const sidebarWidthKey = "mavis.sidebar-width.v1";
    const defaultWidth = 304;
    const minimumWidth = 240;
    const maximumWidth = 480;
    const minimumContentWidth = 480;
    const widthStep = 16;
    let dragging = false;

    // Narrow windows lower the limit so the reading area stays usable.
    const availableMaximum = () => Math.max(
      minimumWidth,
      Math.min(maximumWidth, window.innerWidth - minimumContentWidth),
    );

    const storedWidth = Number.parseInt(readPreference(sidebarWidthKey) || "", 10);
    let preferredWidth = Math.min(maximumWidth, Math.max(minimumWidth, Number.isFinite(storedWidth) ? storedWidth : defaultWidth));

    // The saved preference survives a temporarily narrow window.
    const applySidebarWidth = () => {
      const limit = availableMaximum();
      const width = Math.min(preferredWidth, limit);
      document.documentElement.style.setProperty("--sidebar-width", `${width}px`);
      resizer.setAttribute("aria-valuemax", String(limit));
      resizer.setAttribute("aria-valuenow", String(width));
    };

    const setSidebarWidth = (requestedWidth) => {
      preferredWidth = Math.min(availableMaximum(), Math.max(minimumWidth, Math.round(requestedWidth)));
      writePreference(sidebarWidthKey, String(preferredWidth));
      applySidebarWidth();
    };

    applySidebarWidth();
    window.addEventListener("resize", applySidebarWidth);

    resizer.addEventListener("pointerdown", (event) => {
      if (mobile.matches || event.button !== 0) {
        return;
      }
      event.preventDefault();
      dragging = true;
      resizer.dataset.dragging = "true";
      resizer.setPointerCapture(event.pointerId);
      setSidebarWidth(event.clientX);
    });

    resizer.addEventListener("pointermove", (event) => {
      if (dragging) {
        setSidebarWidth(event.clientX);
      }
    });

    const stopDragging = (event) => {
      if (!dragging) {
        return;
      }
      dragging = false;
      delete resizer.dataset.dragging;
      if (resizer.hasPointerCapture(event.pointerId)) {
        resizer.releasePointerCapture(event.pointerId);
      }
    };
    resizer.addEventListener("pointerup", stopDragging);
    resizer.addEventListener("pointercancel", stopDragging);

    resizer.addEventListener("keydown", (event) => {
      if (mobile.matches) {
        return;
      }
      const currentWidth = Number.parseInt(resizer.getAttribute("aria-valuenow"), 10);
      let nextWidth = currentWidth;
      if (event.key === "ArrowLeft") {
        nextWidth -= widthStep;
      } else if (event.key === "ArrowRight") {
        nextWidth += widthStep;
      } else if (event.key === "Home") {
        nextWidth = minimumWidth;
      } else if (event.key === "End") {
        nextWidth = maximumWidth;
      } else {
        return;
      }
      event.preventDefault();
      setSidebarWidth(nextWidth);
    });
  }

  const expandedFoldersKey = "mavis.expanded-folders.v1";
  const expandedFolders = new Set();
  try {
    const stored = JSON.parse(readPreference(expandedFoldersKey) || "[]");
    if (Array.isArray(stored)) {
      stored.filter((entry) => typeof entry === "string").forEach((entry) => expandedFolders.add(entry));
    }
  } catch {
    // Invalid preferences are equivalent to no saved expansion state.
  }

  const saveExpandedFolders = () => {
    writePreference(expandedFoldersKey, JSON.stringify(Array.from(expandedFolders).sort()));
  };

  const folderElements = (button) => {
    const item = button.closest(".navigation-item");
    return {
      item,
      children: item ? item.querySelector(":scope > .navigation-children") : null,
      name: item ? item.querySelector(":scope > .navigation-row .navigation-name") : null,
    };
  };

  const syncFolder = (button, expanded) => {
    const { children, name } = folderElements(button);
    button.setAttribute("aria-expanded", String(expanded));
    button.setAttribute("aria-label", `${expanded ? "Collapse" : "Expand"} ${name ? name.textContent : "folder"}`);
    const arrow = button.querySelector(".navigation-arrow");
    if (arrow) {
      arrow.textContent = expanded ? "▾" : "▸";
    }
    if (children) {
      children.hidden = !expanded;
    }
  };

  const createNavigationIcon = (directory) => {
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "navigation-icon");
    svg.setAttribute("viewBox", "0 0 16 16");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");
    const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("d", directory
      ? "M1.5 3.5h4l1.5 2h7.5v7h-13z"
      : "M3.5 1.5h5l4 4v9h-9zM8.5 1.5v4h4");
    path.setAttribute("fill", "none");
    path.setAttribute("stroke", "currentColor");
    path.setAttribute("stroke-linejoin", "round");
    svg.append(path);
    return svg;
  };

  const createNavigationEntry = (entry) => {
    const item = document.createElement("li");
    item.className = "navigation-item";
    item.dataset.navigationPath = entry.path;
    const row = document.createElement("div");
    row.className = "navigation-row";

    let button = null;
    if (entry.directory) {
      button = document.createElement("button");
      button.className = "navigation-expander";
      button.type = "button";
      button.dataset.navigationPath = entry.path;
      const arrow = document.createElement("span");
      arrow.className = "navigation-arrow";
      arrow.setAttribute("aria-hidden", "true");
      button.append(arrow);
      row.append(button);
    } else {
      const placeholder = document.createElement("span");
      placeholder.className = "navigation-expander-placeholder";
      placeholder.setAttribute("aria-hidden", "true");
      row.append(placeholder);
    }

    const link = document.createElement("a");
    link.className = "navigation-link";
    link.href = entry.url;
    link.append(createNavigationIcon(entry.directory));
    const name = document.createElement("span");
    name.className = "navigation-name";
    name.textContent = entry.name;
    link.append(name);
    row.append(link);
    item.append(row);

    if (entry.directory) {
      const children = document.createElement("ul");
      children.className = "navigation-children";
      children.hidden = true;
      item.append(children);
      syncFolder(button, false);
    }
    return { item, button };
  };

  const loadFolder = async (button) => {
    const { children } = folderElements(button);
    if (!children || children.dataset.loaded === "true" || button.getAttribute("aria-busy") === "true") {
      return;
    }
    const relativePath = button.dataset.navigationPath || "";
    button.setAttribute("aria-busy", "true");
    try {
      const response = await fetch(`/_mavis/navigation?path=${encodeURIComponent(relativePath)}`, {
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        throw new Error("Folder is no longer available");
      }
      const result = await response.json();
      if (!result || !Array.isArray(result.entries)) {
        throw new Error("Invalid navigation response");
      }
      const loadedButtons = [];
      result.entries.forEach((entry) => {
        if (!entry || typeof entry.name !== "string" || typeof entry.path !== "string" ||
            typeof entry.url !== "string" || typeof entry.directory !== "boolean") {
          return;
        }
        const { item, button: childButton } = createNavigationEntry(entry);
        children.append(item);
        if (childButton) {
          loadedButtons.push(childButton);
        }
      });
      children.dataset.loaded = "true";
      loadedButtons.forEach(restoreFolder);
    } catch {
      expandedFolders.delete(relativePath);
      saveExpandedFolders();
      syncFolder(button, false);
    } finally {
      button.removeAttribute("aria-busy");
    }
  };

  const expandFolder = (button) => {
    const { children } = folderElements(button);
    const relativePath = button.dataset.navigationPath || "";
    expandedFolders.add(relativePath);
    saveExpandedFolders();
    syncFolder(button, true);
    if (children && children.dataset.loaded !== "true") {
      void loadFolder(button);
    }
  };

  function restoreFolder(button) {
    const { children } = folderElements(button);
    const relativePath = button.dataset.navigationPath || "";
    const serverExpanded = button.getAttribute("aria-expanded") === "true";
    if (serverExpanded) {
      // Revealed ancestry is not a user choice, so it is not saved.
      if (children) {
        children.dataset.loaded = String(children.childElementCount > 0);
      }
      syncFolder(button, true);
    } else if (expandedFolders.has(relativePath)) {
      expandFolder(button);
    } else {
      syncFolder(button, false);
    }
  }

  sidebar.addEventListener("click", (event) => {
    const button = event.target.closest(".navigation-expander");
    if (!button || !sidebar.contains(button) || button.getAttribute("aria-busy") === "true") {
      return;
    }
    const relativePath = button.dataset.navigationPath || "";
    if (button.getAttribute("aria-expanded") === "true") {
      expandedFolders.delete(relativePath);
      saveExpandedFolders();
      syncFolder(button, false);
    } else {
      expandFolder(button);
    }
  });

  sidebar.querySelectorAll(".navigation-expander").forEach(restoreFolder);
  saveExpandedFolders();
})();
