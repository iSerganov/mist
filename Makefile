BIN     ?= bin/mist
KEYDIR  ?= keys
NAME    ?= mist
INPUT   ?=
DATA    ?=
OUTPUT  ?=
CODEC   ?=
KEY     ?=
TIMEOUT ?= 0

# libav's own verbosity during a test run, on top of Go's -v:
# quiet | error | warning | info | verbose | debug | trace
LOG ?= info

test:
	GOTRACEBACK=all MIST_AV_LOG=$(LOG) \
		go test -v -race -count=1 -cover -timeout 20m ./...

# Same coverage, without the per-test firehose.
test-quiet:
	go test -race -count=1 ./...

lint:
	golangci-lint run --timeout=5m

build:
	@mkdir -p $(dir $(BIN))
	go build -o $(BIN) ./cmd/mist

# make embed INPUT=song.mp3 DATA="hello" [OUTPUT=out.flac] [CODEC=alac] [KEY=keys/mist.pub]
# OUTPUT's extension picks the format, as it does for ffmpeg; see make formats.
embed: build
	@test -n "$(INPUT)" || { echo "usage: make embed INPUT=<file|url> DATA=<text> [OUTPUT=..] [CODEC=..] [KEY=..]"; exit 2; }
	@test -n "$(DATA)"  || { echo "usage: make embed INPUT=<file|url> DATA=<text> [OUTPUT=..] [CODEC=..] [KEY=..]"; exit 2; }
	./$(BIN) embed --input "$(INPUT)" --data "$(DATA)" \
		$(if $(OUTPUT),--output "$(OUTPUT)") $(if $(CODEC),--out-codec "$(CODEC)") \
		$(if $(KEY),--key "$(KEY)")

# Every output format the installed FFmpeg can write.
formats: build
	./$(BIN) formats

# make estimate INPUT=song.mp3 [OUTPUT=out.flac] [CODEC=alac]
estimate: build
	@test -n "$(INPUT)" || { echo "usage: make estimate INPUT=<file|url> [OUTPUT=..] [CODEC=..]"; exit 2; }
	./$(BIN) estimate --input "$(INPUT)" \
		$(if $(OUTPUT),--output "$(OUTPUT)") $(if $(CODEC),--out-codec "$(CODEC)")

# make catch INPUT=out.ogg KEY=keys/mist.key [TIMEOUT=30s]
catch: build
	@test -n "$(INPUT)" || { echo "usage: make catch INPUT=<file|url> KEY=<private key> [TIMEOUT=30s]"; exit 2; }
	@test -n "$(KEY)"   || { echo "usage: make catch INPUT=<file|url> KEY=<private key> [TIMEOUT=30s]"; exit 2; }
	./$(BIN) catch --input "$(INPUT)" --key "$(KEY)" --timeout "$(TIMEOUT)"

# X25519 keypair via OpenSSL, written in the hex form the CLI reads.
# OpenSSL emits PKCS#8/SPKI DER, whose final 32 bytes are the raw key.
keys:
	@command -v openssl >/dev/null || { echo "openssl not found"; exit 2; }
	@mkdir -p $(KEYDIR)
	@umask 077 && openssl genpkey -algorithm X25519 -out $(KEYDIR)/$(NAME).pem
	@openssl pkey -in $(KEYDIR)/$(NAME).pem -outform DER \
		| tail -c 32 | xxd -p -c 32 > $(KEYDIR)/$(NAME).key
	@openssl pkey -in $(KEYDIR)/$(NAME).pem -pubout -outform DER \
		| tail -c 32 | xxd -p -c 32 > $(KEYDIR)/$(NAME).pub
	@rm -f $(KEYDIR)/$(NAME).pem
	@chmod 600 $(KEYDIR)/$(NAME).key
	@chmod 644 $(KEYDIR)/$(NAME).pub
	@echo "public key  $(KEYDIR)/$(NAME).pub  (give to senders)"
	@echo "private key $(KEYDIR)/$(NAME).key  (keep secret)"

clean:
	rm -rf bin

.PHONY: test test-quiet lint build embed catch formats keys clean
