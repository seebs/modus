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
	okay=true ; \
	for profile in cpu heap alloc; do \
		path=$${profile}-profile; \
		if [ -f $$path.dat ] ; then \
			rm $$path.pdf; \
			go tool pprof --pdf $$path.dat > $$path.pdf || okay=false ; \
		fi; \
	done; $$okay
