# Release Notes Version 0.5
Feb 5, 2026

- Bumped version to 0.5 and copyright stmt to 2026.
- Fixed some modern code warnings / lints.
- Tested with Go 1.25.6.
- Fixed failing `test.sh` test.
- Allow "journaltitle" as an alternative to "journal" (#11).
- Allow "date" instead of "year" (#11).
- `check` command produces warning for 1--X page ranges.
- Added some tests in `tests.sh` for `check` command.
- Don't change case of sTraNge-caSe journal title words.
- Fixed title-casing bugs in journal names with nested {} regions.
- Blessed field name `biblint__options`. This field name is reserved for future 
  biblint use. At present, it doesn't do anything, but in the future, it may be
  used to hold per-entry options. The only change in this release is that if
  this field is present, it will be kept in the output unchanged.
- Added option to merge similar journal names into symbols during `clean` --- 
  this feature is off by default because it is experimental and also b/c it
  can mangle journal names. But it can also help normalize journal names.
  Use `--merge-journal-names=k` with `k` equal to the minimum number of entries 
  of similar journal names needed to symbolize a name.

