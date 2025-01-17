# Omnistrate CTL

Omnistrate CTL is a command line tool designed to streamline the creation, deployment, and management of your Omnistrate SaaS. Use it to build services from docker-compose files, manage service plans, and interact with the Omnistrate platform efficiently.

See more details on the [documentation](./mkdocs/docs/index.md)

## Configuration

Omnistrate CTL support configuration using environment variables

| Environment Variable          | Description                                                                                     |
| ----------------------------- | ----------------------------------------------------------------------------------------------- |
| `OMNISTRATE_DRY_RUN`          | Specifies whether the application should run in dry-run mode, where no actual changes are made. |
| `OMNISTRATE_LOG_LEVEL`        | Defines the logging level (e.g., DEBUG, INFO, WARN, ERROR).                                     |
| `OMNISTRATE_LOG_FORMAT_LEVEL` | Determines the format of log output (e.g., json, pretty).                                       |
| `OMNISTRATE_ROOT_DOMAIN`      | The root domain for the Omnistrate platform.                                                    |
| `OMNISTRATE_HOST_SCHEME`      | The protocol scheme to use for the host (e.g., HTTP, HTTPS).                                    |

