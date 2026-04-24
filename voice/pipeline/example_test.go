package pipeline_test

import (
	"context"
	"fmt"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	"github.com/mahdi-awadi/gopkg/voice/pipeline/fake"
)

func Example() {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.Script(pipeline.EventAssistantText{Text: "hello", Final: true})
	ll.CloseEvents()
	tr.CloseInbound()

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	for _, e := range rec.Events() {
		if a, ok := e.(fake.RecAssistantText); ok {
			fmt.Println("assistant said:", a.Text)
		}
	}
	// Output:
	// assistant said: hello
}

type spellcheckObs struct {
	pipeline.NoopObserver
}

func (spellcheckObs) OnAssistantText(_ context.Context, _ pipeline.Session, text string, _ bool) {
	fmt.Println("spellcheck:", text)
}

func ExampleNoopObserver_embed() {
	var _ pipeline.Observer = spellcheckObs{}
	// See the struct above — embedding NoopObserver and overriding just one method.
	fmt.Println("ok")
	// Output:
	// ok
}

func ExampleMulti() {
	a := fake.NewRecorder()
	b := fake.NewRecorder()
	multi := pipeline.Multi(a, b)
	multi.OnSessionStart(context.Background(), pipeline.Session{})
	fmt.Println(len(a.Events()), len(b.Events()))
	// Output:
	// 1 1
}
