# unlimitedClaw

A progressive Go AI assistant — learning by building, inspired by PicoClaw.

## Quick Start

Build the binary:

```bash
go build -o build/unlimitedclaw ./cmd/unlimitedclaw
```

Run the binary:

```bash
./build/unlimitedclaw --help
./build/unlimitedclaw version
./build/unlimitedclaw agent --help
./build/unlimitedclaw gateway --help
```

## Architecture

See `docs/study/` for architecture guides and learning materials.

## Development

### Dependencies

- Go 1.25+
- [cobra](https://github.com/spf13/cobra) - CLI framework

### Project Structure

```
unlimitedClaw/
├── cmd/
│   └── unlimitedclaw/        # CLI entry point
├── pkg/                        # Public packages
├── internal/                   # Internal packages
├── config/
│   ├── config.example.json    # Configuration template
│   └── secrets/               # Secret configuration (not in git)
├── docs/
│   └── study/                 # Learning documentation
├── scripts/                   # Build and utility scripts
├── docker/                    # Docker configurations
├── k8s/                       # Kubernetes configurations
├── build/                     # Build output directory
├── go.mod                     # Go module definition
└── README.md                  # This file
```

## License

MIT License
