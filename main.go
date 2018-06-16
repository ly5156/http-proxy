package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var (
	staticFileDir string
	port          string
	proxyAddr     string
	addrPrefix    string
	stripPrefix   bool
)

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&staticFileDir, "dir", currentDir, "静态文件目录，默认为当前工作目录")
	flag.StringVar(&port, "port", ":9090", "服务监听端口，以冒号开头")
	flag.StringVar(&proxyAddr, "proxyAddr", "http://api-dev:9500", "代理地址")
	flag.StringVar(&addrPrefix, "addrPrefix", "/api", "拦截地址前缀")
	flag.BoolVar(&stripPrefix, "stripPrefix", false, "是否清除拦截地址前缀")

}
func main() {
	flag.Parse()
	staticFileDir, err := filepath.Abs(staticFileDir)
	if err != nil {
		log.Fatal(err)
	}
	targetUrl, err := url.Parse(proxyAddr)
	if err != nil {
		log.Fatal(err)
	}
	httpProxy := httputil.NewSingleHostReverseProxy(targetUrl)
	url := "http://localhost" + port + "/"
	fsh := http.FileServer(http.Dir(staticFileDir))
	router := httprouter.New()
	router.NotFound = fsh
	var h httprouter.Handle
	if stripPrefix {
		h = convertToHandle(http.StripPrefix(addrPrefix, httpProxy))
	} else {
		h = convertToHandle(httpProxy)
	}
	router.Handle("GET", addrPrefix+"/*filepath", h)
	router.Handle("POST", addrPrefix+"/*filepath", h)
	srv := &http.Server{
		Addr:    port,
		Handler: router,
	}
	go func() {
		fmt.Println("访问地址:", url)
		fmt.Println("服务目录:", staticFileDir)
		fmt.Println("代理地址:", proxyAddr)
		fmt.Println("拦截路径", addrPrefix)
		fmt.Println("Start Server...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")

}
func convertToHandle(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}
