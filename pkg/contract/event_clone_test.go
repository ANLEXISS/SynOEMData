package contract

import (
	"reflect"
	"testing"
)

type cloneTestPayload struct {
	Labels []string
	Values map[string][]int
}

func TestEventCloneDeepCopiesNestedPayloadAndCycles(t *testing.T) {
	nested := map[string]any{"labels": []any{"initial"}}
	payload := &cloneTestPayload{Labels: []string{"initial"}, Values: map[string][]int{"values": {1}}}
	body := map[string]any{"nested": nested, "typed": payload}
	body["self"] = body
	event := Event{ID: "event", Payload: body}
	cloned := event.Clone()

	nested["labels"].([]any)[0] = "input change"
	payload.Labels[0] = "input change"
	payload.Values["values"][0] = 9
	if cloned.Payload["nested"].(map[string]any)["labels"].([]any)[0] != "initial" {
		t.Fatal("nested map/slice still aliases the original")
	}
	clonedPayload := cloned.Payload["typed"].(*cloneTestPayload)
	if clonedPayload.Labels[0] != "initial" || clonedPayload.Values["values"][0] != 1 {
		t.Fatal("typed payload still aliases the original")
	}
	self := cloned.Payload["self"].(map[string]any)
	if reflect.ValueOf(self).Pointer() != reflect.ValueOf(cloned.Payload).Pointer() {
		t.Fatal("cycle was not preserved inside the detached payload")
	}
}
