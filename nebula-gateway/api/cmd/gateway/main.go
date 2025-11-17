package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nebula/gateway/internal/common"
	didcontroller "github.com/nebula/gateway/internal/didcontract/controller"
	didservice "github.com/nebula/gateway/internal/didcontract/service"
	didtransport "github.com/nebula/gateway/internal/didcontract/transport"
	jobcontroller "github.com/nebula/gateway/internal/jobcontract/controller"
	jobservice "github.com/nebula/gateway/internal/jobcontract/service"
	jobtransport "github.com/nebula/gateway/internal/jobcontract/transport"
	nationcontroller "github.com/nebula/gateway/internal/nationcontract/controller"
	nationservice "github.com/nebula/gateway/internal/nationcontract/service"
	nationtransport "github.com/nebula/gateway/internal/nationcontract/transport"
	statecontroller "github.com/nebula/gateway/internal/statecontract/controller"
	stateservice "github.com/nebula/gateway/internal/statecontract/service"
	statetransport "github.com/nebula/gateway/internal/statecontract/transport"
)

func main() {
	cfg, err := common.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fabricClient := common.NewFabricClient(cfg)
	if err := fabricClient.WaitForChannelReady(2 * time.Minute); err != nil {
		log.Fatalf("fabric channel not ready: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(cfg))

	initStateContract(mux, cfg, fabricClient)
	initJobContract(mux, cfg, fabricClient)
	initDIDContract(mux)
	initNationContract(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("nebula gateway listening on %s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func initJobContract(mux *http.ServeMux, cfg *common.Config, fabric *common.FabricClient) {
	transport := jobtransport.NewTransport(fabric)
	svc := jobservice.NewService(transport)
	handler := jobcontroller.NewHandler(cfg, svc)
	handler.RegisterRoutes(mux)
}

func initStateContract(mux *http.ServeMux, cfg *common.Config, fabric *common.FabricClient) {
	transport := statetransport.NewTransport(fabric)
	svc := stateservice.NewService(transport)
	handler := statecontroller.NewHandler(cfg, svc)
	handler.RegisterRoutes(mux)
}

func initDIDContract(mux *http.ServeMux) {
	transport := didtransport.NewTransport()
	svc := didservice.NewService(transport)
	handler := didcontroller.NewHandler(svc)
	handler.RegisterRoutes(mux)
}

func initNationContract(mux *http.ServeMux) {
	transport := nationtransport.NewTransport()
	svc := nationservice.NewService(transport)
	handler := nationcontroller.NewHandler(svc)
	handler.RegisterRoutes(mux)
}

func healthHandler(cfg *common.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		peer := cfg.ResolvePeer(r.URL.Query().Get("peer"))
		common.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"peer":   peer,
		})
	}
}
