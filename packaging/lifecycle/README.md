# Lifecycle packaging manifest

`owned-paths.json` is the reviewed list of InfoBoard-owned artifacts removed by
uninstall. The installer or host composition supplies the absolute root; an
uninstall request never supplies a path.

The Go enforcement lives in `host/lifecycle` (`Manifest`, `Cleaner`, and
`UninstallPlan`). It rejects absolute, parent-traversing, wildcard, duplicate,
root-level, and symlink targets, and it never follows a link while removing a
directory tree.

Platform native-messaging manifests, host binaries, and extension package
artifacts remain deferred until IP-18 freezes their platform-specific locations.
They must be added as named entries, not inferred wildcards.
