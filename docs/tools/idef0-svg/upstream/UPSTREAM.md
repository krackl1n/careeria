Vendored from [jimmyjazz/IDEF0-SVG](https://github.com/jimmyjazz/IDEF0-SVG),
commit `f689fe913260e0582905a9cb8ef7434c960112ea` (MIT license).

The `lib/` directory is the upstream Ruby layout and SVG generator. Local
color configuration and CLI integration live in `../render.rb`. Two local
changes escape XML text in `lib/idef0/labels.rb` and
`lib/idef0/process_box.rb`. Russian verbs, text measurement and wrapped block
titles are implemented in `../language.rb` without changing the vendored code.
