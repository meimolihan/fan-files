# fan-files completion fish

Generate the autocompletion script for fish

## Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	fan-files completion fish | source

To load completions for every new session, execute once:

	fan-files completion fish > ~/.config/fish/completions/fan-files.fish

You will need to start a new shell for this setup to take effect.


```
fan-files completion fish [flags]
```

## Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./fan-files.db")
```

## See Also

* [fan-files completion](fan-files-completion.md)	 - Generate the autocompletion script for the specified shell

