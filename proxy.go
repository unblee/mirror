package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus" // used by vulcand/oxy
	"github.com/pkg/errors"
	"github.com/vulcand/oxy/forward"
)

type Proxy struct {
	forwarder       *forward.Forwarder
	defaultDestPort string
	baseDomain      string
	db              DB
	enableStream    bool
}

func newProxyServer(handler *Proxy, listenPort string) *http.Server {
	return &http.Server{
		Addr:    ":" + listenPort,
		Handler: handler,
	}
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
