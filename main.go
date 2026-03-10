package main

import (
	"flag"
	"gateway/ai_gateway"
	"gateway/dao"
	"gateway/golang_common/lib"
	"gateway/http_proxy_pipeline"
	"gateway/http_proxy_plugin"
	"gateway/http_proxy_router"
	"gateway/router"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	endpoint = flag.String("endpoint", "", "input endpoint dashboard or server")
	config   = flag.String("config", "", "input config file like ./conf/dev/")
)

func main() {
	flag.Parse()
	if *endpoint == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *config == "" {
		flag.Usage()
		os.Exit(1)
	}

	lib.InitModule(*config)
	defer lib.Destroy()

	if err := http_proxy_plugin.RegisterBuiltinPlugins(); err != nil {
		log.Fatalf("[ERROR] builtin plugin register failed: %v", err)
	}

	if *endpoint == "dashboard" {
		router.HttpServerRun()

		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		router.HttpServerStop()
		return
	}

	dao.ServiceManagerHandler.LoadOnce()
	dao.AppManagerHandler.LoadOnce()

	if err := ai_gateway.Bootstrap(); err != nil {
		log.Printf("[WARN] ai_gateway bootstrap failed: %v", err)
	}
	if err := http_proxy_pipeline.ReloadAIServiceConfigRuntime(0); err != nil {
		log.Printf("[WARN] preload ai service config runtime failed: %v", err)
	}

	go func() {
		http_proxy_router.HttpServerRun()
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	http_proxy_router.HttpServerStop()
}
