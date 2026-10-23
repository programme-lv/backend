package img

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"golang.org/x/sync/singleflight"
)

// Variant is a named illustration derivative. Only these exist; clients cannot invent sizes.
type Variant string

const (
	VariantList Variant = "list" // fit inside 160×160
	VariantView Variant = "view" // fit inside 256×256
	VariantFull Variant = "full" // original pixels as WebP
)

const illustrationDir = "illustrations"

// derivableRoots are the public asset prefixes that may be served as derived
// WebP variants. Illustrations are flat; statement images live under a task
// directory, and older installs use the task-md-images/ and task/ prefixes.
var derivableRoots = []struct {
	prefix string
	flat   bool
}{
	{illustrationDir + "/", true},
	{"md-images/", false},
	{"task-md-images/", false},
	{"task/", false},
}

const derivedCacheControl = "public, max-age=31536000, immutable"

var (
	deriveGroup singleflight.Group
	encodeSem   = make(chan struct{}, 2)
)

// ObjectStore is the filestore surface used to cache derived WebPs.
type ObjectStore interface {
	Upload(content []byte, key string, mediaType string) (string, error)
	Download(key string) ([]byte, error)
	Exists(key string) (bool, error)
}

// Variants is the closed allowlist of illustration derivatives.
func Variants() []Variant {
	return []Variant{VariantList, VariantView, VariantFull}
}

// MaxEdge is the longest side of the variant box. Zero means do not resize.
func (v Variant) MaxEdge() int {
	switch v {
	case VariantList:
		return 160
	case VariantView:
		return 256
	default:
		return 0
	}
}

func (v Variant) valid() bool {
	return v == VariantList || v == VariantView || v == VariantFull
}

// DerivedCacheControl is the Cache-Control value for derived image objects.
func DerivedCacheControl() string {
	return derivedCacheControl
}

// DerivedKey is the filestore key for a variant of originalKey.
func DerivedKey(originalKey string, v Variant) string {
	return originalKey + "." + string(v) + ".webp"
}

// ServeKey is the object that should be served for a variant.
// A WebP original is the full variant; no extra file is written.
func ServeKey(originalKey string, v Variant) string {
	if v == VariantFull && isWebPKey(originalKey) {
		return originalKey
	}
	return DerivedKey(originalKey, v)
}

// IsDerivable reports whether originalKey can be transcoded to WebP. Vector and
// unsupported formats must keep their original URL.
func IsDerivable(originalKey string) bool {
	switch strings.ToLower(path.Ext(originalKey)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}

func derivableAsset(key string) bool {
	for _, root := range derivableRoots {
		rest, ok := strings.CutPrefix(key, root.prefix)
		if !ok {
			continue
		}
		if rest == "" {
			return false
		}
		return !root.flat || !strings.Contains(rest, "/")
	}
	return false
}

// ParseDerivedKey reports the original asset key and variant if key is an
// allowlisted derivative. Unknown suffixes are not parsed (caller 404s).
func ParseDerivedKey(key string) (originalKey string, variant Variant, ok bool) {
	for _, v := range Variants() {
		suffix := "." + string(v) + ".webp"
		if !strings.HasSuffix(key, suffix) {
			continue
		}
		originalKey = strings.TrimSuffix(key, suffix)
		if !derivableAsset(originalKey) || !IsDerivable(originalKey) {
			return "", "", false
		}
		return originalKey, v, true
	}
	return "", "", false
}

// EnsureVariant writes the derived WebP if missing and returns the key to serve.
func EnsureVariant(ctx context.Context, store ObjectStore, originalKey string, variant Variant) (string, error) {
	if !variant.valid() {
		return "", fmt.Errorf("unknown image variant %q", variant)
	}
	serveKey := ServeKey(originalKey, variant)
	if serveKey == originalKey {
		return originalKey, nil
	}

	exists, err := store.Exists(serveKey)
	if err != nil {
		return "", err
	}
	if exists {
		return serveKey, nil
	}

	origExists, err := store.Exists(originalKey)
	if err != nil {
		return "", err
	}
	if !origExists {
		return "", os.ErrNotExist
	}

	v, err, _ := deriveGroup.Do(serveKey, func() (any, error) {
		exists, err := store.Exists(serveKey)
		if err != nil {
			return nil, err
		}
		if exists {
			return serveKey, nil
		}

		select {
		case encodeSem <- struct{}{}:
			defer func() { <-encodeSem }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		raw, err := store.Download(originalKey)
		if err != nil {
			return nil, err
		}
		src, err := decodeRaster(raw)
		if err != nil {
			return nil, err
		}
		webp, err := rasterWebP(src, variant.MaxEdge())
		if err != nil {
			return nil, err
		}
		if _, err := store.Upload(webp, serveKey, "image/webp"); err != nil {
			return nil, err
		}
		return serveKey, nil
	})
	if err != nil {
		return "", err
	}
	key, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("unexpected derive result type %T", v)
	}
	return key, nil
}

func isWebPKey(key string) bool {
	return strings.EqualFold(path.Ext(key), ".webp")
}
