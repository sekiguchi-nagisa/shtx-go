package main

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
)

type Version struct {
	Major uint
	Minor uint
	Patch uint
}

func NewVersionFill() Version {
	return Version{9999, 9999, 9999}
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
	return &Version{
		uint(major), uint(minar), uint(patch),
	}, nil
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

type FeatureSet struct {
	set map[Feature]struct{}
}

func NewFeatureSetFromVersion(v Version) FeatureSet {
	featureSet := FeatureSet{map[Feature]struct{}{}}

	if v.Compare(&Version{0, 38, 0}) >= 0 {
		featureSet.set[FeatureSubshell] = struct{}{}
	}
	return featureSet
}

func (fs *FeatureSet) Has(f Feature) bool {
	_, ok := fs.set[f]
	return ok
}
