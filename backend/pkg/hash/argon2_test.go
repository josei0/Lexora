package hash

import "testing"

func TestPasswordRoundtrip(t *testing.T) {
	h, err := Password("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify("s3cret-pw", h); err != nil {
		t.Fatalf("verify correct pw: %v", err)
	}
	if err := Verify("wrong-pw", h); err == nil {
		t.Fatal("wrong pw should fail")
	}
}
