.PHONY: cfssl ca req clean

CFSSL	= @env PATH=$(GOPATH)/bin:$(PATH) cfssl
JSON	= env PATH=$(GOPATH)/bin:$(PATH) cfssljson

all: cfssl ca req

cfssl:
	go get -u -tags nopkcs11 github.com/cloudflare/cfssl/cmd/cfssl
	go get -u github.com/cloudflare/cfssl/cmd/cfssljson
	go get -u github.com/mattn/goreman

ca:
	mkdir -p certs
	$(CFSSL) gencert -initca config/ca-csr.json | $(JSON) -bare certs/ca

req:
	$(CFSSL) gencert \
	  -ca certs/ca.pem \
	  -ca-key certs/ca-key.pem \
	  -config config/ca-config.json \
	  config/req-idp.json | $(JSON) -bare certs/idp
	$(CFSSL) gencert \
	  -ca certs/ca.pem \
	  -ca-key certs/ca-key.pem \
	  -config config/ca-config.json \
	  config/req-sp.json | $(JSON) -bare certs/sp
	$(CFSSL) gencert \
	  -ca certs/ca.pem \
		-profile client \
	  -ca-key certs/ca-key.pem \
	  -config config/ca-config.json \
	  config/req-user.json | $(JSON) -bare certs/user
	openssl pkcs12 -export -out certs/user.p12 -inkey certs/user-key.pem -in certs/user.pem -passout pass:"password" -CAfile certs/ca.pem

clean:
	rm -rf certs

