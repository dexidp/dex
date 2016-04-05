package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/coreos/dex/client"
	phttp "github.com/coreos/dex/pkg/http"
	"github.com/coreos/dex/pkg/log"
	schema "github.com/coreos/dex/schema/workerschema"
	"github.com/coreos/go-oidc/oidc"
)

type clientResource struct {
	repo client.ClientIdentityRepo
}

func registerClientResource(prefix string, repo client.ClientIdentityRepo) (string, http.Handler) {
	mux := http.NewServeMux()
	c := &clientResource{
		repo: repo,
	}
	relPath := "clients"
	absPath := path.Join(prefix, relPath)
	mux.Handle(absPath, c)
	return relPath, mux
}

func (c *clientResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c.list(w, r)
	case "POST":
		c.create(w, r)
	default:
		msg := fmt.Sprintf("HTTP %s method not supported for this resource", r.Method)
		writeAPIError(w, http.StatusMethodNotAllowed, newAPIError(errorInvalidRequest, msg))
	}
}

func (c *clientResource) list(w http.ResponseWriter, r *http.Request) {
	cs, err := c.repo.All()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, "error listing clients"))
		return
	}

	scs := make([]*schema.Client, len(cs))
	for i, ci := range cs {
		sc := schema.MapClientIdentityToSchemaClient(ci)
		scs[i] = &sc
	}

	page := schema.ClientPage{
		Clients: scs,
	}
	writeResponseWithBody(w, http.StatusOK, page)
}

func (c *clientResource) create(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		log.Debugf("Unsupported request content-type: %v", ct)
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest, "unsupported content-type"))
		return
	}

	var sc schema.Client
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&sc)
	if err != nil {
		log.Debugf("Error decoding request body: %v", err)
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidRequest, "unable to decode request body"))
		return
	}

	ci, err := schema.MapSchemaClientToClientIdentity(sc)
	if err != nil {
		log.Debugf("Invalid request data: %v", err)
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidClientMetadata, "missing or invalid field: redirectURIs"))
		return
	}

	if err := ci.Metadata.Valid(); err != nil {
		log.Debugf("ClientMetadata invalid: %v", err)
		writeAPIError(w, http.StatusBadRequest, newAPIError(errorInvalidClientMetadata, err.Error()))
		return
	}

	clientID, err := oidc.GenClientID(ci.Metadata.RedirectURIs[0].Host)
	if err != nil {
		log.Errorf("Failed generating ID for new client: %v", err)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, "unable to generate client ID"))
		return
	}

	creds, err := c.repo.New(clientID, ci.Metadata, false)
	if err != nil {
		log.Errorf("Failed creating client: %v", err)
		writeAPIError(w, http.StatusInternalServerError, newAPIError(errorServerError, "unable to create client"))
		return
	}
	ci.Credentials = *creds

	ssc := schema.MapClientIdentityToSchemaClientWithSecret(ci)
	w.Header().Add("Location", phttp.NewResourceLocation(r.URL, ci.Credentials.ID))
	writeResponseWithBody(w, http.StatusCreated, ssc)
}
