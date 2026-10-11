package statestore

import (
	"fmt"
	"testing"

	"synora/pkg/contract"
)

func BenchmarkStoreReadAPIs(b *testing.B) {
	store := NewUniversalStore[testSnapshot, testSnapshotV3, testDecision, testDecisionV3, testAction](testSnapshot{Value: "initial"})
	facts := make([]string, 32)
	for i := range facts {
		facts[i] = fmt.Sprintf("fact-%02d", i)
	}
	metadata := map[string][]string{"labels": append([]string(nil), facts[:16]...)}
	for i := 0; i < 128; i++ {
		commit := testCommit{
			Event:      contract.Event{ID: fmt.Sprintf("bench-%03d", i), Type: "test", Payload: map[string]any{"facts": append([]string(nil), facts...)}},
			Snapshot:   testSnapshot{Value: "snapshot", Facts: append([]string(nil), facts...), Metadata: metadata},
			SnapshotV3: &testSnapshotV3{Value: "snapshot-v3", Facts: append([]string(nil), facts...), Metadata: metadata},
			Decision:   testDecision{Name: "decision", Facts: append([]string(nil), facts...), Metadata: metadata},
			DecisionV3: &testDecisionV3{Name: "decision-v3", Facts: append([]string(nil), facts...), Metadata: metadata},
			Action:     &testAction{RequestID: fmt.Sprintf("action-%03d", i), Steps: append([]string(nil), facts...), Metadata: metadata},
		}
		if _, err := store.Commit(commit); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	benchmarks := []struct {
		name string
		run  func() any
	}{
		{"Snapshot", func() any { return store.Snapshot() }},
		{"SnapshotV3", func() any { v, _ := store.SnapshotV3(); return v }},
		{"Journal", func() any { return store.Journal() }},
		{"Decisions", func() any { return store.Decisions() }},
		{"ActionOutbox", func() any { return store.ActionOutbox() }},
		{"History", func() any { h, _ := store.History(); return h }},
	}
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = benchmark.run()
			}
		})
	}
}
