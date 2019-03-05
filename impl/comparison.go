package impl

import (
	"fmt"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/semver/semver"
	"regexp"
)

func init() {
	eval.PuppetMatch = match
}

func match(a eval.Value, b eval.Value) bool {
	result := false
	switch b := b.(type) {
	case eval.Type:
		result = eval.IsInstance(b, a)

	case eval.StringValue, *types.RegexpValue:
		var rx *regexp.Regexp
		if s, ok := b.(eval.StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(eval.Error(eval.MatchNotRegexp, issue.H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*types.RegexpValue).Regexp()
		}

		sv, ok := a.(eval.StringValue)
		if !ok {
			panic(eval.Error(eval.MatchNotString, issue.H{`left`: a.PType()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			result = true
		}

	case *types.SemVerValue, *types.SemVerRangeValue:
		var version semver.Version

		if v, ok := a.(*types.SemVerValue); ok {
			version = v.Version()
		} else if s, ok := a.(eval.StringValue); ok {
			var err error
			version, err = semver.ParseVersion(s.String())
			if err != nil {
				panic(eval.Error(eval.NotSemver, issue.H{`detail`: err.Error()}))
			}
		} else {
			panic(eval.Error(eval.NotSemver,
				issue.H{`detail`: fmt.Sprintf(`A value of type %s cannot be converted to a SemVer`, a.PType().String())}))
		}
		if lv, ok := b.(*types.SemVerValue); ok {
			result = lv.Version().Equals(version)
		} else {
			result = b.(*types.SemVerRangeValue).VersionRange().Includes(version)
		}

	default:
		result = eval.PuppetEquals(b, a)
	}
	return result
}
