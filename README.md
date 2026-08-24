# Fork notice

This is **`avargaskun/filestash`**, a fork of
[`mickael-kerjean/filestash`](https://github.com/mickael-kerjean/filestash) maintained for a
single private deployment. Upstream's own README follows below.

## Why the fork exists

Upstream is rolling-release: its git tags stopped in 2019 and the official image
(`machines/filestash`) is effectively `latest`-only, built from master on a private Jenkins.
That is invisible to a semver-based update watcher, and it leaves no place to carry local
patches. This fork exists to publish **semver-tagged images** and to carry a small set of
patches on top of upstream master.

## Images

Published to **`ghcr.io/avargaskun/filestash`** (`linux/amd64`) on every release, tagged
`X.Y.Z`, `X.Y`, `X` and `latest`. Releases are cut by
[release-please](https://github.com/googleapis/release-please) from conventional-commit PR
titles (`fix:` → patch, `feat:` → minor, `feat!:`/`BREAKING CHANGE` → major). The publish job
is chained inside `.github/workflows/release-please.yml` on purpose: a tag created with
`GITHUB_TOKEN` never triggers a second workflow. `workflow_dispatch` on that workflow takes a
`tag` input for a manual re-publish.

## Fork patches

| Area | Change | Landed |
|---|---|---|
| `docker/Dockerfile` | Build from the fork checkout (upstream cloned `mickael-kerjean/filestash` at build time, so a fork could never build its own code) | 1.0.0 |
| `docker/Dockerfile` | Ship the Intel VAAPI runtime — enable Debian's non-free component and install `intel-media-va-driver-non-free` + `vainfo` on amd64, so the transcoder's `h264_vaapi` encoder actually works | 1.0.0 |
| `.github/workflows/` | release-please + GHCR publishing + a PR build/smoke CI job | 1.0.0 |
| `plg_video_transcoder`, `plg_backend_local` | Stream local files without copying them into the video cache first. Upstream `io.Copy`s the whole source file before it will serve the master playlist, so a 40 GB remux costs a full disk-to-disk copy before the first HLS segment can even be requested. Backends may now expose `LocalPath()`; when the session's backend does, the transcoder points the segment encoders at the real file. Other backends keep the copy path | 1.1.0 |
| `plg_video_transcoder/mediaref` | Resolve the HLS `?path=` parameter through a server-side key→path registry instead of joining it onto the cache directory. The HLS playlist and segment routes have no session middleware, so upstream's `filepath.Join(cacheDir, userInput)` was an unauthenticated arbitrary-file read (`?path=../../../../etc/...`). Regression tests: `server/plugin/plg_video_transcoder/mediaref/mediaref_test.go`, run in CI | 1.1.0 |
| `pkg/middleware`, `pkg/config` | Remove the telemetry uploader. Upstream buffers every request — full URI (i.e. the file paths of the library), client IP, user agent, referer, backend, share id, salted session hash, license — and POSTs batches to `downloads.filestash.app/event` every 10 s whenever `log.telemetry` is on. It defaults off, but it is one admin-console checkbox away and was default-**on** earlier in the project's history. The uploader, its flush goroutine and the `log.telemetry` config knob are all gone, and `LogEntry` is trimmed to what the local access log actually prints, so there is nothing left to switch on. Verify on any image with `grep -a downloads.filestash.app /app/filestash` — no match | 1.1.0 |
| `plg_video_transcoder` (`preset` pkg, `config`, `index.js`, both encode paths), `application_video.js`, `component_menubar.js` | Client-controllable quality + force-transcode. A new `&preset=720p\|480p\|360p` HLS parameter is whitelist-mapped **server-side** to a height + VAAPI/x264 rate control (720p→4 Mbps, 480p→1.5 Mbps, 360p→800 kbps) in both the ffmpeg-exec and libav/cgo paths — upstream's VAAPI had **no** rate control, so its bitrate was driver-default and unbounded. The name never reaches an ffmpeg argument (the fixed `preset` package resolves it to compiled-in numbers; a bogus value is a 400 before any transcode). The player gets a quality menu (the presets + **Original (direct)**): picking a preset force-transcodes regardless of `canPlayType`, Original direct-streams the raw file; the choice persists via `settings_put`. Config knobs `features.video.default_preset` and `features.video.force_transcode_default` (default **true** — transcode by default everywhere, Original one click away). Whitelist tests + a handler-validation guard: `server/plugin/plg_video_transcoder/preset/preset_test.go`, run in CI | 1.2.0 |

## Upstream-sync policy

- **`master` tracks upstream `master`.** Upstream is a moving rolling release with no tags to
  sync against, so there is no automated sync — syncing is **manual and deliberate**.
- **Cadence:** on demand — when an upstream change is wanted, or when a CVE/bug affects us.
  There is no scheduled merge; an unattended fast-forward of a rolling upstream is exactly the
  risk this fork exists to control.
- **Mechanism:** `git fetch upstream && git merge upstream/master` on a branch, resolve, open a
  PR with a conventional-commit title (`feat:` for a feature-bearing sync, `fix:` otherwise) so
  release-please cuts a version for it. Never rebase master — published tags point into it.
- **Every sync is a re-audit event.** Re-verify, at minimum: the telemetry removal still holds
  (`grep -a downloads.filestash.app` on the built binary must find nothing — a merge can quietly
  restore the uploader); no new outbound endpoint appeared in the Go backend or the first-party
  frontend JS; the fork patches above still apply and are still needed.
- Local remotes: `origin` = this fork, `upstream` =
  `https://github.com/mickael-kerjean/filestash`.

---

![screenshot](https://raw.githubusercontent.com/mickael-kerjean/filestash_images/master/.assets/photo.jpg)

# What is this?

<p>
    It started as a storage agnostic Dropbox-like file manager that works with every storage protocol: <a href="https://www.filestash.app/ftp-client.html">FTP</a>, <a href="https://www.filestash.app/ssh-file-transfer.html">SFTP</a>, <a href="https://www.filestash.app/s3-browser.html">S3</a>, <a href="https://www.filestash.app/smb-client.html">SMB</a>, <a href="https://www.filestash.app/webdav-client.html">WebDAV</a>, IPFS, and <a href="https://www.filestash.app/docs/plugin/#storage">about 20 more</a>.
</p>
<p>
    It grew into what we want to be the world's best file management platform. Around the core engine sit 3 pillars: the web client, a <a href="https://github.com/mickael-kerjean/fdrive">native drive client</a>, and <a href="https://www.filestash.app/docs/guide/storage-gateway.html">gateways</a> to expose storages over any protocol.
</p>
<p>
    The engine follows one rule: everything that's not a fundamental truth of the universe lives in a plugin. Where other platforms are take-it-or-leave-it, ours gives you a rock solid core and a plugin system to handle opinions, so however deep requirements go, the only limit won't be technical but your own creativity.
</p>

<p>
    <a href="http://demo.filestash.app"><img src="https://www.filestash.app/img/illustration/filestash-integrations.png" alt="storage + auth architecture" /></a>
</p>

# Key Features

<ul>
    <li><a href="#vision--philosophy">Plugin Driven Architecture</a>: everything that matters is a plugin, browse the <a href="https://www.filestash.app/docs/plugin/">ecosystem</a> or <a href="https://www.filestash.app/docs/guide/plugin-development.html?origin=github">build your own</a>. With this approach, you get exactly what you need without overhead and bloat.</li>
    <li>Universal Access: the web client is just one way to access your data (albeit an awesome one, handcrafted in vanilla JS). <a href="https://www.filestash.app/docs/api/#api">APIs</a> and <a href="https://www.filestash.app/docs/guide/storage-gateway.html?origin=github">Gateways</a> let you also expose your data over protocols like <a href="https://www.filestash.app/docs/guide/sftp-gateway.html?origin=github">SFTP</a>, S3, FTP, WebDAV, <a href="https://www.filestash.app/docs/guide/mcp-gateway.html?origin=github">MCP</a>, and AS2.</li>
    <li><a href="https://www.filestash.app/docs/plugin/#storage">Integrations</a>: our explicit goal is to support 100% of storage and authentication technologies on the market. Beyond your usual options, you can go much further, like a <a href="https://www.filestash.app/docs/guide/virtual-filesystem.html?origin=github">virtual filesystem</a> delegating authentication to your <a href="https://github.com/mickael-kerjean/filestash/tree/master/server/plugin/plg_authenticate_wordpress">WordPress site</a> and using its roles to drive <a href="https://www.filestash.app/docs/guide/authorization.html#option-2-rbac">RBAC authorization</a>.</li>
    <li><a href="https://www.filestash.app/docs/guide/workflow-engine.html">Workflow Engine</a>: automate anything that happens to your files by chaining actions on events, from simple notifications via Slack or email to full on MFT pipelines and everything in between.</li>
    <li>File Apps: use any of the existing apps or <a href="https://www.filestash.app/docs/guide/plugin-development.html#xdg-open-plugins-in-depth">build your own</a>, from astronomy to embroidery and everything in between like:
        <ul>
            <li><a href="https://demo.filestash.app/assets/plugin/application_photography.zip">photography</a>: heif, nef, raf, <a href="https://www.filestash.app/tools/tiff-viewer.html">tiff</a>, raw, arw, sr2, srf, nrw, cr2, crw, x3f, pef, rw2, orf, mrw, mdc, mef, mos, dcr, kdc, 3fr, erf and srw</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_photography.zip">astronomy</a>: <a href="https://www.filestash.app/tools/fits-viewer.html">fits</a>, <a href="https://www.filestash.app/tools/xisf-viewer.html">xisf</a></li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_science.zip">science</a>: with latex, plantuml & pandoc compilers</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_musician.zip">music</a>: mid, midi, gp4 and gp5</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_gis.zip">GIS</a>: <a href="https://www.filestash.app/tools/geojson-viewer.html">geojson</a>, <a href="https://www.filestash.app/tools/shp-viewer.html">shp</a>, gpx, wms and <a href="https://www.filestash.app/tools/dbf-viewer.html">dbf</a></li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_engineering.zip">data engineering</a>: <a href="https://www.filestash.app/tools/parquet-viewer.html">parquet</a>, <a href="https://www.filestash.app/tools/arrow-viewer.html">arrow</a>, <a href="https://www.filestash.app/tools/feather-viewer.html">feather</a>, <a href="https://www.filestash.app/tools/avro-viewer.html">avro</a>, <a href="https://www.filestash.app/tools/orc-viewer.html">orc</a>, <a href="https://www.filestash.app/tools/hdf5-viewer.html">hdf5</a>, <a href="https://www.filestash.app/tools/hdf5-viewer.html">h5</a>, <a href="https://www.filestash.app/tools/netcdf-viewer.html">netcdf</a>, <a href="https://www.filestash.app/tools/netcdf-viewer.html">nc</a>, rds, rda and rdata</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_dev.zip">dev</a>: a, so, o, dylib, dll, tar, tgz, zip, har, cap, pcap, pcapng and <a href="https://www.filestash.app/tools/sqlite-viewer.html">sqlite</a></li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_creative.zip">creative work</a>: svg, <a href="https://www.filestash.app/tools/psd-viewer.html">psd</a>, ai, <a href="https://www.filestash.app/tools/sketch-viewer.html">sketch</a>, <a href="https://www.filestash.app/tools/cdr-viewer.html">cdr</a>, woff, woff2, ttf, otf, eot, exr, tga, pgm, ppm, dds, ktx, dpx, pcx, xpm, pnm, xbm, aai, xwd, cin, pbm, pcd, sgi, wbmp and rgb</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_biomed.zip">biomedical</a>: dicom, sam, bam, cif, pdb, xyz, sdf, mol, mol2 and mmtf</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_autodesk.zip">autodesk</a>: <a href="https://www.filestash.app/tools/dwg-viewer.html">dwg</a> and <a href="https://www.filestash.app/tools/dxf-viewer.html">dxf</a></li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_adobe.zip">adobe</a>: <a href="https://www.filestash.app/tools/psd-viewer.html">psd</a>, ai, <a href="https://www.filestash.app/tools/xd-viewer.html">xd</a>, <a href="https://www.filestash.app/tools/dng-viewer.html">dng</a>, <a href="https://www.filestash.app/tools/eps-viewer.html">postscript</a>, aco, ase, swf</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_3d.zip">3d</a>: fbx, gltf, obj, stl, step, mesh, ifc, dae</li>
            <li><a href="https://demo.filestash.app/assets/plugin/application_embroidery.zip">embroidery</a>: dgt, dst, dsb, dsz, edr, exp, 10o, col, hus, inf, jef, ksm, pcm, pcs, pes, sew, shv, sst, tap, u01, vip, vp3 and xxx</li>
            <li><a href="https://github.com/mickael-kerjean/filestash/tree/master/server/plugin/plg_widget_pgp">e2e</a>: pgp, gpg</li>
        </ul>
    </li>
    <li>Themes: <br>
        <img src="https://www.filestash.app/img/screenshots/theme_github.png" height="150" />
        <img src="https://www.filestash.app/img/screenshots/theme_apple.png" height="150" />
        <img src="https://www.filestash.app/img/screenshots/theme_dropbox.png" height="150" />
        <img src="https://www.filestash.app/img/screenshots/theme_ibm.png" height="150" />
    </li>
    <li>AI features for <a href="https://www.filestash.app/docs/guide/search.html">search</a>, <a href="https://www.filestash.app/features/smart-folder.html">smart folders</a> and OCRs.</li>
    <li>... and much <sub>much <sub>more (versioning, audit, public site, antivirus, quota, chat, chromecast support, on demand video transcoding, mounting shared links as network drive, ....)</sub></sub><br> As a rule of thumb, if your problem involves files, we either already <a href="https://www.filestash.app/docs/plugin/">have a plugin</a> for it or can make a plugin for it
</ul>


# Getting Started

To install Filestash, head to the [Getting started](https://www.filestash.app/docs/?origin=github) guide. If you want to leverage plugins, head over to the [inventory](https://www.filestash.app/docs/plugin/?origin=github), or learn about [developing your own plugins](https://www.filestash.app/docs/guide/plugin-development.html?origin=github).

If you want guidance and expert help on your file management problem, [book a call](https://www.filestash.app/tunnel/demo/?origin=github) and let's figure out if Filestash is the right platform for you.


# Vision & Philosophy

Our goal is simple: **to build the best file management platform ever made. Period.** But "best" means different things to different people, so we made everything pluggable. The core defines interfaces, plugins implement them. Disagree with our implementation? Write your own. Anything that isn't a fundamental truth of the universe and might spark a debate belongs in a plugin. Literally every piece listed in the key features is a plugin you can swap for another implementation or remove entirely.

Say you want to give your users a Dropbox like experience on top of your existing FTP server (remember the [FTP guy during the Dropbox launch on HN](https://news.ycombinator.com/item?id=9224)?). All the [FTP plugin](https://github.com/mickael-kerjean/filestash/tree/master/server/plugin/plg_backend_ftp) does is implement this interface:
```go
type IBackend interface {
	Ls(path string) ([]os.FileInfo, error)           // list files in a folder
	Stat(path string) (os.FileInfo, error)           // file stat
	Cat(path string) (io.ReadCloser, error)          // download a file
	Mkdir(path string) error                         // create a folder
	Rm(path string) error                            // remove something
	Mv(from string, to string) error                 // rename something
	Save(path string, file io.Reader) error          // save a file
	Touch(path string) error                         // create a file

	// I have omitted 2 other methods, a first one to enable connections reuse and
	// another one to declare what should the login form be like.
}
```

There are interfaces you can implement for every key component of Filestash: from storage, to authentication, <a href="https://www.filestash.app/docs/guide/authorization.html">authorisation</a>, custom apps, <a href="https://www.filestash.app/docs/guide/search.html">search</a>, thumbnailing, frontend patches, middleware, endpoint creation and a few others documented in the [plugin development guide](https://www.filestash.app/docs/guide/plugin-development.html).

To see what's currently installed in your instance, head over to [/about](https://demo.filestash.app/about). The inventory of plugins is [documented here](https://www.filestash.app/docs/plugin/)


# Support

- Commercial Users → [support contract](https://www.filestash.app/pricing/?origin=github)
- For individuals:
  - [#filestash](https://kiwiirc.com/nextclient/#irc://irc.libera.chat/#filestash?nick=guest??) on IRC (libera.chat)
  - Bitcoin: `3LX5KGmSmHDj5EuXrmUvcg77EJxCxmdsgW`
  - [Open Collective](https://opencollective.com/filestash)


# Credits

Filestash stands on the shoulder of: [contributors](https://github.com/mickael-kerjean/filestash/graphs/contributors), folks developing [awesome libraries](https://github.com/mickael-kerjean/filestash/blob/master/go.mod), a whole bunch of C stuff (the [C standard library](https://imgs.xkcd.com/comics/dependency.png), [libjpeg](https://libjpeg-turbo.org/), [libpng](https://www.libpng.org/pub/png/libpng.html), [libgif](https://giflib.sourceforge.net/), [libraw](https://www.libraw.org/about) and many more), [fontawesome](https://fontawesome.com), [material](https://material.io/icons/), [Browser stack](https://www.browserstack.com/) to let us test on real devices, and the many guys from Nebraska and elsewhere who have been thanklessly maintaining the critical pieces that Filestash sits on top:

<img src="https://imgs.xkcd.com/comics/dependency.png" alt="credit to the nebraska guy on xkcd" />
