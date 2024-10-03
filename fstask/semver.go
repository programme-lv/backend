package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

type Version struct {
	major int
	minor int
	patch int
}

func getSpecVersionFromToml(tomlBytes []byte) (Version, error) {
	var specVersStruct struct {
		Specification string `toml:"specification"`
	}

	err := toml.Unmarshal(tomlBytes, &specVersStruct)
	if err != nil {
		return Version{}, fmt.Errorf("failed to unmarshal the specification version: %w", err)
	}

	return semanticVersionFromStr(specVersStruct.Specification)
}

func (sv *Version) GreaterOrEqThan(other Version) bool {
	if sv.major > other.major {
		return true
	}
	if sv.major < other.major {
		return false
	}

	if sv.minor > other.minor {
		return true
	}
	if sv.minor < other.minor {
		return false
	}

	if sv.patch >= other.patch {
		return true
	}

	return false
}

func (sv *Version) LessThan(other Version) bool {
	return !sv.GreaterOrEqThan(other)
}

func (sv *Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.major, sv.minor, sv.patch)
}

func semanticVersionFromStr(s string) (Version, error) {
	var sv Version
	if s[0] == 'v' {
		_, err := fmt.Sscanf(s, "v%d.%d.%d", &sv.major, &sv.minor, &sv.patch)
		if err != nil {
			return Version{}, fmt.Errorf("error parsing semver: %w", err)
		}

		return sv, nil
	} else {
		_, err := fmt.Sscanf(s, "%d.%d.%d", &sv.major, &sv.minor, &sv.patch)
		if err != nil {
			return Version{}, fmt.Errorf("error parsing semver: %w", err)
		}

		return sv, nil
	}
}
