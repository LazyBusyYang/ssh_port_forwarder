package lb

import (
	"testing"

	"ssh-port-forwarder/internal/model"
)

func TestWeightedStrategyPrefersHigherWeightWhenEmpty(t *testing.T) {
	strategy := &WeightedStrategy{}
	hosts := []model.SSHHost{
		{ID: 1, Name: "a", Weight: 50},
		{ID: 2, Name: "b", Weight: 100},
	}

	selected := strategy.SelectHost(hosts, map[uint64]int64{})
	if selected == nil {
		t.Fatal("expected selected host")
	}
	if selected.ID != 2 {
		t.Fatalf("expected host 2, got %d", selected.ID)
	}
}

func TestWeightedStrategyBalancesByPredictedNormalizedLoad(t *testing.T) {
	strategy := &WeightedStrategy{}
	hosts := []model.SSHHost{
		{ID: 1, Name: "a", Weight: 50},
		{ID: 2, Name: "b", Weight: 100},
	}
	counts := map[uint64]int64{}

	for i := 0; i < 6; i++ {
		selected := strategy.SelectHost(hosts, counts)
		if selected == nil {
			t.Fatal("expected selected host")
		}
		counts[selected.ID]++
	}

	if counts[1] != 2 || counts[2] != 4 {
		t.Fatalf("expected 50/100 weights to produce 2/4 rule split, got host1=%d host2=%d", counts[1], counts[2])
	}
}

func TestWeightedStrategyUsesLoadNotOnlyWeight(t *testing.T) {
	strategy := &WeightedStrategy{}
	hosts := []model.SSHHost{
		{ID: 1, Name: "a", Weight: 50},
		{ID: 2, Name: "b", Weight: 100},
	}
	counts := map[uint64]int64{
		1: 1,
		2: 4,
	}

	selected := strategy.SelectHost(hosts, counts)
	if selected == nil {
		t.Fatal("expected selected host")
	}
	if selected.ID != 1 {
		t.Fatalf("expected lower-weight host 1 due to lower normalized load, got %d", selected.ID)
	}
}

func TestWeightedStrategyTieBreakers(t *testing.T) {
	t.Run("fewer current rules wins equal predicted score", func(t *testing.T) {
		strategy := &WeightedStrategy{}
		hosts := []model.SSHHost{
			{ID: 1, Name: "a", Weight: 50},
			{ID: 2, Name: "b", Weight: 100},
		}
		counts := map[uint64]int64{
			1: 1,
			2: 3,
		}

		selected := strategy.SelectHost(hosts, counts)
		if selected == nil {
			t.Fatal("expected selected host")
		}
		if selected.ID != 1 {
			t.Fatalf("expected host 1 to win equal predicted score by fewer current rules, got %d", selected.ID)
		}
	})

	t.Run("lower host id wins final tie", func(t *testing.T) {
		strategy := &WeightedStrategy{}
		hosts := []model.SSHHost{
			{ID: 2, Name: "b", Weight: 100},
			{ID: 1, Name: "a", Weight: 100},
		}
		counts := map[uint64]int64{
			1: 2,
			2: 2,
		}

		selected := strategy.SelectHost(hosts, counts)
		if selected == nil {
			t.Fatal("expected selected host")
		}
		if selected.ID != 1 {
			t.Fatalf("expected host 1 to win final tie by lower id, got %d", selected.ID)
		}
	})
}

func TestWeightedStrategyNormalizesNonPositiveWeight(t *testing.T) {
	strategy := &WeightedStrategy{}
	hosts := []model.SSHHost{
		{ID: 1, Name: "a", Weight: 0},
		{ID: 2, Name: "b", Weight: 2},
	}

	selected := strategy.SelectHost(hosts, map[uint64]int64{})
	if selected == nil {
		t.Fatal("expected selected host")
	}
	if selected.ID != 2 {
		t.Fatalf("expected positive weight host 2, got %d", selected.ID)
	}
}
