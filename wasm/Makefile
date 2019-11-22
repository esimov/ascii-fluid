ifeq ($(OS),Windows_NT)
    BROWSER = start
else
	UNAME := $(shell uname -s)
	ifeq ($(UNAME), Linux)
		BROWSER = xdg-open
		COMMAND = gnome-terminal --tab --command="bash -c 'go run ../main.go; $(SHELL)'"
	endif
	ifeq ($(UNAME), Darwin)
		BROWSER = open
		COMMAND = osascript -e 'tell app "Terminal" to do script "go run ../main.go"'
	endif
endif

.PHONY: all clean serve

all: wasm terminal serve

wasm:
	GOOS=js GOARCH=wasm go build -o lib.wasm wasm.go

serve:
	$(BROWSER) 'http://localhost:5000'
	serve

terminal:
	$(COMMAND)

clean:
	rm -f *.wasm

debug:
	@echo $(UNAME)