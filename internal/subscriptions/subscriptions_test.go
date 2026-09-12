package subscriptions

import "testing"

func TestMaxSubscriptionsPerIPDay(t *testing.T) {
	if MaxSubscriptionsPerIPDay != 10 {
		t.Fatalf("MaxSubscriptionsPerIPDay = %d, want 10", MaxSubscriptionsPerIPDay)
	}
}
