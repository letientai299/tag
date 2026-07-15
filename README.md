# tag - Tag your ripgrep matches

> Fork from https://github.com/aykamko/tag, as I like the tool and want to keep
> using it.

==== ![revolv++](tag.gif)

`tag` is a lightweight wrapper around
**[ripgrep](https://github.com/BurntSushi/ripgrep)** that generates shell
aliases for your search matches. tag is a very fast Golang reimagining of
[sack](https://github.com/sampson-chen/sack). `tag` supports only
`ripgrep (rg)`. If you'd like to add support for more search backends, I
encourage you to contribute!

## Why should I use tag?

`tag` makes it easy to _immediately_ jump to a search match in your favorite
editor. It eliminates the tedious task of typing `vim foo/bar/baz.qux +42` to
jump to a match by automatically generating these commands for you as shell
aliases.

Inside vim, [vim-grepper](https://github.com/mhinz/vim-grepper) or
[ag.vim](https://github.com/rking/ag.vim) is probably the way to go. Outside vim
(or inside a Neovim `:terminal`), `tag` is your best friend.

Finally, `tag` is unobtrusive. It should behave exactly like `ripgrep` under
most circumstances.

## Performance Benchmarks

`tag` processes `rg`'s output on-the-fly with Golang using pipes so the
performance loss is neglible. In other words, **`tag` is just as fast as rg**!

```text
$ cd ~/github/torvalds/linux
$ time ( for _ in {1..10}; do  rg EXPORT_SYMBOL_GPL >/dev/null 2>&1; done )
16.66s user 16.54s system 347% cpu 9.562 total
$ time ( for _ in {1..10}; do tag EXPORT_SYMBOL_GPL >/dev/null 2>&1; done )
16.84s user 16.90s system 356% cpu 9.454 total
```

## Installation

1. Install the `tag` binary with `go install`:

   ```sh
   go install github.com/letientai299/tag@latest
   ```

   Or compile from source:

   ```sh
   git clone https://github.com/letientai299/tag.git
   cd tag
   go build
   ```

2. Since tag generates a file with command aliases for your shell, you'll have
   to drop the following in your `bashrc`/`zshrc` to actually pick up those
   aliases.
   - `bash`

     ```bash
     if hash rg 2>/dev/null; then
       tag() { command tag "$@"; source ${TAG_ALIAS_FILE:-/tmp/tag_aliases} 2>/dev/null; }
       alias rg=tag
     fi
     ```

   - `zsh`

     ```zsh
     if (( $+commands[tag] )); then
       tag() { command tag "$@"; source ${TAG_ALIAS_FILE:-/tmp/tag_aliases} 2>/dev/null }
       alias rg=tag
     fi
     ```

   - `fish - ~/.config/fish/functions/tag.fish`

     ```fish
     function tag
         set -q TAG_ALIAS_FILE; or set -l TAG_ALIAS_FILE /tmp/tag_aliases
         command tag $argv; and source $TAG_ALIAS_FILE ^/dev/null
         alias rg tag
     end
     ```

## Configuration

`tag` exposes the following configuration options via environment variables:

- `TAG_ALIAS_FILE`
  - Path where shortcut alias file will be generated.
  - Default: `/tmp/tag_aliases`
- `TAG_ALIAS_PREFIX`
  - Prefix for alias commands, e.g. the `e` in generated alias `e42`.
  - Default: `e`
- `TAG_CMD_FMT_STRING`
  - Format string for alias commands. Must contain `{{.Filename}}`,
    `{{.LineNumber}}`, and `{{.ColumnNumber}}` for proper substitution.
  - Default:
    `vim -c 'call cursor({{.LineNumber}}, {{.ColumnNumber}})' '{{.Filename}}'`

## License

[MIT](LICENSE)

## Author

[aykamko](https://github.com/aykamko)
