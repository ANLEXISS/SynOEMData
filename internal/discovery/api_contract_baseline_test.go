package discovery

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestH1ResidentGalleryResponseMatchesAPIContractFixture(t *testing.T) {
	got, err := json.Marshal(ResidentGalleryView{ResidentRef: "res_0123456789abcdef0123456789abcdef", GalleryStatus: "active", Generation: 3, PolicyVersion: "resident-gallery/v1"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/resident-gallery-response-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	got, _ = json.Marshal(gotValue)
	want, _ = json.Marshal(wantValue)
	if !bytes.Equal(got, want) {
		t.Fatalf("API response fixture changed:\n got %s\nwant %s", got, want)
	}
}
