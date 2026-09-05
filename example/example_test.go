package example

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/iSerganov/mist"
)

func Example_generateKeyPair() {
	pub, priv, err := mist.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(pub), len(priv))
	// Output: 32 32
}

func Example_emitter() {
	pub, _, err := mist.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	emitter, err := mist.NewEmitter(pub)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	out, err := emitter.EmbedReader(ctx, bytes.NewReader(nil), mist.Text("hello"))
	if out != nil {
		defer func() { _ = out.Close() }()
	}
	_ = err
}

func Example_catcher() {
	_, priv, err := mist.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	catcher, err := mist.NewCatcher(priv)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ch, err := catcher.Listen(ctx, "/path/to/recording.ogg")
	if err != nil {
		_ = err
		return
	}
	for range ch {
	}
}
