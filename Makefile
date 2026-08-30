default: test

templates/*_templ.go: templates/*.templ
	go tool templ generate templates/*.templ

templ: templates/*_templ.go

tmp/main: **/*.go templates/*_templ.go
	go build -o tmp/main

build: tmp/main

generate: templ

deploy: generate
	git push dokku main:master

browse:
	open "https://notes.sune.one/"

run: generate
	go run .

test: generate
	go test ./...

test-update: generate
	UPDATE_SNAPS=true go test ./...

dev:
	NOTES_VIEWER_SKIP_VERIFICATION=true air

dev-with-login:
	NOTES_VIEWER_SKIP_VERIFICATION=false air

cover: generate
	go test ./... -coverprofile=cover.prof
	@covreport
	@echo
	@echo "Written file://$$PWD/cover.html"

clean:
	rm -rf tmp templates/*_templ.go
