package main

import (
	"flag"
	"net/http"
	"net/url"
	"strings"
	"log"
	"time"

	_ "net/http/pprof"

	"github.com/elazarl/goproxy"
	xlog "github.com/xhq928/goutil/logutil"
	"github.com/xhq928/quic-proxy/common"
)

func main() {
	xlog.Debug("client")
	log.SetFlags(log.Llongfile | log.Lmicroseconds | log.Ldate)
	time.Now()
	var (
		listenAddr     string
		proxyUrl       string
		skipCertVerify bool
		auth           string
		verbose        bool
		pprofile       bool
	)

	flag.StringVar(&listenAddr, "l", ":18080", "listenAddr")
	flag.StringVar(&proxyUrl, "proxy", "", "upstream proxy url")
	flag.BoolVar(&skipCertVerify, "k", false, "skip Cert Verify")
	flag.StringVar(&auth, "auth", "quic-proxy:Go!", "basic auth, format: username:password")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&pprofile, "p", false, "http pprof")
	flag.Parse()

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = verbose

	if pprofile {
		pprofAddr := "localhost:6061"
		xlog.Notice("listen pprof:%s", pprofAddr)
		go http.ListenAndServe(pprofAddr, nil)
	}

	Url, err := url.Parse(proxyUrl)
	if err != nil {
		xlog.Error("proxyUrl:%s invalid", proxyUrl)
		return
	}
	if Url.Scheme == "https" {
		xlog.Error("quic-proxy only support http proxy")
		return
	}

	parts := strings.Split(auth, ":")
	if len(parts) != 2 {
		xlog.Error("auth param invalid")
		return
	}
	username, password := parts[0], parts[1]

	proxy.Tr.Proxy = func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxyUrl)
	}

	dialer := common.NewQuicDialer(skipCertVerify)
	proxy.Tr.Dial = dialer.Dial

	// proxy.ConnectDial = proxy.NewConnectDialToProxy(proxyUrl)
	proxy.ConnectDial = proxy.NewConnectDialToProxyWithHandler(proxyUrl,
		SetAuthForBasicConnectRequest(username, password))

	// set basic auth
	proxy.OnRequest().Do(SetAuthForBasicRequest(username, password))

	xlog.Info("start serving %s", listenAddr)
	xlog.Error("%v", http.ListenAndServe(listenAddr, proxy))
}
