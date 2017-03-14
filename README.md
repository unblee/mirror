# mirror

VirtualHost based dynamic reverse proxy

TODO: write document

## Available Environment Variables

```
Flags:
  -h, --help: this message
  --version:  show version

Available Environment Variables:
  LISTEN_PORT              Listening port number
    default: 8080
    e.g.) LISTEN_PORT=8080

  DEFAULT_DEST_PORT        Default destination port number
    default: 80
    e.g.) DEFAULT_DEST_PORT=80

  BASE_DOMAIN              Base Domain
    default: not set(required)
    e.g.) BASE_DOMAIN=127.0.0.1.xip.io
    e.g.) BASE_DOMAIN=example.com

  DEFAULT_DEST_URL         Default destination URL
    default: not set(allow empty)
    e.g.) DEFAULT_DEST_URL=http://127.0.0.1:5000

  DB_HOST                  Hostname of the database to connect
    default: 127.0.0.1
    e.g.) DB_HOST=127.0.0.1

  DB_PORT                  Port number of the database to connect
    default: 6379
    e.g.) DB_PORT=6379

  REDIS_HASH_KEY           Hash key of Redis
    default: mirror-store
    e.g.) REDIS_HASH_KEY=mirror-store

  STREAM                   Enable stream support
    default: off(false)
    e.g.) STREAM=on
```