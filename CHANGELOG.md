# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.2] - 2020-02-28
### Fixed
- replace "address" with "host" in configuration
- change one log level from ERROR to WARN

## [1.1.1] - 2020-01-07
### Added
- CHANGELOG.md file to document any changes.
- WaittingGroup for clean up jobs and Cancel function when stop driver.
- save/load current devices subscription info in time.


### Fixed
- Change "Origin" timestamp unit to micro seconds.
- few times for execute defer functions.
- replace time.After() with time.NewTicker()
- cvs slice reallocate space

### Deprecated
- Error Handler function when data dropped, low I/O effects of log file.
- Logger Info for new data arrived, low I/O.