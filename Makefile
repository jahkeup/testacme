GOIMPORTS = goimports -local github.com/jahkeup/testacme

build test:
	$(MAKE) -f ci/Makefile test

fmt goimports:
	$(GOIMPORTS) -w .
