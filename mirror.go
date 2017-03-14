package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus" // used by vulcand/oxy
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/vulcand/oxy/forward"
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

func newProxyServer(handler *Proxy, listenPort string) *http.Server {
	return &http.Server{
		Addr:    ":" + listenPort,
		Handler: handler,
	}
}

type Proxy struct {
	forwarder       *forward.Forwarder
	defaultDestPort string
	baseDomain      string
	db              DB
	enableStream    bool
}

func newProxy(defaultDestPort, baseDomain string, db DB, enableStream bool) (*Proxy, error) {
	fwd, err := forward.New(forward.Stream(enableStream))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize forward proxy")
	}

	p := &Proxy{
		forwarder:       fwd,
		defaultDestPort: defaultDestPort,
		baseDomain:      baseDomain,
		db:              db,
		enableStream:    enableStream,
	}

	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	vhost, err := p.splitVirtualHostName(req.Host)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "404 Upstream Not Found")
		log.Errorln(err)
		return
	}
	req.URL, err = p.fetchDestURL(vhost, req.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "404 Upstream Not Found")
		log.Errorln(err)
		return
	}

	// First aid
	if p.enableStream {
		req.RequestURI = "/"
	} else {
		req.RequestURI = req.URL.Path
	}

	p.forwarder.ServeHTTP(w, req)
}

// vhost:      e.g.) example.foo.bar.127.0.0.1.xip.io:3344
// baseDomain: e.g.)                 127.0.0.1.xip.io
// return:     e.g.) example.foo.bar
func (p *Proxy) splitVirtualHostName(vHostName string) (string, error) {
	var host string
	var err error

	// trim a port number
	if strings.Contains(vHostName, ":") {
		host, _, err = net.SplitHostPort(vHostName)
		if err != nil {
			return "", errors.Wrap(err, "Failed split VirtualHost name")
		}
	} else {
		host = vHostName
	}

	return strings.TrimSuffix(host, "."+p.baseDomain), nil
}

func (p *Proxy) fetchDestURL(virtualHostName, origReqPath string) (*url.URL, error) {
	// rawDest: e.g.) "http://example.com"
	//          e.g.) "http://example.com:9999"
	//          e.g.) "http://example.com:9999/target/path"
	//          e.g.) "http://example.com:9999/target?q1=foo&q2=bar
	//          e.g.) "http://example.com:9999/{}/path" '{}' is replaced the virtual host name
	rawDest, err := p.db.get(virtualHostName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed get upstream URL from database")
	}
	if rawDest == "" {
		return nil, errors.Wrap(err, "Not exists upstream")
	}

	if !strings.Contains(rawDest, "://") {
		rawDest = "http://" + rawDest
	}

	// remove path
	if origReqPath != "/" {
		u, err := url.Parse(rawDest)
		if err != nil {
			return nil, errors.Wrapf(err, "Invalid Upstream URL '%s'", rawDest)
		}
		u.Path = origReqPath
		rawDest = u.String()
	}

	destURL, err := p.buildDestURL(rawDest, virtualHostName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed build upstream URL")
	}
	return destURL, nil
}

func (p *Proxy) buildDestURL(rawDest, virtualHostName string) (*url.URL, error) {
	r := strings.Replace(rawDest, "{}", virtualHostName, -1)
	rawDestURL, err := url.Parse(r)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid Upstream URL '%s'", rawDest)
	}

	var hostport string
	if strings.Contains(rawDestURL.Host, ":") { // when contains a port number
		hostport = rawDestURL.Host
	} else {
		hostport = net.JoinHostPort(rawDestURL.Host, p.defaultDestPort)
	}

	dest := rawDestURL.Scheme + "://" + hostport + rawDestURL.Path
	if rawDestURL.RawQuery != "" {
		dest = dest + "?" + rawDestURL.RawQuery
	}

	destURL, err := url.Parse(dest)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid Upstream URL '%s'", dest)
	}
	return destURL, nil
}

type DB interface {
	get(field string) (string, error)
	close() error
}

type Redis struct {
	conn    redis.Conn
	hashKey string
}

func newRedis(host, port, defaultDestURL, hashKey string) (DB, error) {
	c, err := redis.Dial("tcp", host+":"+port)
	if err != nil {
		return nil, errors.Wrap(err, "Failed start connection to Redis")
	}
	_, err = c.Do("HSET", hashKey, "default", defaultDestURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed Redis Command 'HSET %s default %s'", hashKey, defaultDestURL)
	}
	r := &Redis{
		conn:    c,
		hashKey: hashKey,
	}
	return r, nil
}

// If the field does not exist, then the return value is the value of the "default" key
func (r *Redis) get(field string) (string, error) {
	reply, err := redis.String(r.conn.Do("HGET", r.hashKey, field))
	switch {
	case err == redis.ErrNil: // when the field not exist field
		reply, _ = redis.String(r.conn.Do("HGET", r.hashKey, "default"))
	case err != nil:
		return "", errors.Wrapf(err, "Failed Redis Command 'HGET %s %s'", r.hashKey, field)
	}
	return reply, nil
}

func (r *Redis) close() error {
	err := r.conn.Close()
	if err != nil {
		return errors.Wrap(err, "Failed close connection to Redis")
	}
	return nil
}
