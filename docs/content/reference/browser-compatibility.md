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
- File and folder icons distinguish entry types. Folder arrow buttons expose their expanded state and load children on
  demand without changing the open document; folder names remain ordinary navigation links.
- Expanded folder paths and desktop navigation width persist in local browser storage where available, with an in-page
  fallback when storage is denied. Current-document ancestry is revealed regardless of saved state and is not itself
  saved as a preference. Without JavaScript, server-rendered root entries, active ancestry, and folder links provide
  navigation.
- Desktop navigation resizes from 240 to 480 pixels with a vertical separator. Drag the separator, or focus it and use
  **Left Arrow** and **Right Arrow** for 16-pixel increments; **Home** and **End** select the bounds. In windows narrower
  than 960 pixels the upper bound drops so the reading area keeps at least 480 pixels; the saved width returns when the
  window widens. The control exposes its current, minimum, and maximum values to assistive technology and is hidden on
  narrow screens.
- File and folder names remain on one line. Wide names and nested trees scroll horizontally within the file browser,
  not the page.
- At 1440×1000 and 390×844, verification confirmed document navigation, duplicate-heading fragments, drawer dismissal,
  and no horizontal page overflow.

## Accessibility

All navigation and controls are keyboard operable. Use **Tab** and **Shift+Tab** to move between controls, **Enter** to
activate links and buttons, and **Space** to activate folder expansion buttons. In Safari's default keyboard
configuration, use **Option+Tab** to include links; enabling full keyboard
access also allows ordinary Tab navigation. **Escape** dismisses mobile file navigation.

The mobile drawer moves focus to its Close button, keeps Tab navigation inside the drawer while open, and restores focus
to the Files button when dismissed. Closed mobile navigation is inert, as is the main reading area while the drawer is
open. A skip link moves focus directly to the main content.

The interface also provides:

- a visible 3-pixel focus outline;
- native buttons and links, with at least 44-by-44-pixel mobile targets for the drawer controls, folder expansion
  buttons, and file-browser rows;
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

Verification on **November 25, 2026** covered H1 titles and the TOC, keyboard-operated front matter, escaped text and
raw responses, the image viewer, PDF viewer markup and framing headers, folder controls and persistence, stale paths,
unsaved revealed ancestry, pointer and keyboard resizing, the narrow-window width limit, long-name overflow,
current-document state, mobile target sizes, mobile focus trapping with collapsed folders, Escape/focus restoration, and
reduced motion. Desktop checks used 1440×1000 and 780×900; mobile checks used 390×844.

| Browser/engine | Tested version | Automation | Result |
| --- | --- | --- | --- |
| Chrome | 158.0.8127.38 | Playwright 1.65.1 | Passed, 12/12 check groups |
| Microsoft Edge | 158.0.4370.33 | Playwright 1.65.1 | Passed, 12/12 check groups |
| Firefox | 159.0 | Playwright 1.65.1 | Passed, 12/12 check groups |
| Safari engine | WebKit 27.2 | Playwright 1.65.1 | Passed, 12/12 check groups |

Safari engine coverage uses Playwright WebKit, not a separate WebDriver run of the Safari application. No native
Safari-driver result is claimed. WebKit's pointer-focus behavior was included in the Escape/focus-restoration checks.

## Full-text search

Search verification on **November 25, 2026** used Playwright 1.65.1 at 1440×1000 and 390×844. Every engine completed
keyboard-only searches with JavaScript enabled and disabled, retained the query, displayed path-ordered filename links,
root-relative paths, escaped contextual excerpts, and highlighted matches outside the file browser. The search region
spanned the viewport above the header, controls met the 44-pixel target minimum, and neither home nor results pages had
horizontal overflow.

## Interface redesign

The redesign of **November 25, 2026** moved search into the top bar and added breadcrumbs, the front-matter tree,
file-type icons, and light and dark themes. It was checked manually in one Chromium build at 1440×900 and 375×812:
document, directory, and search pages in both themes, the saved theme choice across navigation, the expanded
front-matter tree, and the absence of horizontal page overflow on mobile. Controls are 36 pixels high on desktop and
44 pixels at mobile widths. The four-engine matrix and search verification above predate the redesign and have not
been repeated for it.

Mobile drawer checks confirmed that the search region and main content become inert, programmatic focus cannot enter the
search field, Tab and Shift+Tab remain within the drawer, Escape closes it, and focus returns to the Files button. Axe-core
4.11.x WCAG A/AA scans of search results and mobile navigation reported **zero violations in every engine and viewport**.

| Browser/engine | Tested version | Search result |
| --- | --- | --- |
| Chrome | 158.0.8127.38 | Passed, 4/4 check groups |
| Microsoft Edge | 158.0.4370.33 | Passed, 4/4 check groups |
| Firefox | 159.0 | Passed, 4/4 check groups |
| Safari engine | WebKit 27.2 | Passed, 4/4 check groups |

## Download menus and generated files

Download verification on **November 25, 2026** used Playwright 1.65.1 at 1440×1000 and 390×844. Each engine opened the
native menu by keyboard, displayed only applicable formats, downloaded Markdown, PDF, DOCX, and a raw attachment, and
completed desktop/mobile checks without horizontal page overflow. Axe-core 4.11.x WCAG A/AA scans of the open download
menu reported **zero violations in every engine and viewport**.

| Browser/engine | Tested version | Download result |
| --- | --- | --- |
| Chrome | 158.0.8127.38 | Passed, 3/3 check groups |
| Microsoft Edge | 158.0.4370.33 | Passed, 3/3 check groups |
| Firefox | 159.0 | Passed, 3/3 check groups |
| Safari engine | WebKit 27.2 | Passed, 3/3 check groups |

A representative German document was also opened with Poppler `pdfinfo`/`pdftotext`, macOS Quick Look, pandoc DOCX
reading, and macOS text conversion. The one-page A4 PDF retained German text, a table, lists, links, code, and a real PNG;
its SVG became a visible **Preview unavailable** image and emoji/CJK became replacement glyphs. The DOCX retained the
German text, table, lists, links, code, and embedded PNG/SVG resources. PDF layout remains simplified, not
browser-identical; DOCX text and rendering can vary by viewer.

### Inline PDF display

The matrix above checks the PDF viewer's frame, raw URL, and response headers. Whether a PDF is drawn inside that frame
depends on the browser's own PDF support, so it was checked separately with a one-page test document:

| Browser | Tested version | Inline PDF display |
| --- | --- | --- |
| Chrome | 158 | Displayed |
| Microsoft Edge | 158.0.4370.33 | Displayed |
| Firefox | 160.0.1 | Displayed |
| Safari | Not tested | Unknown |

The Playwright Firefox 159.0 and WebKit 27.2 builds used for the matrix do not display PDFs inside a frame, with or
without Mavis's response headers, so they provide no evidence about inline display. When a browser cannot display a PDF
inline, **View raw file** still opens or downloads the original document.

## Automated accessibility results

The **November 23, 2026 baseline** Axe-core 4.14.x checks used the WCAG 2.0/2.1 A and AA rule tags on a desktop
document, a mobile document, and the open mobile drawer. **All four browser/engine runs reported zero violations in
every state.** Narrow-layout runs marked some
`color-contrast` checks as requiring manual review; the explicit palette checks above passed those reviews. Chrome's
final mobile Lighthouse accessibility score was **100/100**.

These checks establish a baseline, not a guarantee against every accessibility issue. Content authors remain
responsible for meaningful link text, image alternatives, heading structure, and accessible source documents.
