SHELL := /bin/bash

install:
	go install
.PHONY: install

test:
	go test -v
.PHONY: test
