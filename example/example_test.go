package example

import (
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
	fmt.Println(emitter != nil)
	// Output: true
}

// A finite file ends by itself; a live stream runs until the context is
// cancelled. The caller tells the two apart with ctx.Err() afterwards.
func Example_catcher() {
	_, priv, err := mist.GenerateKeyPair()
	if err != nil {
		log.Fatal(err)
	}
	catcher, err := mist.NewCatcher(priv)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := catcher.Listen(ctx, "/path/to/recording.ogg")
	if err != nil {
		return // unreadable source or malformed key
	}
	for result := range ch {
		fmt.Println("frame", result.FrameIdx, string(result.Payload.Data))
	}
	if ctx.Err() != nil {
		fmt.Println("stopped by caller")
	}
}
