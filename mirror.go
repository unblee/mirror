package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus" // used by vulcand/oxy
)

var (
	Version     string
	Revision    string
	GoVersion   string
	ShowVersion = flag.Bool("version", false, "show version")
)

const UsageMsg = `
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
`

const (
	DEFAULT_REDIS_HASH_KEY = "mirror-store" // Use this hash key in Redis

	DEFAULT_LISTEN_PORT       = "8080"
	DEFAULT_DEFAULT_DEST_PORT = "80"
	DEFAULT_DB_HOST           = "127.0.0.1"
	DEFAULT_DB_PORT           = "6379"
)

func main() {
	flag.Usage = func() {
		fmt.Println(UsageMsg)
		os.Exit(0)
	}
	flag.Parse()
	if *ShowVersion {
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Revision:   %s\n", Revision)
		fmt.Printf("Go version: %s\n", GoVersion)
		os.Exit(0)
	}

	destPort := os.Getenv("DEST_PORT")
	if destPort == "" {
		destPort = DEFAULT_DEFAULT_DEST_PORT
	}
	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		log.Fatalln("Please set environment variable 'BASE_DOMAIN'")
	}
	defaultDestURL := os.Getenv("DEFAULT_DEST_URL") // Allow empty 'DEFAULT_DEST_URL' environment variable

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = DEFAULT_DB_HOST
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = DEFAULT_DB_PORT
	}
	redisHashKey := os.Getenv("REDIS_HASH_KEY")
	if redisHashKey == "" {
		redisHashKey = DEFAULT_REDIS_HASH_KEY
	}

	redi, err := newRedis(dbHost, dbPort, defaultDestURL, redisHashKey)
	if err != nil {
		log.Fatal(err)
	}
	defer redi.close()

	enableStream := false
	if os.Getenv("STREAM") != "" {
		enableStream = true
	}

	proxy, err := newProxy(destPort, baseDomain, redi, enableStream)
	if err != nil {
		log.Fatal(err)
	}

	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = DEFAULT_LISTEN_PORT
	}

	s := newProxyServer(proxy, listenPort)
	s.ListenAndServe()
}
