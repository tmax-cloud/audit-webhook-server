package main

import (
	"net/http"

	"github.com/tmax-cloud/audit-webhook-server/audit"
	"github.com/tmax-cloud/audit-webhook-server/dataFactory"
	"github.com/tmax-cloud/audit-webhook-server/util"
	"k8s.io/klog"
)

func main() {
	util.UpdateAuditResourceList()
	dataFactory.CreateConnection()
	//flag.StringVar(&dataFactory.DBPassWordPath, "dbPassword", "/run/secrets/timescaledb/password", "Timescaledb Server Password")

	mux := http.NewServeMux()
	mux.HandleFunc("/audit", serveAudit)
	mux.HandleFunc("/audit/member_suggestions", serveAuditMemberSuggestions)
	mux.HandleFunc("/audit/batch", serveAuditBatch)
	mux.HandleFunc("/audit/resources", serveAuditResources)
	mux.HandleFunc("/audit/verb", serveAuditVerb)
	mux.HandleFunc("/audit/websocket", serveAuditWss)

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

func serveAuditVerb(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		audit.ListAuditVerb(w, r)
	default:
		//error
	}
}

func serveAuditResources(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		audit.ListAuditResource(w, r)
	default:
		//error
	}
}

func serveAuditMemberSuggestions(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		audit.MemberSuggestions(w, r)
	default:
	}
}

func serveAuditBatch(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	audit.AddAuditBatch(w, r)
}

func serveAuditWss(w http.ResponseWriter, r *http.Request) {
	klog.Infof("Http request: method=%s, uri=%s", r.Method, r.URL.Path)
	audit.ServeWss(w, r)
}

func serveTest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		util.SetResponse(w, "Hello Audit Test", nil, http.StatusOK)
	case http.MethodPost:
	case http.MethodPut:
	case http.MethodDelete:
	default:
		//error
	}
}
