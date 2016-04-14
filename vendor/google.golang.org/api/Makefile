all:
	go install google.golang.org/api/googleapi
	go install google.golang.org/api/google-api-go-generator
	$(GOPATH)/bin/google-api-go-generator -cache=false -install -api=*

cached:
	go install google.golang.org/api/googleapi
	go install google.golang.org/api/google-api-go-generator
	$(GOPATH)/bin/google-api-go-generator -cache=true -install -api=*
