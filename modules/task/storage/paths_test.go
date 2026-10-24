package storage

import "testing"

func TestArchiveAndTraversalAreNotPublicImages(t *testing.T) {
	for _, key := range []string{
		"tasks/seifs/archive/solution.cpp", "tasks/seifs/archive/tests/1.in",
		"tasks/seifs/images/../archive/solution.cpp", "tasks/../images/a.png",
		"testfiles/hash.zst", "tasks/seifs/images",
	} {
		if PublicImage(key) {
			t.Errorf("private or malformed path allowed: %s", key)
		}
	}
	if !PublicImage(Image("seifs", Digest([]byte("image")), "1.png")) {
		t.Fatal("image rejected")
	}
}
