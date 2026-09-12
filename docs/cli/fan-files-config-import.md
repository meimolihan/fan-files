# fan-files config import

Import a configuration file

## Synopsis

Import a configuration file. This will replace all the existing
configuration. Can be used with or without unexisting databases.

If used with a nonexisting database, a key will be generated
automatically. Otherwise the key will be kept the same as in the
database.

The path must be for a json or yaml file.

```
fan-files config import <path> [flags]
```

## Options

```
  -h, --help   help for import
```

## Options inherited from parent commands

```
  -c, --config string     config file path
  -d, --database string   database path (default "./fan-files.db")
```

## See Also

* [fan-files config](fan-files-config.md)	 - Configuration management utility

