package main

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Version struct {
	Major uint
	Minor uint
	Patch uint
}

const (
	versionLimit uint = 10000
)

func NewDummyVersion() Version {
	return Version{versionLimit - 1, versionLimit - 1, versionLimit - 1}
}

var versionRegex = regexp.MustCompile(`^v?(?P<major>[0-9]+)\.(?P<minor>[0-9]+)\.(?P<patch>[0-9]+)$`)

func ParseVersion(v string) (*Version, error) {
	matches := versionRegex.FindStringSubmatch(v)
	if matches == nil {
		return nil, fmt.Errorf("invalid version string: %s", v)
	}
	major, err := strconv.ParseUint(matches[versionRegex.SubexpIndex("major")], 10, 32)
	if err != nil {
		return nil, err
	}
	minar, err := strconv.ParseUint(matches[versionRegex.SubexpIndex("minor")], 10, 32)
	if err != nil {
		return nil, err
	}
	patch, err := strconv.ParseUint(matches[versionRegex.SubexpIndex("patch")], 10, 32)
	if err != nil {
		return nil, err
	}

	version := Version{uint(major), uint(minar), uint(patch)}
	dummy := NewDummyVersion()
	if version.Compare(&dummy) <= 0 {
		return &version, nil
	}
	return &dummy, nil
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v *Version) Compare(o *Version) int {
	left := []uint{v.Major, v.Minor, v.Patch}
	right := []uint{o.Major, o.Minor, o.Patch}
	return slices.Compare(left, right)
}

type Feature string

const (
	FeatureSubshell = Feature("subshell")
)

func (f Feature) String() string { return string(f) }

func (f Feature) RequiredVersion() (*Version, *Version) {
	switch f {
	case FeatureSubshell:
		return &Version{0, 38, 0}, &Version{versionLimit, 0, 0}
	}
	return nil, nil
}

func (f Feature) Message() string {
	v1, _ := f.RequiredVersion()
	if v1 != nil {
		sb := strings.Builder{}
		sb.WriteString("require arsh version ")
		sb.WriteString(v1.String())
		sb.WriteString(" or later")
		return sb.String()
	}
	return ""
}

type FeatureSet struct {
	set map[Feature]struct{}
}

var features = []Feature{
	FeatureSubshell,
}

func NewFeatureSetFromVersion(v Version) FeatureSet {
	featureSet := FeatureSet{map[Feature]struct{}{}}
	for _, feature := range features {
		v1, v2 := feature.RequiredVersion()
		if v.Compare(v1) >= 0 && v.Compare(v2) < 0 {
			featureSet.set[feature] = struct{}{}
		}
	}
	return featureSet
}

func (fs *FeatureSet) Has(f Feature) bool {
	_, ok := fs.set[f]
	return ok
}
