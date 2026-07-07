package embedding

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/panjf2000/ants/v2"
)

type batchTestEmbedder struct {
	EmbedderPooler
	active int32
	max    int32
}

func (e *batchTestEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1}, nil
}

func (e *batchTestEmbedder) BatchEmbed(_ context.Context, texts []string) ([][]float32, error) {
	active := atomic.AddInt32(&e.active, 1)
	for {
		max := atomic.LoadInt32(&e.max)
		if active <= max || atomic.CompareAndSwapInt32(&e.max, max, active) {
			break
		}
	}
	defer atomic.AddInt32(&e.active, -1)

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{float32(i)}
	}
	return out, nil
}

func (e *batchTestEmbedder) GetModelName() string { return "test" }
func (e *batchTestEmbedder) GetDimensions() int   { return 1 }
func (e *batchTestEmbedder) GetModelID() string   { return "test" }

func TestBatchEmbedWithPoolUsesPoolCapAsGlobalLimit(t *testing.T) {
	t.Setenv("BATCH_EMBED_SIZE", "1")
	pool, err := ants.NewPool(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Release()

	pooler := NewBatchEmbedder(pool)
	model := &batchTestEmbedder{EmbedderPooler: pooler}

	done := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := pooler.BatchEmbedWithPool(context.Background(), model, []string{"a", "b", "c"})
			done <- err
		}()
	}

	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&model.max); got > 2 {
		t.Fatalf("max concurrent BatchEmbed calls = %d, want <= 2", got)
	}
}
