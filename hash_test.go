package gps

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"
)

func TestHashInputs(t *testing.T) {
	fix := basicFixtures["shared dependency with overlapping constraints"]

	params := SolveParameters{
		RootDir:         string(fix.ds[0].n),
		RootPackageTree: fix.rootTree(),
		Manifest:        fix.rootmanifest(),
	}

	s, err := Prepare(params, newdepspecSM(fix.ds, nil))
	if err != nil {
		t.Errorf("Unexpected error while prepping solver: %s", err)
		t.FailNow()
	}

	dig := s.HashInputs()
	h := sha256.New()

	elems := []string{
		hhConstraints,
		"a",
		"1.0.0",
		"b",
		"1.0.0",
		hhImportsReqs,
		"a",
		"b",
		hhIgnores,
		hhOverrides,
		hhAnalyzer,
		"depspec-sm-builtin",
		"1.0.0",
	}
	for _, v := range elems {
		h.Write([]byte(v))
	}
	correct := h.Sum(nil)

	if !bytes.Equal(dig, correct) {
		t.Errorf("Hashes are not equal. Inputs:\n%s", diffHashingInputs(s, elems))
	}
}

func TestHashInputsReqsIgs(t *testing.T) {
	fix := basicFixtures["shared dependency with overlapping constraints"]

	rm := fix.rootmanifest().(simpleRootManifest).dup()
	rm.ig = map[string]bool{
		"foo": true,
		"bar": true,
	}

	params := SolveParameters{
		RootDir:         string(fix.ds[0].n),
		RootPackageTree: fix.rootTree(),
		Manifest:        rm,
	}

	s, err := Prepare(params, newdepspecSM(fix.ds, nil))
	if err != nil {
		t.Errorf("Unexpected error while prepping solver: %s", err)
		t.FailNow()
	}

	dig := s.HashInputs()
	h := sha256.New()

	elems := []string{
		hhConstraints,
		"a",
		"1.0.0",
		"b",
		"1.0.0",
		hhImportsReqs,
		"a",
		"b",
		hhIgnores,
		"bar",
		"foo",
		hhOverrides,
		hhAnalyzer,
		"depspec-sm-builtin",
		"1.0.0",
	}
	for _, v := range elems {
		h.Write([]byte(v))
	}
	correct := h.Sum(nil)

	if !bytes.Equal(dig, correct) {
		t.Errorf("Hashes are not equal. Inputs:\n%s", diffHashingInputs(s, elems))
	}

	// Add requires
	rm.req = map[string]bool{
		"baz": true,
		"qux": true,
	}

	params.Manifest = rm

	s, err = Prepare(params, newdepspecSM(fix.ds, nil))
	if err != nil {
		t.Errorf("Unexpected error while prepping solver: %s", err)
		t.FailNow()
	}

	dig = s.HashInputs()
	h = sha256.New()

	elems = []string{
		hhConstraints,
		"a",
		"1.0.0",
		"b",
		"1.0.0",
		hhImportsReqs,
		"a",
		"b",
		"baz",
		"qux",
		hhIgnores,
		"bar",
		"foo",
		hhOverrides,
		hhAnalyzer,
		"depspec-sm-builtin",
		"1.0.0",
	}
	for _, v := range elems {
		h.Write([]byte(v))
	}
	correct = h.Sum(nil)

	if !bytes.Equal(dig, correct) {
		t.Errorf("Hashes are not equal. Inputs:\n%s", diffHashingInputs(s, elems))
	}

	// remove ignores, just test requires alone
	rm.ig = nil
	params.Manifest = rm

	s, err = Prepare(params, newdepspecSM(fix.ds, nil))
	if err != nil {
		t.Errorf("Unexpected error while prepping solver: %s", err)
		t.FailNow()
	}

	dig = s.HashInputs()
	h = sha256.New()

	elems = []string{
		hhConstraints,
		"a",
		"1.0.0",
		"b",
		"1.0.0",
		hhImportsReqs,
		"a",
		"b",
		"baz",
		"qux",
		hhIgnores,
		hhOverrides,
		hhAnalyzer,
		"depspec-sm-builtin",
		"1.0.0",
	}
	for _, v := range elems {
		h.Write([]byte(v))
	}
	correct = h.Sum(nil)

	if !bytes.Equal(dig, correct) {
		t.Errorf("Hashes are not equal. Inputs:\n%s", diffHashingInputs(s, elems))
	}
}

func TestHashInputsOverrides(t *testing.T) {
	basefix := basicFixtures["shared dependency with overlapping constraints"]

	// Set up base state that we'll mutate over the course of each test
	rm := basefix.rootmanifest().(simpleRootManifest).dup()
	params := SolveParameters{
		RootDir:         string(basefix.ds[0].n),
		RootPackageTree: basefix.rootTree(),
		Manifest:        rm,
	}

	table := []struct {
		name  string
		mut   func()
		elems []string
	}{
		{
			name: "override source; not imported, no deps pp",
			mut: func() {
				// First case - override just source, on something without
				// corresponding project properties in the dependencies from
				// root
				rm.ovr = map[ProjectRoot]ProjectProperties{
					"c": ProjectProperties{
						Source: "car",
					},
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"c",
				"car",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override source; required, no deps pp",
			mut: func() {
				// Put c into the requires list, which should make it show up under
				// constraints
				rm.req = map[string]bool{
					"c": true,
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				"c",
				"car",
				"*", // Any isn't included under the override, but IS for the constraint b/c it's equivalent
				hhImportsReqs,
				"a",
				"b",
				"c",
				hhIgnores,
				hhOverrides,
				"c",
				"car",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override source; imported, no deps pp",
			mut: func() {
				// Take c out of requires list and put it directly in root's imports
				rm.req = nil
				poe := params.RootPackageTree.Packages["root"]
				poe.P.Imports = []string{"a", "b", "c"}
				params.RootPackageTree.Packages["root"] = poe
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				"c",
				"car",
				"*",
				hhImportsReqs,
				"a",
				"b",
				"c",
				hhIgnores,
				hhOverrides,
				"c",
				"car",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "other override constraint; not imported, no deps pp",
			mut: func() {
				// Override not in root, just with constraint
				rm.ovr["d"] = ProjectProperties{
					Constraint: NewBranch("foobranch"),
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				"c",
				"car",
				"*",
				hhImportsReqs,
				"a",
				"b",
				"c",
				hhIgnores,
				hhOverrides,
				"c",
				"car",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override constraint; not imported, no deps pp",
			mut: func() {
				// Remove the "c" pkg from imports for remainder of tests
				poe := params.RootPackageTree.Packages["root"]
				poe.P.Imports = []string{"a", "b"}
				params.RootPackageTree.Packages["root"] = poe
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"c",
				"car",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override both; not imported, no deps pp",
			mut: func() {
				// Override not in root, both constraint and network name
				rm.ovr["c"] = ProjectProperties{
					Source:     "groucho",
					Constraint: NewBranch("plexiglass"),
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"1.0.0",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"c",
				"groucho",
				"plexiglass",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override constraint; imported, with constraint",
			mut: func() {
				// Override dep present in root, just constraint
				rm.ovr["a"] = ProjectProperties{
					Constraint: NewVersion("fluglehorn"),
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"fluglehorn",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"a",
				"fluglehorn",
				"c",
				"groucho",
				"plexiglass",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override source; imported, with constraint",
			mut: func() {
				// Override in root, only network name
				rm.ovr["a"] = ProjectProperties{
					Source: "nota",
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"nota",
				"1.0.0",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"a",
				"nota",
				"c",
				"groucho",
				"plexiglass",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
		{
			name: "override both; imported, with constraint",
			mut: func() {
				// Override in root, network name and constraint
				rm.ovr["a"] = ProjectProperties{
					Source:     "nota",
					Constraint: NewVersion("fluglehorn"),
				}
			},
			elems: []string{
				hhConstraints,
				"a",
				"nota",
				"fluglehorn",
				"b",
				"1.0.0",
				hhImportsReqs,
				"a",
				"b",
				hhIgnores,
				hhOverrides,
				"a",
				"nota",
				"fluglehorn",
				"c",
				"groucho",
				"plexiglass",
				"d",
				"foobranch",
				hhAnalyzer,
				"depspec-sm-builtin",
				"1.0.0",
			},
		},
	}

	for _, fix := range table {
		fix.mut()
		params.Manifest = rm

		s, err := Prepare(params, newdepspecSM(basefix.ds, nil))
		if err != nil {
			t.Errorf("(fix: %s) Unexpected error while prepping solver: %s", fix.name, err)
			t.FailNow()
		}

		h := sha256.New()
		for _, v := range fix.elems {
			h.Write([]byte(v))
		}

		if !bytes.Equal(s.HashInputs(), h.Sum(nil)) {
			t.Errorf("(fix: %s) Hashes are not equal. Inputs:\n%s", fix.name, diffHashingInputs(s, fix.elems))
		}
	}
}

func diffHashingInputs(s Solver, wnt []string) string {
	actual := HashingInputsAsString(s)
	got := strings.Split(actual, "\n")

	lg, lw := len(got), len(wnt)

	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 4, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "  (GOT)  \t  (WANT)  \t")

	if lg == lw {
		// same length makes the loop pretty straightforward
		for i := 0; i < lg; i++ {
			fmt.Fprintf(tw, "%s\t%s\t\n", got[i], wnt[i])
		}
	} else if lg > lw {
		offset := 0
		for i := 0; i < lg; i++ {
			if lw <= i-offset {
				fmt.Fprintf(tw, "%s\t\t\n", got[i])
			} else if got[i] != wnt[i-offset] && i+1 < lg && got[i+1] == wnt[i-offset] {
				// if the next slot is a match, realign by skipping this one and
				// bumping the offset
				fmt.Fprintf(tw, "%s\t\t\n", got[i])
				offset++
			} else {
				fmt.Fprintf(tw, "%s\t%s\t\n", got[i], wnt[i-offset])
			}
		}
	} else {
		offset := 0
		for i := 0; i < lw; i++ {
			if lg <= i-offset {
				fmt.Fprintf(tw, "\t%s\t\n", wnt[i])
			} else if got[i-offset] != wnt[i] && i+1 < lw && got[i-offset] == wnt[i+1] {
				// if the next slot is a match, realign by skipping this one and
				// bumping the offset
				fmt.Fprintf(tw, "\t%s\t\n", wnt[i])
				offset++
			} else {
				fmt.Fprintf(tw, "%s\t%s\t\n", got[i-offset], wnt[i])
			}
		}
	}

	tw.Flush()
	return buf.String()
}
