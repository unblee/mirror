# mirror

VirtualHost based dynamic reverse proxy

TODO: write document

## Available Environment Values

```
Flags:
  -h, --help: this message
  --version:  show version

Available Environment Values:
  LISTEN_PORT              Listening port number
    default: 8080

  DEFAULT_DEST_PORT        Default destination port number
    default: 80

  BASE_DOMAIN              Base Domain
    default: not set(required)

  DEFAULT_DEST_URL         Default destination URL
    default: not set(allow empty)

  DB_HOST                  Hostname of the database to connect
    default: 127.0.0.1

  DB_PORT                  Port number of the database to connect
    default: 6379

  DEFAULT_REDIS_HASH_KEY   Hash key of Redis
    default: mirror-store
```