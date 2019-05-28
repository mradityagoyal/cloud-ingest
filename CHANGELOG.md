# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A major version bump indicates support for a major job run version was dropped.
A minor version bump means support for a new major job run version was added.
A patch version bump is used for any change that does not affect the supported range of
major job run versions.

## [Unreleased]

## [1.0.9] - 2019-05-24
### Changed
- Make the incremental copy chunk size flag controlled, default to 128MiB.
  Increase the default PubSub message timeout to 20m.
- Flag control which files use the resumable vs veneer copy paths.

## [1.0.8] - 2019-05-17
### Added
- AgentUpdate to control message
### Changed
- Memory usage improvements for file copies.
- The 'copy memory limiter' is removed, which likely improves Agent performance.
- Reduced the number of copy handler retries.

## [1.0.7] - 2019-04-18
### Changed
- Clean up the control subscription upon exit.
- Exit if project id isn't set.

## [1.0.6] - 2019-04-10
### Added
- Count for unlisted directories to list log.
- Retries for copy handling.
### Changed
- Tune the throughput smoothing to use no averaging. The displayed will be much
  more responsive.
- No functional change. Refactor all of the functionality out of /helpers into
  appropriate locations.

## [1.0.5] - 2019-04-03
### Fixed
- Adding calls to close writer in error cases
- Waiting for the control topic before trying to subscribe to it.
- Temporarily in-place fixing the Pub/Sub bug in issue https://github.com/googleapis/google-cloud-go/issues/1379
### Changed
- Remove 'frequency' from the pulse messages. It was never used.

## [1.0.4] - 2019-03-28
### Changed
- Refactor the Agent go files into packages.
- No functional change. Rename 'workprocessor' to 'taskprocessor'.
- Move PubSub settings into the pubsub package.
- No functional change. Move flags and TaskProcessor creation out of agent.go, into the relevant packages.

## [1.0.3] - 2019-03-22
### Changed
- The Agent's "stats" log line format.
- Send 'transferred bytes' in the Pulse message.
- Migrate PubSub functionality out of the main binary (agent.go) into its own package. Add a SIG handler so ctrl-c exits gracefully.
### Added
- Tool for extracting and parsing stats from the Agent's log.

## [1.0.2] - 2019-03-06
### Changed
- Minor refactors/cleanup.
- Make the log lines for copyBundle errors human readable.

## [1.0.1] - 2019-02-14
### Changed
- Increased the list file size threshold.
- UserAgent update.

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
