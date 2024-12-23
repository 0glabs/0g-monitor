package stat

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func CompareGitVersionString(a, b interface{}) int {
	v1, err1 := ParseGitVersion(a.(string))
	v2, err2 := ParseGitVersion(b.(string))

	if err1 != nil {
		if err2 == nil {
			return -1
		}

		// compare raw version if both failed to parse git version
		return strings.Compare(a.(string), b.(string))
	}

	if err2 != nil {
		return 1
	}

	return v1.Compare(v2)
}

type GitVersion struct {
	Major    int
	Minor    int
	Build    int
	Revision int
	Commit   string
	Dirty    bool
}

func ParseGitVersion(version string) (result GitVersion, err error) {
	if len(version) == 0 || version == "unknown" {
		return
	}

	// parse "+" as dirty flag
	if strings.HasSuffix(version, "+") {
		result.Dirty = true
		version = strings.TrimSuffix(version, "+")
	}

	// parse "v" as version prefix, otherwise, treat it as commit
	if !strings.HasPrefix(version, "v") {
		result.Commit = version
		return
	}

	fields := strings.Split(version[1:], "-")
	numFields := len(fields)
	if numFields == 0 || numFields > 3 {
		return GitVersion{}, errors.Errorf("Failed to split version by dash, invalid number of fields %v", numFields)
	}

	// parse major, minor and build
	if result.Major, result.Minor, result.Build, err = parseVersion(fields[0]); err != nil {
		return GitVersion{}, errors.WithMessage(err, "Failed to parse version")
	}

	switch numFields {
	case 2:
		result.Commit = fields[1]
	case 3:
		if result.Revision, err = strconv.Atoi(fields[1]); err != nil {
			return GitVersion{}, errors.WithMessage(err, "Failed to parse Revision as Int")
		}

		result.Commit = fields[2]
	}

	return
}

func parseVersion(version string) (major, minor, build int, err error) {
	fields := strings.Split(version, ".")

	numFields := len(fields)
	if numFields == 0 || numFields > 3 {
		return 0, 0, 0, errors.Errorf("Failed to split version by dot, invalid number of fields %v", numFields)
	}

	if major, err = strconv.Atoi(fields[0]); err != nil {
		return 0, 0, 0, errors.WithMessage(err, "Failed to parse Major as Int")
	}

	if numFields > 1 {
		if minor, err = strconv.Atoi(fields[1]); err != nil {
			return 0, 0, 0, errors.WithMessage(err, "Failed to parse Minor as Int")
		}
	}

	if numFields > 2 {
		if build, err = strconv.Atoi(fields[2]); err != nil {
			return 0, 0, 0, errors.WithMessage(err, "Failed to parse Build as Int")
		}
	}

	return major, minor, build, nil
}

func (version GitVersion) Compare(other GitVersion) int {
	s1 := []int{version.Major, version.Minor, version.Build, version.Revision}
	s2 := []int{other.Major, other.Minor, other.Build, other.Revision}
	if v := slices.Compare(s1, s2); v != 0 {
		return v
	}

	if v := strings.Compare(version.Commit, other.Commit); v != 0 {
		return v
	}

	bool2Int := func(v bool) int {
		if v {
			return 1
		}

		return 0
	}

	return bool2Int(version.Dirty) - bool2Int(other.Dirty)
}
