// cryptopasta - basic cryptography examples
//
// Written in 2016 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

// Provides a recommended TLS configuration.
package cryptopasta

import (
	"log"
	"net/http"
)

func ExampleTLSServer() {
	// Get recommended basic configuration
	config := DefaultTLSConfig()

	// Serve up some HTTP
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello, world\n"))
	})

	server := &http.Server{
		Addr:      ":8080",
		TLSConfig: config,
	}

	err := server.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		log.Fatal(err)
	}
}
