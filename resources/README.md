# Resources

This directory is the migration target and current policy boundary for vendor-derived assets used by the app at runtime or in tests.

- `themes/` contains curated `.data` theme files copied from the local `35inchENG/` reference bundle.
- `testdata/` contains reduced fixtures derived from curated themes for parser tests.

Current policy:

1. New application and test code must load vendor-derived assets from `resources/`, not `35inchENG/`.
2. Existing direct `35inchENG/` references may remain temporarily during the migration, but they are transitional and must be removed from runtime code.
3. New vendor-derived assets must be copied here before use.
4. Only the minimum files needed by the selected theme should be committed.
