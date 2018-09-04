package ui

import "net/http"

func UI() http.Handler {
	h := http.FileServer(assetFS())
	return &idpUI{h, http.StripPrefix("/ui/", h)}
}

type idpUI struct {
	h             http.Handler
	prefixHandler http.Handler
}

func (s *idpUI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if "/favicon.ico" == req.URL.Path {
		s.h.ServeHTTP(w, req)
		return
	}
	// Encourage caching of UI
	w.Header().Add("Cache-Control", "public, max-age=31536000")
	s.prefixHandler.ServeHTTP(w, req)
}
