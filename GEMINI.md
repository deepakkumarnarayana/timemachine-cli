e # Project Overview

This project is a command-line tool called "Time Machine" written in Go. It's designed to automatically create Git snapshots in a shadow repository, allowing developers to roll back changes easily, especially when working with AI-assisted coding tools. The tool is built using the `cobra` framework for its command-line interface and `fsnotify` for file system watching.

## Building and Running

### Building the project

To build the project, you can use the `make build` command or run the Go build command directly:

```sh
make build
```

or

```sh
go build -o timemachine ./cmd/timemachine
```

### Running the project

To run the project in development mode, you can use the `make dev` command:

```sh
make dev
```

### Running tests

To run the tests, you can use the `make test` command:

```sh
make test
```

## Development Conventions

The project uses Go modules for dependency management. The code is structured with a `cmd` directory for the main application, an `internal` directory for the core logic, and a `scripts` directory for utility scripts. The `Makefile` contains several useful commands for building, testing, and formatting the code.

### Formatting

The code can be formatted using the `make fmt` command:

```sh
make fmt
```

### Linting

The code can be linted using the `make lint` command, which uses `golangci-lint`:

```sh
make lint
```
