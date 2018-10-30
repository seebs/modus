run: modus
	./modus

modus: *.go
	go build -race

lint:
	go fmt
	go vet
	golint

full-lint: lint
	staticcheck

ppdf:
	-rm cpu-profile.pdf heap-profile.pdf alloc-profile.pdf
	-[ -f cpu-profile.dat ] && go tool pprof --pdf cpu-profile.dat > cpu-profile.pdf
	-[ -f heap-profile.dat ] && go tool pprof --pdf heap-profile.dat > heap-profile.pdf
	-[ -f alloc-profile.dat ] && go tool pprof --pdf alloc-profile.dat > alloc-profile.pdf
