image=quay.io/coreos/openldap:2.4.44
image_file=assets/openldap_2_4_44.docker

aci_file=quay.io-coreos-openldap-2.4.44.aci

$(shell mkdir -p assets)

user=$(shell id -u -n)
group=$(shell id -g -n)

.PHONY: build
build: $(image_file)

$(image_file): assets/openldap-2.4.44.tgz Dockerfile scripts/entrypoint.sh
	sudo docker build -t  $(image) .
	sudo docker save -o $(image_file) $(image)
	sudo chown $(user):$(group) $(image_file)

assets/openldap-2.4.44.tgz: scripts/download.sh
	./scripts/download.sh

$(aci_file): $(image_file)
	docker2aci $(image_file)
	mv $(aci_file) assets/$(aci_file)

.PHONY: import-aci
import-aci: $(aci_file)
	sudo rkt fetch --insecure-options=image ./assets/quay.io-coreos-openldap-2.4.44.aci

clean:
	rm -rf assets/*

.PHONY: push
push:
	sudo docker push quay.io/coreos/openldap:2.4.44
