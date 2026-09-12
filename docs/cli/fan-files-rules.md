# fan-files rules

Rules management utility

## Synopsis

On each subcommand you'll have available at least two flags:
"username" and "id". You must either set only one of them
or none. If you set one of them, the command will apply to
an user, otherwise it will be applied to the global set or
rules.

## Options

```
  -h, --help              help for rules
  -i, --id uint           id of user to which the rules apply
  -u, --username string   username of user to which the rules apply
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./fan-files.db")
```

## See Also

* [fan-files](fan-files.md)	 - A stylish web-based file browser
* [fan-files rules add](fan-files-rules-add.md)	 - Add a global rule or user rule
* [fan-files rules ls](fan-files-rules-ls.md)	 - List global rules or user specific rules
* [fan-files rules rm](fan-files-rules-rm.md)	 - Remove a global rule or user rule

