.PHONY: \
	install-web build-site build-all lint-web test-web cover-web ci-web

install-web:
	cd web/landing && npm ci

build-site: install-web
	cd web/landing && npm run build

build-all: build build-site

lint-web:
	cd web/landing && npm run lint

test-web:
	cd web/landing && npm run test

cover-web:
	cd web/landing && npm run cover

ci-web: install-web lint-web test-web cover-web build-site
