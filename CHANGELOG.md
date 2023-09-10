# Changlog

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
