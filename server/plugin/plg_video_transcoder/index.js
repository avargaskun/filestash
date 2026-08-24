export default function (mime, url, {
    enabled = {{ .enabled }},
    presets = {{ .presets }},
    defaultPreset = "{{ .defaultPreset }}",
    forceTranscodeDefault = {{ .forceTranscodeDefault }},
} = {}) {
    let transcodable = enabled;
    {{- range $item := splitList "," .blacklist }}
    if (mime == "{{ $item | trim }}") transcodable = false;
    {{- end }}
    return {
        original: [[mime, url]],
        hls: (quality) => url + "&transcode=hls&preset=" + encodeURIComponent(quality),
        transcodable,
        presets,
        defaultPreset,
        forceTranscodeDefault,
    };
}
