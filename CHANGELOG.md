# Changelog

## [Unreleased]
### Added
- share a bookmark by email
  - send the bookmark's content as a message
  - send the bookmark as an EPUB attachment
- mosaic view (image wall on bookmark list)
- browser extension calls to action (onboarding and sidebar)
- author to the exported EPUB, by [@ruben](https://codeberg.org/ruben)

### Changed
- Reverted the SQLite driver to its CGO version

### Removed
- Application passwords. API token can now be used in HTTP Basic Authentication when needed

### Fixed
- Bug in Wallabag import adapter that would not load entries with empty headers

### Improved
- Check Origin header in CSRF protection
- Strings that were misspelled or difficult to translate

## [0.18.2] - 2025-05-07
### Added
- server.base_url setting

### Fixed
- Extraction error when a page contains empty link header value

## [0.18.1] - 2025-04-30
### Fixed
- Fatal error on "About Readeck" page with fresh install
- Pocket import adapter
- Bug with contiguous highlights not displaying properly (or at all)
- `href` value in bookmark list (API)

### Improved
- Use current list ordering when exporting EPUBs, by [@knightpp](https://codeberg.org/knightpp)
- Removed forced server port in container image, by [@martenlienen](https://codeberg.org/martenlienen)
- Removed AMD64=v2 build optimisation

## [0.18.0] - 2025-04-12
### Added
- Export command
- Import command
- Import from CSV and Instapaper
- Last used dates on API Tokens and Application Passwords
- Show disk usage in the "about Readeck" page, by [@Pontoporeia](https://codeberg.org/Pontoporeia)
- Spanish (español de España) translation
- Galician (galego) translation

### Improved
- Mark imported archived bookmarks as read
- Option to archive and/or mark read all imported bookmarks
- Reading time pluralization, by [@zull](https://codeberg.org/zull)
- Case insensitive label suggestions
- Exhaustive and restrictive Permissions-Policy headers
- Extraction of pages on Wordpress with the activitypub plugin
- Extraction of pages with a Link `<url>; rel="original"` header

### Changed
- Updated Go to 1.24
- New SQLite, CGO-free driver. Important: it won't run with `MemoryDenyWriteExecute=yes` in the systemd service file.
- New internal form API
- New secure cookie for session and CSRF protection

### Fixed
- Safari display bug with "details" HTML elements in bookmarks
- Regression with Safari when creating highlights

### Breaking changes
- All bookmark internal URLs have changed
- New API token format. Previous token format won't work anymore.
- New secret key derivation scheme. Application password must be refreshed.
- When using the browser extension, you must logout and login again.
- `MemoryDenyWriteExecute=yes` must be removed from the systemd service (see above).

## [0.17.1] - 2025-01-15
### Fixed
- Username validation during onboarding

### Changed
- Build with Go 1.23.4

## [0.17.0] - 2025-01-15
### Added
- Reading progress tracker on articles
- Filter bookmarks by reading progress
- Filter bookmarks by more than one type
- `user` sub command to create or update users, by [@algernon](https://codeberg.org/algernon)
- Colored highlights, by [@makebit](https://codeberg.org/makebit)
- Adjustable content's width on bookmark view
- Added word_count and reading_time to bookmark api, by [@eleith](https://codeberg.org/eleith)

### Fixed
- Search query with backslash would be parsed incorrectly

### Improved
- Only load translation that are at least 90% completed
- Resize images by maximum width only
- Bookmark display with wide tables, by [@makebit](https://codeberg.org/makebit)
- Container based responsive layout

### Changed
- New, native, logging library
- Bigger card icons on desktop and even bigger on touch screens

## [0.16.0] - 2024-11-27
### Added
- Content Script for 404media.co
- Permit removal of a loading bookmark
- Filter bookmarks with labels, with errors and by loading state
- Show bookmark creation date on the sidebar
- Alignment and hyphenation reader settings
- Use alignment and hyphenation settings in EPUB export, by [@bramaudi](https://codeberg.org/bramaudi)

### Fixed
- Prevent the sort menu to go offscreen on mobile
- Identify a photo type by using the DCMI type value

### Improved
- Hide top navigation when scrolling down
- Sliding menu on mobile
- Bookmark sidebar can be hidden on all screen size
- Normal size of label input field, by [@lilymara](https://codeberg.org/lilymara)
- Better User-Agent for nytimes.com retrieval

### Changed
- `trusted_proxies` setting. This list the proxy that are authorized to set `X-Forwarded-...` headers. The default list provides sensible defaults so it's not a major breaking change.
- Updated site-config rules


## [0.15.6] - 2024-11-02
### Fixed
- Fixed label extraction from Pocket export file

## [0.15.5] - 2024-11-02
### Added
- Added Literata, by [@diginaut](https://codeberg.org/diginaut)

### Fixed
- Support of new Pocket export format

## [0.15.4] - 2024-10-30
### Added
- Import from Omnivore
- Improved Youtube video descriptions
- Support for text/plain extraction

### Fixed
- Fixed a bug with video bookmarks without an actual video content
- Fixed a typo on the onboarding page, by [@ggtr1138](https://codeberg.org/ggtr1138)

## [0.15.3] - 2024-09-7
### Fixed
- Goodlinks date format in JSON file

## [0.15.2] - 2024-09-7
### Added
- Documentation in German, by [@FrankDex](https://codeberg.org/FrankDex)
- Convert sections from browser bookmarks to labels
- Support for "tag" attribute in browser bookmarks import files (pinboard format)
- Support for "toread" attribute in browser bookmarks import files (pinboard format)
- Import from Goodlinks

### Improved
- Custom error page when a bookmark doesn't exist
- Improved selected text color in dark mode

### Fixed
- Support for t.co links, by [@joachimesque](https://codeberg.org/joachimesque)
- Fixed extraction error with fastcompany.com
- Stop emptying the resource cache when closing its entry io.Reader

## [0.15.1] - 2024-08-17
### Added
- Spanish translation, by [@vsc55](https://codeberg.org/vsc55)
- Hugarian translation, by [@User240803](https://codeberg.org/User240803)
- Chinese translation, by [@shichen437](https://codeberg.org/shichen437)

### Changed
- Updated dependencies
- Updated site-config rules

### Fixed
- German translation errors
- Only reset the user seed upon username or email address change.

## [0.15.0] - 2024-07-17
### Added
- Bookmark list ordering options
- Polish translation, by [@anarion](https://codeberg.org/anarion)
- Czech translation, by [@marapa](https://codeberg.org/marapa)
- German translation, by [@FrankDex](https://codeberg.org/FrankDex)

### Improved
- Can now change an API token name
- Extract inline SVG as remote images

### Changed
- Updated readability
- Updated site-config rules

### Fixed
- Fixed a bug with PostgreSQL installation on a different schema, by [@winston0410](https://codeberg.org/winston0410)
- Fixed a bug with the Wallabag import tool. It would fail when the provided URL is missing a trailing slash.

## [0.14.0] - 2024-05-20
### Added
- Import from other services
  - _Text file_. A simple file with one URL per line.
  - _Browser Bookmarks_. The file you can export from your browser menu. The title and date are retrieved from the file.
  - _Pocket_. From Pocket, you can export a file and upload it in Readeck import tool. Title, labels and date are retrieved from the file.
  - _Wallabag_. This uses the Wallabag API to retrieve everything, including the content the way it was initially saved.
- Breadcrumbs on profile, admin, documentation and import
- Keep the URL as submitted by the user in a new database column
- cleanup command to remove all pending (loading state) bookmarks
- Optional compact layout for bookmark lists
- Unified bookmark list header with layout switcher and export menu
- Publicly shared link expiration can be configured (instance wide)
- Direct link to the original URL on every bookmark card

### Changed
- Major internal/bookmarks refactor
- Updated site-config rules
- Metrics and workers configuration
  - keep only one metrics configuration (in `[metrics]`) with environment
    overriding when needed.
  - added environment configuration for the worker configuration
  - adjust the default number of workers
  - unset every `READECK_` environment variable upon startup so they won't
    leak in case of a major security breach.
- SQLite connection optimizations
- `DELETE /api/bookmarks/{id}` performs an immediate deletion (unlike the web view that schedules a task)
- Button appearance (better active state, slight change of layout)
- HTTP client configuration (TLS 1.2 and 1.3 only), chrome like cipher list

### Fixed
- Keep namespaces during SVG files cleanup
- Data race condition with the in-memory store

## [0.13.2] - 2024-03-24
### Fixed
- Healthcheck command (only connect to server, don't perform any request)

## [0.13.1] - 2024-03-09
### Fixed
- Failed migration when label list is empty

## [0.13.0] - 2024-03-09
### Added
- Internationalization
- French translation
- Bluesky content script. Thanks @joachimesque
- Healthcheck command

### Improved
- Bookmark search with wildcards and exclusions
- Preserve titles in content during extraction
- Embed video privacy
- Embed player for invidious videos
- Disable readability when saving selection from the browser addon
- More natural label ordering
- White border around QR codes so scanning works in dark mode too

## Changed
- Go 1.22
- Turbo 8
- Updated site-config rules

## [0.12.0] - 2024-02-04
### Added
- More font choices. The reader interface offers now 8 font faces
- Dark mode. A new color palette for dark mode and a theme switcher
  between auto / dark / light.
- Progressive Web App support
- Extraction of Fediverse content
- Directly save images
- HLS video player

### Improved
- Accept TLS 1.0 and negotiate more ciphers in the extractor HTTP client
- Selected font is applied on bookmark's title and description
- Regrouped user preferences and admin under the same menu item
- Mark required form fields
- Better password reveal widget

## [0.11.2] - 2024-01-13
### Improved
- Better newyorker.com configuration
- Improved reddit.com content extraction (pictures, galleries, self text, links)
- Enabled HTTP/2, compression and keep-alive on the extractor HTTP client
- Ensure images from articles are properly requested to avoid edge cases

## [0.11.1] - 2024-01-09
### Fixed
- Always keep not empty inline nodes during cleanup. Some can be used for punctuation purposes.

## [0.11.0] - 2024-01-08
### Added
- Sharing saved bookmarks. Thanks @bacardi55

### Improved
- Better display of images in lists (when used as bullets by some contents)
- Improved display of inner links in titles
- Save more images from articles
- Apply Wikipedia extraction rules to Wikinews
- Added robots rules on all the app's pages (noindex, nofollow, noarchive)
- Restrict CSP frame-src rule to video provider when a bookmark is a video
- Force youtube videos to play through the "nocookie" variant
- Escape key on menu-like UI elements
- Reduced title sizes. Thanks @joachim
- More tests
- Container for arm64 architecture

### Fixed
- Removed the "summary" element marker on Safari. Thanks @joachim
- Fixed an extraction bug with code formatting in articles

## [0.10.6] - 2023-12-27
### Improved
- Content Script for BNF websites going through oclc.org. This is mainly for the extension where a user can save content on mediapart and arretsurimage while using the BNF portal.
- When readability is disabled, keep performing the pre and post cleaning process
- Much better Wikipedia extraction. Thanks @joachimesque
- Remove some absolute image sources that were not playing well when behind a reverse proxy. Thanks @JerryWham and @franckpaul
- Added a global READECK_USE_X_FORWARDED environment variable that should be set when running Readeck in a container behind a reverse proxy.

### Fixed
- Presentation bug with very long numbered lists
- Typo in the main menu, by @denis_defreyne
- Fixed a bug preventing loading all the site configuration rules

## [0.10.5] - 2023-12-20
### Improved
- Added more API documentation and some examples

### Fixed
- Fixed API documentation "try" mode not working
- Fixed a bug when creating a bookmark with cached resources

## [0.10.4] - 2023-12-18
### Fixed
- Fixed a bug preventing the execution of the main content script
- Fixed the newyorker.com site configuration

## [0.10.3] - 2023-12-16
### Changed
- Updated site config files
- Updated dependencies

### Fixed
- Fixed a CSS print display bug with QR Codes

## [0.10.2] - 2023-12-10
### Improved
- Bookmark print stylesheet
- Display QR codes in print and ebook
- Better youtube transcript retrieval

## [0.10.1] - 2023-12-09
### Improved
- Compress HTTP responses for some content types

### Fixed
- Fixed metric name for bookmark_resources_total

## [0.10.0] - 2023-12-02
### Added
- Content Script system. This includes the five filters rulesets and improves the previous rules system with a documented and much safer JS API to override predefined rules or create complex workflows
- Video transcripts for youtube, vimeo and TED
- Content script for substack
- Store the video duration when available
- Extract text direction, when available and apply it when showing an article's content
- Added a scroll indicator on articles, by @joachimesque
- Added a quick access menu for keyboard navigation, by @joachimesque
- API documentation (not full yet)
- Prometheus metrics

### Improved
- Shorten the bookmark card's title to avoid disgraceful overflows
- Prefixed every `id` and `name` (on `a` tags) attribute on extracted articles
- Lots of CSS improvement, by @joachimesque
- Improved the collection creation workflow
- Refactored and improved the bookmark storage system (non breaking)

### Fixed
- Better initial content cleanup
- Fixed Safari Mobile layout, by @joachimesque

## [0.9.2] - 2023-09-15
### Added
- New, web based, onboarding process
- Code of Conduct

### Fixed
- Default listening port is now 8000
- EPUB filenames length limit

## [0.9.1] - 2023-09-12
### Changed
- Internals

## [0.9.0] - 2023-09-10
### Changed
- Improved bookmark creation form
- Links are extracted in the background so a bookmark is visible earlier
- Bookmark creation from the API can now receive a label list
- Improved performances of log-in with an application password
- Configuration/env/flags priority for command line
- Bookmark order in collection's ebook

### Fixed
- Send a CSRF secure cookie only if scheme is https
- Search string must keep invalid fields and treat them as quoted values
- Major improvements in Print Stylesheet (by @joachimesque)
- Instagram picture title and description

## [0.8.1] - 2023-08-29
### Fixed
- Initial secret key must not create an unreadable configuration file
- server.allowed_host is not mandatory anymore in configuration
- Set host and port through environment variables

## [0.8.0] - 2023-08-29
### Added
- Scoped API tokens
- Application passwords
- OPDS catalog
- Embeded documentation
- Fetch and display links from a page
- Markdown export (only from API for now)
- Added a time range in bookmark filters and collections
- Global highlight list
- Label deletion

### Changed
- Hardened CSP headers
- New random image on bookmark cards
- Go 1.21 required to build
- Release process (linux, mac, windows, container)

### Fixed
- Reddit picture fetch

## [0.7.3] - 2023-07-16
### Added
- Reader typography options

### Changed
- Stylesheet

## [0.7.2] - 2023-06-27
### Changed
- New layout, with enhanced UX and mobile support

## [0.7.1] - 2023-06-18
### Added
- SVG image support during extraction

## [0.7.0] - 2023-05-24
### Added
- Highlights in reader content

### Changed
- Go 1.20
- Dependencies upgrade (Go and JS)

### Fixed
- Extractor improvements
- Site configuration update

## [0.6.1] - 2022-03-24
### Added
- Label autocomplete on bookmark's label form

### Changed
- Go 1.18
- Dependencies upgrade (Go and JS)

## [0.6.0] - 2022-03-22
### Added
- Bookmark's title can be changed
- Bookmark creation with multipart can receive any resources that could
  be used later by the extraction and archive process.

## [0.5.0] - 2021-10-25
### Added
- Password recovery
- Label list and label management
- Top menu and sidebar
- Epub export of bookmark(s)
- Advanced search
- Collections

### Changed
- Refactored a lot of things
- New forms library
- Go 1.17

### Fixed
- Many things; this release is too big...

## [0.3.1] - 2021-05-02
### Changed
- New design, more mobile friendly
- Session now only uses encrypted cookies

### Fixed
- Limit the document type to predefined values

### Added
- Security features on blocked IPs and loop prevention during extraction

## [0.2.3] - 2021-04-18
### Added
- Rule engine for extraction, in ES5
- Rules for some big websites

### Removed
- Read status on bookmark, only keep "archived".

### Fixed
- Focus bug on label edition.

### Changed
- Assets are now made with gulp and the JS bundle is served as a module.

## [0.2.2] - 2021-04-11
### Added
- Increased remote image size limit to 30Mpx

## [0.2.1] - 2021-04-11
### Added
- Reading time information
- Optional Redis session store
- Bookmark actions at the end of the article

## [0.2.0] - 2021-04-08
### Added
- ACLs using RBAC with roles on users
- Admin section, for user management
- Improved profile section
- Better caching of pages when possible

## [0.1.2] - 2021-03-21
### Fixed
- Never try to download and resize images that are too big (more than 3Mpx)

### Added
- CLI adduser command

## [0.1.0] - 2021-03-21
### Added
- Initial release 🎉
