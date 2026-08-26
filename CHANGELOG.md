# Changelog

## [1.3.4](https://github.com/avargaskun/filestash/compare/v1.3.3...v1.3.4) (2026-08-26)


### Bug Fixes

* clamp the HLS playlist to the media's real content end ([e0028d8](https://github.com/avargaskun/filestash/commit/e0028d86740105c5d30d003f0d1677d2801f01de))

## [1.3.3](https://github.com/avargaskun/filestash/compare/v1.3.2...v1.3.3) (2026-08-25)


### Bug Fixes

* skip the encoder EOF flush for a segment that never fed the encoder ([e29f705](https://github.com/avargaskun/filestash/commit/e29f70520bffb88d99398958174df82735f5b5d8))

## [1.3.2](https://github.com/avargaskun/filestash/compare/v1.3.1...v1.3.2) (2026-08-25)


### Bug Fixes

* survive and cleanly end HLS segments whose window holds no video frame ([#12](https://github.com/avargaskun/filestash/issues/12)) ([c3241e2](https://github.com/avargaskun/filestash/commit/c3241e2573e31d3d532e3999fbb5d1441b498745))

## [1.3.1](https://github.com/avargaskun/filestash/compare/v1.3.0...v1.3.1) (2026-08-25)


### Bug Fixes

* declare only the codecs a source actually has in the HLS master playlist ([#10](https://github.com/avargaskun/filestash/issues/10)) ([aea0a2e](https://github.com/avargaskun/filestash/commit/aea0a2e541c9b79a8260d586d5da8e1151c42621))

## [1.3.0](https://github.com/avargaskun/filestash/compare/v1.2.0...v1.3.0) (2026-08-25)


### Features

* fullscreen player controls with +/-10s skip and auto-hide ([#8](https://github.com/avargaskun/filestash/issues/8)) ([9d48e43](https://github.com/avargaskun/filestash/commit/9d48e43b77caef10485cb35e77f05cf6682cae66))

## [1.2.0](https://github.com/avargaskun/filestash/compare/v1.1.0...v1.2.0) (2026-08-24)


### Features

* client-selectable transcode quality presets and force-transcode ([#6](https://github.com/avargaskun/filestash/issues/6)) ([5448333](https://github.com/avargaskun/filestash/commit/5448333982d36947bc138809236a16bd73de4c18))

## [1.1.0](https://github.com/avargaskun/filestash/compare/v1.0.0...v1.1.0) (2026-08-24)


### Features

* remove the telemetry uploader ([#5](https://github.com/avargaskun/filestash/issues/5)) ([7ef1b6e](https://github.com/avargaskun/filestash/commit/7ef1b6e2aa8c113d7ef440ee2cb0ff0f3f3c7036))
* stream local files without copying them into the video cache first ([#3](https://github.com/avargaskun/filestash/issues/3)) ([1d686f5](https://github.com/avargaskun/filestash/commit/1d686f509483920b9e353d96439d167a82721c11))

## 1.0.0 (2026-08-24)


### Features

* fork bootstrap — build from the checkout, Intel VAAPI runtime, release-please + GHCR publishing ([#1](https://github.com/avargaskun/filestash/issues/1)) ([1f9605a](https://github.com/avargaskun/filestash/commit/1f9605aeac15982ff6f711e70039541dc13879cf))
