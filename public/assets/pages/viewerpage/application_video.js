import { createElement, onDestroy } from "../../lib/skeleton/index.js";
import rxjs, { effect } from "../../lib/rx.js";
import { animate } from "../../lib/animate.js";
import { qs, safe } from "../../lib/dom.js";
import { settings_get, settings_put } from "../../lib/settings.js";
import { ApplicationError } from "../../lib/error.js";
import assert from "../../lib/assert.js";
import Hls from "../../lib/vendor/hlsjs/hls.js";
import { loadCSS } from "../../helpers/loader.js";
import ctrlError from "../ctrl_error.js";

import createSources from "./application_video/sources.js";
import { transition } from "./common.js";
import { formatTimecode } from "./common_player.js";
import { ICON } from "./common_icon.js";
import { renderMenubar, buttonDownload, buttonFullscreen, buttonChromecast, buttonQuality } from "./component_menubar.js";
import ctrlDownloader, { init as initDownloader } from "./application_downloader.js";

import "../../components/icon.js";

const STATUS_PLAYING = "PLAYING";
const STATUS_PAUSED = "PAUSED";
const STATUS_BUFFERING = "BUFFERING";

const AUTOHIDE_DELAY = 3000;
const SKIP_SECONDS = 10;
const ICON_FULLSCREEN_EXIT = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0iI2YyZjJmMiIgZD0iTTUgMTZoM3YzaDJ2LTVINXYyem0zLThINXYyaDVWNUg4djN6bTYgMTFoMnYtM2gzdi0yaC01djV6bTItMTFWNWgtMnY1aDVWOGgtM3oiLz48L3N2Zz4K";
const ICON_SKIP_BACK = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0iIzZmNmY2ZiIgZD0iTTEyIDVWMUw3IDZsNSA1VjdjMy4zMSAwIDYgMi42OSA2IDZzLTIuNjkgNi02IDYtNi0yLjY5LTYtNkg0YzAgNC40MiAzLjU4IDggOCA4czgtMy41OCA4LTgtMy41OC04LTgtOHptLTEuMSAxMUgxMHYtMy4zTDkgMTN2LS43bDEuOC0uNmguMVYxNnptNC4zLTEuOGMwIC4zIDAgLjYtLjEuOGwtLjMuNnMtLjMuMy0uNS4zLS40LjEtLjYuMS0uNCAwLS42LS4xLS4zLS4yLS41LS4zLS4yLS4zLS4zLS42LS4xLS41LS4xLS44di0uN2MwLS4zIDAtLjYuMS0uOGwuMy0uNnMuMy0uMy41LS4zLjQtLjEuNi0uMS40IDAgLjYuMS4zLjIuNS4zLjIuMy4zLjYuMS41LjEuOHYuN3ptLS45LS44di0uNXMtLjEtLjItLjEtLjMtLjEtLjEtLjItLjItLjItLjEtLjMtLjEtLjIgMC0uMy4xbC0uMi4ycy0uMS4yLS4xLjN2MnMuMS4yLjEuMy4xLjEuMi4yLjIuMS4zLjEuMiAwIC4zLS4xbC4yLS4ycy4xLS4yLjEtLjN2LTEuNXoiLz48L3N2Zz4K";
const ICON_SKIP_FORWARD = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0iIzZmNmY2ZiIgZD0iTTE4IDEzYzAgMy4zMS0yLjY5IDYtNiA2cy02LTIuNjktNi02IDIuNjktNiA2LTZ2NGw1LTUtNS01djRjLTQuNDIgMC04IDMuNTgtOCA4czMuNTggOCA4IDggOC0zLjU4IDgtOGgtMnptLTcuNDYgM0g5LjY2di0zLjI4bC0xLjAxLjMxdi0uN2wxLjgtLjY0aC4wOVYxNnptNC4xOC0xLjcxYzAgLjMyLS4wMy42LS4xLjgycy0uMTcuNDItLjI5LjU3LS4yOC4yNi0uNDUuMzMtLjM3LjEtLjU5LjEtLjQxLS4wMy0uNTktLjEtLjMzLS4xOC0uNDYtLjMzLS4yMy0uMzQtLjMtLjU3LS4xMS0uNS0uMTEtLjgydi0uNzRjMC0uMzIuMDMtLjYuMS0uODJzLjE3LS40Mi4yOS0uNTcuMjgtLjI2LjQ1LS4zMy4zNy0uMS41OS0uMS40MS4wMy41OS4xLjMzLjE4LjQ2LjMzLjIzLjM0LjMuNTcuMTEuNS4xMS44MnYuNzR6bS0uODUtLjg2YzAtLjE5LS4wMS0uMzUtLjA0LS40OHMtLjA3LS4yMy0uMTItLjMxLS4xMS0uMTQtLjE5LS4xNy0uMTYtLjA1LS4yNS0uMDUtLjE4LjAyLS4yNS4wNS0uMTQuMDktLjE5LjE3LS4wOS4xOC0uMTIuMzEtLjA0LjI5LS4wNC40OHYuOTdjMCAuMTkuMDEuMzUuMDQuNDhzLjA3LjI0LjEyLjMyLjExLjE0LjE5LjE3LjE2LjA1LjI1LjA1LjE4LS4wMi4yNS0uMDUuMTMtLjA5LjE5LS4xNy4wOS0uMTkuMTEtLjMyLjA0LS4yOS4wNC0uNDh2LS45N3oiLz48L3N2Zz4K";

export default function(render, { getFilename, getDownloadUrl, acl$, mime }) {
    const $page = createElement(`
        <div class="component_videoplayer">
            <component-menubar filename="${safe(getFilename())}"></component-menubar>
            <div class="video_container">
                <span>
                    <div class="video_screen">
                        <div class="video_wrapper">
                            <video controlslist="noremoteplayback"></video>
                        </div>
                        <div class="loader no-select">
                            <component-icon name="loading"></component-icon>
                        </div>
                        <div class="videoplayer_control no-select hidden">
                            <div class="progress">
                                <div data-bind="progress-buffer">
                                   <div class="progress-buffer" style="left: 0%; width: 0%;"></div>
                                </div>
                                <div class="progress-active" style="width: 0%;">
                                    <div class="thumb"></div>
                                </div>
                                <div class="progress-placeholder"></div>
                            </div>
                            <img class="component_icon skip_button" draggable="false" src="${ICON_SKIP_BACK}" alt="skip_backward" role="button" tabindex="0" aria-label="skip back 10 seconds">
                            <img class="component_icon" draggable="false" src="${ICON.PLAY}" alt="play">
                            <img class="component_icon hidden" draggable="false" src="${ICON.PAUSE}" alt="pause">
                            <component-icon name="loading" class="hidden"></component-icon>
                            <img class="component_icon skip_button" draggable="false" src="${ICON_SKIP_FORWARD}" alt="skip_forward" role="button" tabindex="0" aria-label="skip forward 10 seconds">

                            <img class="component_icon hidden" draggable="false" src="${ICON.VOLUME_MUTE}" alt="volume_mute">
                            <img class="component_icon hidden" draggable="false" src="${ICON.VOLUME_LOW}" alt="volume_low">
                            <img class="component_icon hidden" draggable="false" src="${ICON.VOLUME_NORMAL}" alt="volume">
                            <input type="range" min="0" max="100" value="13">
                            <span class="timecode">
                                <div class="current"></div>
                                <div class="hint hidden"></div>
                            </span>
                        </div>
                    </div>
                </span>
            </div>
        </div>
    `);
    render($page);
    transition(qs($page, ".video_container"));

    const init$ = new rxjs.Subject();
    const $video = qs($page, "video");
    const $loader = qs($page, ".loader");
    const $control = {
        main: qs($page, ".videoplayer_control"),
        play: qs($page, `.videoplayer_control [alt="play"]`),
        pause: qs($page, `.videoplayer_control [alt="pause"]`),
        loading: qs($page, `.videoplayer_control component-icon[name="loading"]`),
        skip_backward: qs($page, `.videoplayer_control [alt="skip_backward"]`),
        skip_forward: qs($page, `.videoplayer_control [alt="skip_forward"]`),
    };
    const $volume = {
        range: qs($page, `input[type="range"]`),
        icon_mute: qs($page, `img[alt="volume_mute"]`),
        icon_low: qs($page, `img[alt="volume_low"]`),
        icon_normal: qs($page, `img[alt="volume"]`),
    };
    const $hint = qs($page, ".hint");
    const $progress = qs($page, ".progress");

    // quality selection: presets force a server-side transcode regardless of
    // what the browser could play natively; "original" direct-streams the file.
    const sourceSpec = createSources(mime, getDownloadUrl());
    const qualities = sourceSpec.transcodable
        ? [...sourceSpec.presets, "original"]
        : ["original"];
    const pickQuality = () => {
        if (!sourceSpec.transcodable) return "original";
        const persisted = settings_get("video_quality");
        if (persisted === "original") return "original";
        if (persisted && sourceSpec.presets.includes(persisted)) return persisted;
        if ($video.canPlayType(mime) && !sourceSpec.forceTranscodeDefault) return "original";
        return sourceSpec.defaultPreset;
    };
    const initialQuality = pickQuality();
    const quality$ = new rxjs.Subject();

    let hls = null;
    const teardown = () => {
        if (hls) { hls.destroy(); hls = null; }
        while ($video.firstChild) $video.removeChild($video.firstChild);
    };
    // attach the sources for the requested quality and resolve on loadeddata,
    // mirroring the original native/MSE/downloader fallback ladder.
    const attach = (quality) => {
        teardown();
        const list = quality === "original"
            ? sourceSpec.original
            : [["application/x-mpegURL", sourceSpec.hls(quality)]];
        const $sources = [];
        for (const [type, src] of list) {
            const $source = document.createElement("source");
            $source.setAttribute("type", type);
            $source.setAttribute("src", src);
            if ($video.canPlayType(type)) $sources.push($source);
        }
        if ($sources.length > 0) {
            $video.append(...$sources);
            $video.load();
            return rxjs.merge(
                rxjs.fromEvent($video, "loadeddata"),
                ...$sources.map(($source) => rxjs.fromEvent($source, "error").pipe(rxjs.mergeMap(() =>
                    rxjs.throwError(() => new ApplicationError("Not Supported", JSON.stringify({ type: $source.type, src: $source.src }, null, 2))),
                ))),
            );
        }
        if (Hls.isSupported()) {
            for (const [type, src] of list) {
                if (type !== "application/x-mpegURL") continue;
                hls = new Hls();
                hls.loadSource(src);
                hls.attachMedia($video);
                return rxjs.fromEvent($video, "loadeddata");
            }
        }
        return rxjs.from(initDownloader()).pipe(rxjs.mergeMap(() => {
            ctrlDownloader(render, { acl$, getFilename, getDownloadUrl });
            return rxjs.EMPTY;
        }));
    };

    // fullscreen: the target is the whole player page so the overlay controls and the
    // quality selector come along; the same button doubles as the exit affordance.
    const fullscreenElement = () => document.fullscreenElement ||
        document.webkitFullscreenElement || document.mozFullScreenElement || null;
    const isFullscreen = () => fullscreenElement() === $page;
    const canFullscreen = !!($page.requestFullscreen || $page.webkitRequestFullscreen || $page.mozRequestFullScreen);
    const requestFullscreen = () => {
        if ($page.requestFullscreen) return $page.requestFullscreen();
        else if ($page.webkitRequestFullscreen) return $page.webkitRequestFullscreen();
        else if ($page.mozRequestFullScreen) return $page.mozRequestFullScreen();
        return null;
    };
    const exitFullscreen = () => {
        if (document.exitFullscreen) return document.exitFullscreen();
        else if (document.webkitExitFullscreen) return document.webkitExitFullscreen();
        else if (document.mozCancelFullScreen) return document.mozCancelFullScreen();
        return null;
    };
    // an unhandled rejection here would reach window.onerror and replace the whole page
    // with the error screen, so every fullscreen/orientation promise is swallowed.
    const swallow = (maybePromise) => {
        if (maybePromise && typeof maybePromise.catch === "function") maybePromise.catch(() => {});
    };
    const toggleFullscreen = () => {
        if (fullscreenElement()) swallow(exitFullscreen());
        else swallow(requestFullscreen());
    };
    const lockOrientation = () => {
        try {
            if (screen.orientation && typeof screen.orientation.lock === "function") swallow(screen.orientation.lock("landscape"));
        } catch (err) { /* not supported on this engine */ }
    };
    const unlockOrientation = () => {
        try {
            if (screen.orientation && typeof screen.orientation.unlock === "function") screen.orientation.unlock();
        } catch (err) { /* not supported on this engine */ }
    };

    const $fullscreen = buttonFullscreen($page, canFullscreen ? toggleFullscreen : undefined);
    const $fullscreenIcon = $fullscreen.querySelector ? $fullscreen.querySelector("img") : null;
    const ICON_FULLSCREEN_ENTER = $fullscreenIcon ? $fullscreenIcon.getAttribute("src") : null;
    renderMenubar(
        qs($page, "component-menubar"),
        buttonDownload(getDownloadUrl()),
        buttonQuality(qualities, initialQuality, (q) => quality$.next(q)),
        $fullscreen,
        buttonChromecast($video),
    );

    const setVolume = (volume) => {
        settings_put("volume", volume);
        $video.volume = volume / 100;
        $volume.range.value = volume;
        if (volume === 0) {
            $volume.icon_mute.classList.remove("hidden");
            $volume.icon_low.classList.add("hidden");
            $volume.icon_normal.classList.add("hidden");
        } else if (volume < 50) {
            $volume.icon_mute.classList.add("hidden");
            $volume.icon_low.classList.remove("hidden");
            $volume.icon_normal.classList.add("hidden");
        } else {
            $volume.icon_mute.classList.add("hidden");
            $volume.icon_low.classList.add("hidden");
            $volume.icon_normal.classList.remove("hidden");
        }
    };
    // auto-hide: only armed while fullscreen AND playing. `playerStatus` exists because
    // setStatus keeps no readable state of its own.
    let playerStatus = null;
    let hideTimer = null;
    let hiddenAtPointerDown = false;
    const controlsHidden = () => $page.classList.contains("controls-hidden");
    const stopHideTimer = () => {
        if (hideTimer === null) return;
        clearTimeout(hideTimer);
        hideTimer = null;
    };
    const startHideTimer = () => {
        stopHideTimer();
        if (!isFullscreen() || playerStatus !== STATUS_PLAYING) return;
        hideTimer = setTimeout(() => {
            hideTimer = null;
            if (isFullscreen() && playerStatus === STATUS_PLAYING) $page.classList.add("controls-hidden");
        }, AUTOHIDE_DELAY);
    };
    const revealControls = () => {
        $page.classList.remove("controls-hidden");
        startHideTimer();
    };
    onDestroy(() => {
        stopHideTimer();
        unlockOrientation();
    });

    const setStatus = (status) => {
        playerStatus = status;
        switch (status) {
        case "PLAYING":
            $control.play.classList.add("hidden");
            $control.pause.classList.remove("hidden");
            $control.loading.classList.add("hidden");
            $video.play();
            break;
        case "PAUSED":
            $control.play.classList.remove("hidden");
            $control.pause.classList.add("hidden");
            $control.loading.classList.add("hidden");
            $video.pause();
            break;
        case "BUFFERING":
            $control.play.classList.add("hidden");
            $control.pause.classList.add("hidden");
            $control.loading.classList.remove("hidden");
            break;
        default:
            assert.fail(status);
        }
        $loader.classList.add("hidden");
        $control.main.classList.remove("hidden");
        $control.main.classList.add("visible");
        switch (status) {
        case STATUS_PLAYING:
            startHideTimer();
            break;
        case STATUS_PAUSED:
            stopHideTimer();
            $page.classList.remove("controls-hidden");
            break;
        case STATUS_BUFFERING:
            // suspend the timer but keep the current visibility: every HLS skip and
            // rebuffer emits `waiting`, and revealing here would flash the bar.
            stopHideTimer();
            break;
        }
    };
    const setSeek = (newTime, shouldSet = false) => {
        if (shouldSet) {
            // assigning a non-finite value to currentTime throws (WebIDL restricted
            // double) and effect() rethrows into window.onerror = the app error screen.
            if (!isFinite(newTime)) return;
            $video.currentTime = newTime;
        }
        const width = 100 * (newTime / $video.duration);
        qs($page, ".progress .progress-active").style.width = `${width}%`;
        if (!isNaN($video.duration)) {
            qs($page, ".timecode .current").textContent = formatTimecode($video.currentTime) + " / " + formatTimecode($video.duration);
        }
    };

    // feature1: setup the dom
    effect(attach(initialQuality).pipe(
        rxjs.mergeMap(() => {
            $loader.replaceChildren(createElement(`<img src="${ICON.PLAY}" />`));
            animate($loader, {
                time: 150,
                keyframes: [
                    { transform: "scale(0.7)" },
                    { transform: "scale(1)" },
                ],
            });
            setSeek(0);
            return rxjs.race(
                rxjs.fromEvent($loader, "click").pipe(rxjs.mapTo($loader)),
                rxjs.fromEvent(document, "keydown").pipe(rxjs.filter((e) => e.code === "Space"), rxjs.first()),
            );
        }),
        rxjs.tap(() => setStatus(STATUS_PLAYING)),
        rxjs.catchError(ctrlError()),
        rxjs.tap(() => init$.next()),
    ));

    // feature1b: quality switching - reload the source, keep position + play state
    effect(quality$.pipe(
        rxjs.skipUntil(init$),
        rxjs.tap((q) => settings_put("video_quality", q)),
        rxjs.mergeMap((q) => {
            const resumeTime = $video.currentTime;
            const wasPlaying = !$video.paused;
            setStatus(STATUS_BUFFERING);
            return attach(q).pipe(
                rxjs.tap(() => {
                    if (!isNaN(resumeTime) && resumeTime > 0) $video.currentTime = resumeTime;
                    setStatus(wasPlaying ? STATUS_PLAYING : STATUS_PAUSED);
                }),
                rxjs.catchError(ctrlError()),
            );
        }),
    ));

    // feature2: player control - volume
    effect(rxjs.combineLatest(
        rxjs.fromEvent($volume.range, "input").pipe(
            rxjs.map((e) => parseInt(e.target.value)),
            rxjs.startWith(settings_get("volume") === null ? 80 : settings_get("volume")),
        ),
        rxjs.merge(
            rxjs.fromEvent($volume.icon_mute, "click"),
            rxjs.fromEvent($volume.icon_low, "click"),
            rxjs.fromEvent($volume.icon_normal, "click"),
        ).pipe(
            rxjs.startWith(false),
            rxjs.scan((isMuted) => !isMuted, true),
        ),
    ).pipe(rxjs.tap(([volume, isMuted]) => {
        if (isMuted) setVolume(0);
        else setVolume(volume);
    })));

    // feature3: player control - play/pause
    effect(rxjs.merge(
        rxjs.fromEvent($control.play, "click").pipe(rxjs.mapTo(STATUS_PLAYING)),
        rxjs.fromEvent($control.pause, "click").pipe(rxjs.mapTo(STATUS_PAUSED)),
        rxjs.fromEvent($video, "ended").pipe(rxjs.mapTo(STATUS_PAUSED)),
        rxjs.fromEvent($video, "waiting").pipe(rxjs.mapTo(STATUS_BUFFERING)),
        rxjs.fromEvent($video, "playing").pipe(rxjs.mapTo(STATUS_PLAYING)),
    ).pipe(
        rxjs.skipUntil(init$),
        rxjs.debounceTime(50),
        rxjs.tap(setStatus),
    ));

    // feature3b: player control - skip +/- 10s. Its own stream on purpose: feature3's
    // debounceTime(50) would collapse two rapid taps into a single skip.
    effect(rxjs.merge(
        rxjs.fromEvent($control.skip_backward, "click").pipe(rxjs.mapTo(-SKIP_SECONDS)),
        rxjs.fromEvent($control.skip_forward, "click").pipe(rxjs.mapTo(SKIP_SECONDS)),
    ).pipe(
        rxjs.skipUntil(init$),
        rxjs.tap((delta) => {
            // the source is torn down and reloaded on every quality switch, so duration
            // is NaN for the whole switch - a skip tap then must be a silent no-op.
            if (!isFinite($video.duration) || !isFinite($video.currentTime)) return;
            setSeek(Math.max(0, Math.min($video.duration, $video.currentTime + delta)), true);
        }),
    ));

    // feature3c: fullscreen state sync - Esc, the android back gesture and the toggle
    // button all land here, so the class and the icon can never desync.
    effect(rxjs.merge(
        rxjs.fromEvent(document, "fullscreenchange"),
        rxjs.fromEvent(document, "webkitfullscreenchange"),
        rxjs.fromEvent(document, "mozfullscreenchange"),
    ).pipe(rxjs.tap(() => {
        const on = isFullscreen();
        $page.classList.toggle("is-fullscreen", on);
        if ($fullscreenIcon) $fullscreenIcon.setAttribute("src", on ? ICON_FULLSCREEN_EXIT : ICON_FULLSCREEN_ENTER);
        if ($fullscreen.setAttribute) $fullscreen.setAttribute("aria-label", on ? "exit fullscreen" : "fullscreen");
        if (on) lockOrientation();
        else unlockOrientation();
        $page.classList.remove("controls-hidden");
        startHideTimer();
    })));

    // feature3d: auto-hide reveal. The latch is read by feature8 so that the tap which
    // reveals the controls does not also toggle play/pause (pointerdown precedes click).
    effect(rxjs.merge(
        rxjs.fromEvent($page, "pointerdown"),
        rxjs.fromEvent($page, "mousemove"),
        rxjs.fromEvent(document, "keydown"),
    ).pipe(rxjs.tap((e) => {
        if (e.type === "pointerdown") hiddenAtPointerDown = isFullscreen() && controlsHidden();
        if (!isFullscreen()) return;
        revealControls();
    })));

    // feature4: player control - seek
    effect(rxjs.fromEvent($progress, "click").pipe(
        rxjs.skipUntil(init$),
        rxjs.map((e) => {
            let $progress = e.target;
            if (e.target.classList.contains("progress") === false) {
                $progress = e.target.parentElement;
            }
            const rec = $progress.getBoundingClientRect();
            return (e.clientX - rec.x) / rec.width;
        }),
        rxjs.tap((n) => {
            if (n < 2/100) {
                setStatus(STATUS_PAUSED);
                n = 0;
            }
            setSeek(n * $video.duration, true);
        }),
    ));

    // feature5: render the progress bar
    effect(rxjs.fromEvent($video, "timeupdate").pipe(
        rxjs.skipUntil(init$),
        rxjs.tap(() => setSeek($video.currentTime)),
    ));

    // feature6: render loading buffer
    effect(rxjs.fromEvent($video, "timeupdate").pipe(
        rxjs.skipUntil(init$),
        rxjs.tap(() => {
            const $container = qs($page, `[data-bind="progress-buffer"]`);
            if ($video.buffered.length !== $container.children.length) {
                $container.innerHTML = "";
                for (let i = 0; i < $video.buffered.length; i++) $container.appendChild(createElement(`
                    <div class="progress-buffer"></div>
                `));
            }
            for (let i=0; i<$video.buffered.length; i++) {
                const width = ($video.buffered.end(i) - $video.buffered.start(i)) / $video.duration * 100;
                const left = $video.buffered.start(i) / $video.duration * 100;
                $container.children[i].style.left = left + "%";
                $container.children[i].style.width = width + "%";
            }
        }),
    ));

    // feature7: hint
    effect(rxjs.merge(
        rxjs.fromEvent($progress, "mousemove"),
        rxjs.fromEvent($progress, "mouseleave"),
    ).pipe(
        rxjs.skipUntil(init$),
        rxjs.map((e) => ({
            type: e.type,
            clientX: e.clientX,
            clientWidth: e.target.clientWidth,
            rec: e.target.getBoundingClientRect(),
            duration: $video.duration,
        })),
        rxjs.map(({ type, clientX, clientWidth, rec, duration }) => {
            switch (type) {
            case "mouseleave":
                return { visible: false };
            case "mousemove":
                const width = clientX - rec.x;
                const time = duration * width / rec.width;
                let posX = width;
                posX = Math.max(posX, 30);
                posX = Math.min(posX, clientWidth - 30);
                return { x: `${posX}px`, time, visible: true };
            default:
                assert.fail(`unexpected event: ${type}`);
            }
            return null;
        }),
        rxjs.tap(({ visible, x, time }) => {
            if (!visible) return $hint.classList.add("hidden");
            $hint.classList.remove("hidden");
            $hint.style.left = x;
            $hint.textContent = formatTimecode(time);
        }),
    ));

    // feature8: player control - keyboard shortcut
    effect(rxjs.merge(
        rxjs.fromEvent(document, "keydown").pipe(rxjs.map((e) => e.code)),
        rxjs.fromEvent($video, "click").pipe(rxjs.filter(() => !hiddenAtPointerDown), rxjs.mapTo("Space")),
    ).pipe(
        rxjs.skipUntil(init$),
        rxjs.tap((code) => {
            switch (code) {
            case "Space":
            case "KeyK":
                setStatus($video.paused ? STATUS_PLAYING : STATUS_PAUSED);
                break;
            case "KeyM":
                setVolume($video.volume > 0 ? 0 : settings_get("volume"));
                break;
            case "ArrowUp":
                setVolume(Math.min($video.volume*100 + 10, 100));
                break;
            case "ArrowDown":
                setVolume(Math.max($video.volume*100 - 10, 0));
                break;
            case "KeyL":
                setSeek(Math.min($video.duration, $video.currentTime + 10), true);
                break;
            case "KeyJ":
                setSeek(Math.max(0, $video.currentTime - 10), true);
                break;
            case "Home":
            case "Digit0":
                setSeek(0, true);
                break;
            case "Digit1":
                setSeek($video.duration / 10, true);
                break;
            case "Digit2":
                setSeek($video.duration * 2 / 10, true);
                break;
            case "Digit3":
                setSeek($video.duration * 3 / 10, true);
                break;
            case "Digit4":
                setSeek($video.duration * 4 / 10, true);
                break;
            case "Digit5":
                setSeek($video.duration * 5 / 10, true);
                break;
            case "Digit6":
                setSeek($video.duration * 6 / 10, true);
                break;
            case "Digit7":
                setSeek($video.duration * 7 / 10, true);
                break;
            case "Digit8":
                setSeek($video.duration * 8 / 10, true);
                break;
            case "Digit9":
                setSeek($video.duration * 9 / 10, true);
                break;
            case "End":
                setSeek($video.duration, true);
                break;
            }
        }),
    ));
}

export function init() {
    return Promise.all([
        loadCSS(import.meta.url, "./application_video.css"),
        loadCSS(import.meta.url, "./component_menubar.css"),
    ]);
}
