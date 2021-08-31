ifeq ($(OS),Windows_NT)
    BROWSER = start
else
	UNAME := $(shell uname -s)
	PARENTDIR := $(shell dirname "$$PWD")
	JSDIR = $(PARENTDIR)/wasm/js
	ifeq ($(UNAME), Linux)
		BROWSER = xdg-open
		COMMAND = gnome-terminal --tab --command="bash -c 'cd $(PARENTDIR) && go run main.go; $(SHELL)'"
	endif
	ifeq ($(UNAME), Darwin)
		BROWSER = open
		COMMAND = osascript -e 'tell app "Terminal" to do script "cd $(PARENTDIR) && go run main.go"'
	endif
endif

.PHONY: all clean serve

all: wasm terminal serve

wasm:
	[ -d $(JSDIR) ] || mkdir -p $(JSDIR)
	cp -f "$$(go env GOROOT)/misc/wasm/wasm_exec.js" $(JSDIR)
	GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go

serve:
	$(BROWSER) 'http://localhost:5000'
	go run $(PARENTDIR)/server/main.go

terminal:
	$(COMMAND)
	@sleep 1.5

clean:
	rm -f *.wasm

debug:
	@echo $(UNAME)