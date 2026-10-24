package storage

import "testing"

func TestArchiveAndTraversalAreNotPublicImages(t *testing.T) {
	for _, key := range []string{
		"assets/tasks/seifs/archive/solution.cpp", "assets/tasks/seifs/archive/tests/1.in",
		"assets/tasks/seifs/images/../archive/solution.cpp", "assets/tasks/../images/a.png",
		"testfiles/hash.zst", "assets/tasks/seifs/images",
		"archive/tasks/seifs/solution.cpp", "cache/images/hash/full.webp",
	} {
		if PublicImage(key) {
			t.Errorf("private or malformed path allowed: %s", key)
		}
	}
	if !PublicImage(Image("seifs", Digest([]byte("image")), "1.png")) {
		t.Fatal("image rejected")
	}
}
