package main

import (
	"net/http"

	"github.com/tmax-cloud/audit-webhook-server/audit"
	"github.com/tmax-cloud/audit-webhook-server/util"
	"k8s.io/klog"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/audit", serveAudit)
	mux.HandleFunc("/test", serveTest)

	klog.Info("Starting Audit Webhook server...")
	klog.Flush()

	if err := http.ListenAndServe(":80", mux); err != nil {
		klog.Errorf("Failed to listen and serve Audit Webhook server: %s", err)
	}
	klog.Info("Terminate Audit Webhook server")
}

func serveAudit(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		audit.GetAudit(w, r)
	case http.MethodPost:
		audit.AddAudit(w, r)
	case http.MethodPut:
	case http.MethodDelete:
	default:
		//error
	}
}

func serveTest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		util.SetResponse(r, "Hello Audit Test", nil, http.StatusOK)
	case http.MethodPost:
	case http.MethodPut:
	case http.MethodDelete:
	default:
		//error
	}
}
