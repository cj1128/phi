SHELL := /bin/bash

install:
	go install
.PHONY: install

test:
	go test -v -cover
.PHONY: test

lint:
	gometalinter
