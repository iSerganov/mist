BIN     ?= bin/mist
KEYDIR  ?= keys
NAME    ?= mist
INPUT   ?=
DATA    ?=
OUTPUT  ?=
CODEC   ?=
KEY     ?=
TIMEOUT ?= 0
CORPUS  ?=
CORPUS_MANIFEST ?=
CORPUS_NAME ?=
CORPUS_OUT ?=
FORMATS ?=
BASELINE ?=
HARNESS_OUT ?= harness-out
JOBS ?=
PUBLIC_KEY_HEX ?=
FFMPEG ?= ffmpeg
VISQOL ?= visqol
PEAQ ?= peaq
GIT ?= git
PKG_CONFIG ?= pkg-config
PYTHON ?= python3
CNN_WARDEN ?= tools/cnn_warden/train.py
CNN_EXPORT ?= $(HARNESS_OUT)/export
CNN_OUT ?= $(HARNESS_OUT)/cnn.json

# libav's own verbosity during a test run, on top of Go's -v:
# quiet | error | warning | info | verbose | debug | trace
LOG ?= info

test:
	GOTRACEBACK=all MIST_AV_LOG=$(LOG) \
		go test -v -race -count=1 -cover -timeout 20m ./...

# Same coverage, without the per-test firehose.
test-quiet:
	go test -race -count=1 ./...

# make harness [CORPUS=dir] [CORPUS_MANIFEST=testdata/harness/corpus.example.json]
#   [CORPUS_NAME=public-name] [FORMATS=ogg,flac|all] [BASELINE=path/report.json]
#   [HARNESS_OUT=harness-out] [JOBS=4] [MAX_SECONDS=300] [MERGE=a/report.json,b/report.json]
#   [FFMPEG=ffmpeg] [VISQOL=visqol] [PEAQ=peaq] [GIT=git] [PKG_CONFIG=pkg-config]
#   [PUBLIC_KEY_HEX=...] [CNN_OUT=$(HARNESS_OUT)/cnn.json]
# Writes report.md/json, manifest.json and scores.json under $(HARNESS_OUT).
# Every executable and filesystem path is an override; do not hard-code local
# audio directories. Prefer CORPUS_MANIFEST so reports use public ids.
# make corpus CORPUS_OUT=/path/outside/this/repo
# Writes a generated PCM corpus and a sealed holdout. Audio is not committed.
corpus:
	@test -n "$(CORPUS_OUT)" || { echo "set CORPUS_OUT to a directory outside the repository" >&2; exit 1; }
	MIST_CORPUS_OUT="$(CORPUS_OUT)" MIST_FFMPEG="$(FFMPEG)" \
		go test -tags harness -run 'TestHarnessSuite/TestWriteGeneratedCorpus' -count=1 -timeout 10m -v .

harness:
	MIST_CORPUS="$(CORPUS)" MIST_HARNESS_FORMATS="$(FORMATS)" \
		MIST_HARNESS_CORPUS_MANIFEST="$(CORPUS_MANIFEST)" MIST_HARNESS_CORPUS_NAME="$(CORPUS_NAME)" \
		MIST_HARNESS_BASELINE="$(BASELINE)" MIST_HARNESS_OUT="$(HARNESS_OUT)" MIST_HARNESS_JOBS="$(JOBS)" MIST_HARNESS_MAX_SECONDS="$(MAX_SECONDS)" MIST_HARNESS_MERGE="$(MERGE)" \
		MIST_HARNESS_PUBLIC_KEY_HEX="$(PUBLIC_KEY_HEX)" MIST_HARNESS_CNN="$(CNN_OUT)" MIST_FFMPEG="$(FFMPEG)" MIST_VISQOL="$(VISQOL)" MIST_PEAQ="$(PEAQ)" MIST_GIT="$(GIT)" MIST_PKG_CONFIG="$(PKG_CONFIG)" \
		go test -tags harness -run TestHarnessSuite -count=1 -timeout 0 -v .

# make cnn-warden [CORPUS=dir] [CORPUS_MANIFEST=...] [FORMATS=ogg,flac|all]
#   [PYTHON=python3] [CNN_WARDEN=tools/cnn_warden/train.py]
#   [CNN_EXPORT=$(HARNESS_OUT)/export] [CNN_OUT=$(HARNESS_OUT)/cnn.json]
# Harness run with the export on, a CNN warden trained on it, then the harness
# again so the report shows the CNN's result. All paths are overrides.
cnn-warden:
	MIST_HARNESS_EXPORT="$(CNN_EXPORT)" $(MAKE) harness
	"$(PYTHON)" "$(CNN_WARDEN)" "$(CNN_EXPORT)" --out "$(CNN_OUT)"
	$(MAKE) harness

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

.PHONY: test test-quiet harness cnn-warden lint build embed catch formats keys clean
