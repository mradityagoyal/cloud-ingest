# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A major version bump indicates support for a major job run version was dropped.
A minor version bump means support for a new major job run version was added.
A patch version bump is used for any change that does not affect the supported range of
major job run versions.

## [Unreleased]

## [1.0.0] - 2019-01-20
### Added
- Support for job run version 1.0.0, which includes Depth-First Listing.
- Support for job run version 2.0.0, which includes file bundling.

## [0.5.8] - 2019-01-09
### Changed
- Setting goog-reserved-file-mtime field back to Unix time from UnixNano, for
  gsutil compatibility.

## [0.5.7] - 2018-12-27
### Added
- WorkHandler for depth-first listing.
- Internal-testing flag.

## [0.0.0] - 2018-12-12
### Added
- This changelog file.
