package http_proxy_router

import (
	"context"
	"gateway/golang_common/lib"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

var (
	pprofSrv  *http.Server
	pprofOnce sync.Once
)

func StartPprof() {
	addr := lib.GetStringConf("proxy.pprof.addr")
	if addr == "" {
		return
	}
	pprofOnce.Do(func() {
		pprofSrv = &http.Server{
			Addr:         addr,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 30 * time.Second,
		}
		go func() {
			log.Printf(" [INFO] pprof_run %s\n", addr)
			if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf(" [ERROR] pprof_run %s err:%v\n", addr, err)
			}
		}()
	})
}

func StopPprof() {
	if pprofSrv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pprofSrv.Shutdown(ctx); err != nil {
		log.Printf(" [ERROR] pprof_stop err:%v\n", err)
	}
}
