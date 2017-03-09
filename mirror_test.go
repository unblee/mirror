package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxy_ServeHTTP(t *testing.T) {
	cases := []struct {
		baseDomain string
		listenAddr string
		listenPort string
		destAddr   string
		destPort   string
		expectMsg  string
		expectCode int
	}{
		{
			baseDomain: "127.0.0.1.xip.io",
			listenAddr: "example.127.0.0.1.xip.io",
			listenPort: "6335",
			destAddr:   "127.0.0.1",
			destPort:   "3665",
			expectMsg:  "hello",
			expectCode: 200,
		},
	}

	for _, tc := range cases {
		// Create test server
		tDestServer := newTestDestServer(tc.destAddr, tc.destPort, tc.expectMsg)
		tDestServer.Start()
		defer tDestServer.Close()

		// Set test server URL to testDB
		testDB := newTestDB(tc.destAddr+":"+tc.destPort, "")
		defer testDB.close()

		// Create proxy
		tProxyServer, err := newTestProxyServer(tc.listenPort, tc.destPort, tc.baseDomain, testDB, tc.listenAddr)
		if err != nil {
			t.Fatal(err)
		}
		tProxyServer.Start()
		defer tProxyServer.Close()

		// Get upstream response
		resp, err := http.Get(tProxyServer.URL)
		if err != nil {
			t.Fatalf("Failed to http.Get(proxy): %s", err)
		}
		defer resp.Body.Close()
		actualMsg, _ := ioutil.ReadAll(resp.Body)
		actualCode := resp.StatusCode
		if string(actualMsg) != tc.expectMsg {
			t.Fatalf("Response Body should be '%s', but '%s'", tc.expectMsg, string(actualMsg))
		}
		if actualCode != tc.expectCode {
			t.Fatalf("Response StatusCode should be '%d', but '%d'", tc.expectCode, actualCode)
		}
	}
}

func newTestDestServer(destAddr, destPort, expectMsg string) *httptest.Server {
	s := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, expectMsg)
		}))
	listener, _ := net.Listen("tcp", destAddr+":"+destPort)
	s.Listener = listener
	return s
}

func newTestProxyServer(listenPort, destPort, baseDomain string, testDB DB, listenAddr string) (*httptest.Server, error) {
	p, err := newProxy(destPort, baseDomain, testDB, false)
	if err != nil {
		return nil, err
	}
	proxy := httptest.NewUnstartedServer(p)
	listener, _ := net.Listen("tcp", listenAddr+":"+listenPort)
	proxy.Listener = listener
	return proxy, nil
}

func TestProxy_splitVirtualHostName(t *testing.T) {
	cases := []struct {
		host       string
		baseDomain string
		expected   string
	}{
		{
			host:       "example.foo.bar.127.0.0.1.xip.io:3342",
			baseDomain: "127.0.0.1.xip.io",
			expected:   "example.foo.bar",
		},
		{
			host:       "foo.bar.example.com:80",
			baseDomain: "example.com",
			expected:   "foo.bar",
		},
	}

	for _, tc := range cases {
		p, _ := newProxy("", tc.baseDomain, nil, false)
		actual, err := p.splitVirtualHostName(tc.host)
		if err != nil {
			t.Fatal(err)
		}
		if tc.expected != actual {
			t.Fatalf("Proxy.splitVirtualHostName('%s') => '%s', expected '%s'", tc.host, actual, tc.expected)
		}
	}
}

func TestProxy_fetchDestURL(t *testing.T) {
	cases := []struct {
		virtualHostName string
		rawDestURL      string
		defaultDestURL  string
		expected        string
	}{
		{
			virtualHostName: "exist",
			rawDestURL:      "http://example.com:9999",
			defaultDestURL:  "",
			expected:        "http://example.com:9999",
		},
		{
			virtualHostName: "not.exist",
			rawDestURL:      "",
			defaultDestURL:  "http://example.com:9999",
			expected:        "http://example.com:9999",
		},
		{
			virtualHostName: "exist",
			rawDestURL:      "http://example.com:9999/{}/foo/{}",
			defaultDestURL:  "",
			expected:        "http://example.com:9999/exist/foo/exist",
		},
	}

	for _, tc := range cases {
		testDB := newTestDB(tc.rawDestURL, tc.defaultDestURL)
		p, err := newProxy("", "", testDB, false)
		if err != nil {
			t.Fatal(err)
		}
		actualURL, err := p.fetchDestURL(tc.virtualHostName, "/")
		if err != nil {
			t.Fatal(err)
		}
		if actual := actualURL.String(); tc.expected != actual {
			t.Fatalf("Proxy.fetchDestURL('%s') => '%s' ,expected '%s'", tc.virtualHostName, actual, tc.expected)
		}
	}
}

func TestProxy_buildDestURL(t *testing.T) {
	cases := []struct {
		rawDestURL      string
		virtualHostName string
		destPort        string
		expected        string
	}{
		{
			rawDestURL:      "http://example.com",
			virtualHostName: "Virtual.Host.Name",
			destPort:        "9999",
			expected:        "http://example.com:9999",
		},
		{
			rawDestURL:      "http://example.com:5555",
			virtualHostName: "Virtual.Host.Name",
			destPort:        "9999",
			expected:        "http://example.com:5555",
		},
		{
			rawDestURL:      "http://example.com?q1=foo&q2=bar",
			virtualHostName: "Virtual.Host.Name",
			destPort:        "9999",
			expected:        "http://example.com:9999?q1=foo&q2=bar",
		},
		{
			rawDestURL:      "http://example.com/{}/foo/{}",
			virtualHostName: "Virtual.Host.Name",
			destPort:        "9999",
			expected:        "http://example.com:9999/Virtual.Host.Name/foo/Virtual.Host.Name",
		},
	}

	for _, tc := range cases {
		p, err := newProxy(tc.destPort, "", nil, false)
		if err != nil {
			t.Fatal(err)
		}
		actualURL, err := p.buildDestURL(tc.rawDestURL, tc.virtualHostName)
		if err != nil {
			t.Fatal(err)
		}
		if actual := actualURL.String(); tc.expected != actual {
			t.Fatalf("Proxy.buildDestURL('%s', '%s') => '%s', expected '%s'", tc.rawDestURL, tc.virtualHostName, actual, tc.expected)
		}
	}
}

type TestDB struct {
	desiredData string
	defaultData string
}

func newTestDB(desiredData, defaultData string) DB {
	return &TestDB{
		desiredData: desiredData,
		defaultData: defaultData,
	}
}

func (d *TestDB) get(field string) (string, error) {
	if field == "not.exist" {
		return d.defaultData, nil
	}
	return d.desiredData, nil
}
func (d *TestDB) close() error {
	return nil
}
