---
title: Browser compatibility
linkTitle: Browsers
description: Supported browsers, responsive behavior, and accessibility verification.
weight: 50
icon: desktop
---

The baseline browser interface supports current stable Chrome/Edge, Firefox, and Safari at release time. It uses
server-rendered HTML, CSS Grid, native controls, and a small JavaScript disclosure/focus manager; it does not require a
client-side framework.

## Layouts and navigation

- Desktop uses a two-pane layout: file navigation on the left and rendered content with its table of contents on the
  right.
- Mobile and narrow screens provide dismissible file navigation without horizontal page scrolling.
- Ordinary document URLs support browser Back/Forward and shareable heading fragments.
- At 1440×1000 and 390×844, verification confirmed document navigation, duplicate-heading fragments, drawer dismissal,
  and no horizontal page overflow.

## Accessibility

All navigation and controls are keyboard operable. Use **Tab** and **Shift+Tab** to move between controls and **Enter**
to activate them. In Safari's default keyboard configuration, use **Option+Tab** to include links; enabling full keyboard
access also allows ordinary Tab navigation. **Escape** dismisses mobile file navigation.

The mobile drawer moves focus to its Close button, keeps Tab navigation inside the drawer while open, and restores focus
to the Files button when dismissed. Closed mobile navigation is inert, as is the main reading area while the drawer is
open. A skip link moves focus directly to the main content.

The interface also provides:

- a visible 3-pixel focus outline;
- native buttons and links with at least 44-by-44-pixel mobile control targets;
- banner, navigation, main, article, and complementary table-of-contents landmarks;
- `aria-current="page"` on the current document;
- heading IDs and a hierarchical current-document table of contents;
- accessible labels and native checked/disabled state for task-list checkboxes;
- reduced-motion support for the drawer transition.

### Contrast

The tested palette meets WCAG AA contrast requirements. Across the declared text and control states, the minimum
measured ratios are:

| State | Minimum ratio | Requirement |
| --- | ---: | ---: |
| Normal text | 5.58:1 | 4.5:1 |
| Focus indicator | 4.91:1 | 3:1 |
| Drawer-control boundary | 3.68:1 | 3:1 |

`internal/web/accessibility_test.go` calculates these ratios from the embedded CSS palette so changes cannot silently
reduce them below the required thresholds.

## Verified browser matrix

Verification on **November 23, 2026** covered desktop and mobile layouts, keyboard-only document navigation, skip-link
activation, visible focus, drawer focus trapping, Escape/focus restoration, current-document state, and horizontal
overflow.

| Browser/engine | Tested version | Automation | Result |
| --- | --- | --- | --- |
| Chrome | 158.0.8127.38 | Native WebDriver | Passed |
| Microsoft Edge | 158.0.4370.33 | Native WebDriver | Passed |
| Firefox | 160.0.1 | Native WebDriver, geckodriver 0.37.1 | Passed |
| Safari engine | WebKit 27.2 | Playwright WebKit | Passed |

Safari engine coverage uses Playwright WebKit and Option+Tab keyboard navigation. It is not a separate WebDriver run of
the Safari application; the installed Safari 27.0.1 application had remote automation disabled during verification.
This distinction is recorded explicitly rather than claiming a native Safari-driver result.

## Automated accessibility results

Axe-core 4.14.x checks used the WCAG 2.0/2.1 A and AA rule tags on a desktop document, a mobile document, and the open
mobile drawer. **All four browser/engine runs reported zero violations in every state.** Narrow-layout runs marked some
`color-contrast` checks as requiring manual review; the explicit palette checks above passed those reviews. Chrome's
final mobile Lighthouse accessibility score was **100/100**.

These checks establish a baseline, not a guarantee against every accessibility issue. Content authors remain
responsible for meaningful link text, image alternatives, heading structure, and accessible source documents.
