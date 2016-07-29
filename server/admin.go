package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path"

	"github.com/coreos/pkg/health"
	"github.com/julienschmidt/httprouter"

	"github.com/coreos/dex/admin"
	"github.com/coreos/dex/connector"
	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/schema/adminschema"
	"github.com/coreos/go-oidc/key"
)

const (
	AdminAPIVersion      = "v1"
	AdminAPISecretLength = 128
)

var (
	AdminGetEndpoint          = addBasePath("/admin/:id")
	AdminCreateEndpoint       = addBasePath("/admin")
	AdminGetStateEndpoint     = addBasePath("/state")
	AdminCreateClientEndpoint = addBasePath("/client")
	AdminConnectorsEndpoint   = addBasePath("/connectors")
)

// AdminServer serves the admin API.
type AdminServer struct {
	adminAPI *admin.AdminAPI
	checker  health.Checker
	secret   string
}

func NewAdminServer(adminAPI *admin.AdminAPI, rotator *key.PrivateKeyRotator, secret string) *AdminServer {
	return &AdminServer{
		adminAPI: adminAPI,
		checker: health.Checker{
			Checks: []health.Checkable{
				rotator,
			},
		},
		secret: secret,
	}
}

func (s *AdminServer) HTTPHandler() http.Handler {
	r := httprouter.New()
	r.GET(AdminGetEndpoint, s.getAdmin)
	r.POST(AdminCreateEndpoint, s.createAdmin)
	r.GET(AdminGetStateEndpoint, s.getState)
	r.POST(AdminCreateClientEndpoint, s.createClient)
	r.Handler("GET", httpPathHealth, s.checker)
	r.HandlerFunc("GET", httpPathDebugVars, health.ExpvarHandler)
	r.PUT(AdminConnectorsEndpoint, s.setConnectors)
	r.GET(AdminConnectorsEndpoint, s.getConnectors)

	return authorizer(r, s.secret, httpPathHealth, httpPathDebugVars)
}

func authorizer(h http.Handler, secret string, public ...string) http.Handler {
	publicSet := map[string]struct{}{}
	for _, p := range public {
		publicSet[p] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, isPublicPath := publicSet[r.URL.Path]

		if !isPublicPath && r.Header.Get("Authorization") != secret {
			writeAPIError(w, http.StatusUnauthorized, newAPIError(errorAccessDenied, ""))
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *AdminServer) getAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	admn, err := s.adminAPI.GetAdmin(id)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeResponseWithBody(w, http.StatusOK, admn)
}

func (s *AdminServer) createAdmin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	admn := adminschema.Admin{}
	err := json.NewDecoder(r.Body).Decode(&admn)
	if err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}

	id, err := s.adminAPI.CreateAdmin(admn)
	if err != nil {
		s.writeError(w, err)
		return
	}

	admn.Id = id
	w.Header().Set("Location", AdminCreateEndpoint+"/"+id)
	writeResponseWithBody(w, http.StatusOK, admn)
}

func (s *AdminServer) getState(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	state, err := s.adminAPI.GetState()
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeResponseWithBody(w, http.StatusOK, state)
}

func (s *AdminServer) createClient(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var req = adminschema.ClientCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}

	resp, err := s.adminAPI.CreateClient(req)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeResponseWithBody(w, http.StatusOK, &resp)
}

func (s *AdminServer) setConnectors(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var req struct {
		Connectors json.RawMessage `json:"connectors"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}

	connectorConfigs, err := connector.ReadConfigs(bytes.NewReader([]byte(req.Connectors)))
	if err != nil {
		writeInvalidRequest(w, "cannot parse JSON body")
		return
	}
	if err := s.adminAPI.SetConnectors(connectorConfigs); err != nil {
		s.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *AdminServer) getConnectors(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	connectorConfigs, err := s.adminAPI.GetConnectors()
	if err != nil {
		s.writeError(w, err)
		return
	}
	var resp adminschema.ConnectorsGetResponse
	resp.Connectors = make([]interface{}, len(connectorConfigs))
	for i, connectorConfig := range connectorConfigs {
		resp.Connectors[i] = connectorConfig
	}
	writeResponseWithBody(w, http.StatusOK, &resp)
}

func (s *AdminServer) writeError(w http.ResponseWriter, err error) {
	log.Errorf("Error calling admin API: %v: ", err)
	if adminErr, ok := err.(admin.Error); ok {
		writeAPIError(w, adminErr.Code, newAPIError(adminErr.Type, adminErr.Desc))
		return
	}

	writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, err.Error()))
}

func writeInvalidRequest(w http.ResponseWriter, msg string) {
	writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest, msg))
}

func addBasePath(s string) string {
	return path.Join(httpPathAPI, APIVersion, s)
}
