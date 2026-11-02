# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.1.0 (2026-11-02)


### ⚠ BREAKING CHANGES

* eventlog.Open with an empty Config.DatabaseURL now returns an error instead of connecting to a built-in default.

### Bug Fixes

* return an error when DatabaseURL is empty ([#12](https://github.com/manovaspace/orbit-event-log/issues/12)) ([2eb72c1](https://github.com/manovaspace/orbit-event-log/commit/2eb72c17589430bedf8421f181860cfb0af40b1a))


### Code Refactoring

* adopt the Go library standard ([#15](https://github.com/manovaspace/orbit-event-log/issues/15)) ([2898a0b](https://github.com/manovaspace/orbit-event-log/commit/2898a0bc1d3c02decba013e6ca4ba48410472e91))

## [Unreleased]

### Added

- Initial public MIT release.
