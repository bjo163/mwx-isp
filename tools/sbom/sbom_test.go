package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	mitText = `MIT License

Copyright (c) 2024 Example

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction.`

	apacheText = `                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/`

	bsd3Text = `Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

   * Redistributions of source code must retain the above copyright notice.
   * Neither the name of Google Inc. nor the names of its contributors may be
     used to endorse or promote products derived from this software.`

	bsd2Text = `Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice.`

	// MPL 2.0 names the GNU licenses in §3.3, including the Affero variant.
	// A body-wide marker list misreads every MPL file as AGPL.
	mplText = `Mozilla Public License Version 2.0
==================================

1. Definitions
--------------

1.1. "Contributor"
    means each individual or legal entity that creates, contributes to
    the creation of, or owns Covered Software.

1.2. "Contributor Version"
    means the combination of the Contributions of others (if any) used
    by a Contributor and that particular Contributor's Contribution.

1.3. "Contribution"
    means Covered Software of a particular Contributor.

1.4. "Covered Software"
    means Source Code Form to which the initial Contributor has attached
    the notice in Exhibit A, the Executable Form of such Source Code
    Form, and Modifications of such Source Code Form, in each case
    including portions thereof.

1.5. "Incompatible With Secondary Licenses"
    means

    (a) that the initial Contributor has attached the notice described
        in Exhibit B to the Covered Software; or

    (b) that the Covered Software was made available under the terms of
        version 1.1 or earlier of the License, but not also under the
        terms of a Secondary License.

1.6. "Executable Form"
    means any form of the work other than Source Code Form.

1.7. "Larger Work"
    means a work that combines Covered Software with other material, in
    a separate file or files, that is not Covered Software.

1.8. "License"
    means this document.

1.9. "Licensable"
    means having the right to grant, to the maximum extent possible,
    whether at the time of the initial grant or subsequently, any and
    all of the rights conveyed by this License.

1.10. "Modifications"
    means any of the following:

    (a) any file in Source Code Form that results from an addition to,
        deletion from, or modification of the contents of Covered
        Software; or

    (b) any new file in Source Code Form that contains any Covered
        Software.

2. License Grants and Conditions
--------------------------------

2.1. Grants

Each Contributor hereby grants You a world-wide, royalty-free,
non-exclusive license under intellectual property rights licensable by
such Contributor to use, reproduce, make available, modify, display,
perform, distribute, and otherwise exploit its Contributions, either on
an unmodified basis, with Modifications, or as part of a Larger Work.

2.2. Effective Date

The licenses granted in Section 2.1 with respect to any Contribution
become effective for each Contribution on the date the Contributor first
distributes such Contribution.

3. Responsibilities
-------------------

3.3. Distribution of a Larger Work

You may create and distribute a Larger Work under terms of Your choice, provided
that You also comply with the requirements of this License. The Larger Work may
be distributed under the terms of the GNU General Public License, Version 2.0,
the GNU Lesser General Public License, Version 2.1, the GNU Affero General
Public License, Version 3.0, or any later versions of those licenses.`

	mplHashicorpText = `Copyright © 2015-2022 HashiCorp, Inc.

Mozilla Public License, version 2.0

1. Definitions
--------------

1.1. "Contributor" means each individual or legal entity that creates,
contributes to the creation of, or owns Covered Software.`

	agplText = `GNU AFFERO GENERAL PUBLIC LICENSE
   Version 3, 19 November 2007

Copyright (C) 2007 Free Software Foundation, Inc. <https://fsf.org/>`
)

func TestDetectLicense(t *testing.T) {
	tests := []struct {
		name string
		file string
		text string
		id   string
	}{
		{"mit", "LICENSE", mitText, "MIT"},
		{"apache", "LICENSE.txt", apacheText, "Apache-2.0"},
		{"bsd3", "LICENSE", bsd3Text, "BSD-3-Clause"},
		{"bsd2", "LICENCE", bsd2Text, "BSD-2-Clause"}, //nolint:misspell // module trees really do ship LICENCE files
		{"isc", "LICENSE", "Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.", "ISC"},
		{"isc-coder-wording", "LICENSE.txt", "Copyright (c) 2025 Coder\n\nPermission to use, copy, modify, and distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.", "ISC"},
		{"0bsd", "LICENSE", "Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted.", "0BSD"},
		{"mpl-with-agpl-reference", "LICENSE", mplText, "MPL-2.0"},
		{"mpl-hashicorp-wording", "LICENSE", mplHashicorpText, "MPL-2.0"},
		{"agpl", "LICENSE", agplText, "AGPL-3.0-only"},
		{"suffixed", "LICENSE-MIT", mitText, "MIT"},
		{"unknown", "LICENSE", "All rights reserved. Contact legal@example.com.", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.text), 0o600); err != nil {
				t.Fatalf("write license file: %v", err)
			}
			got := detectLicense(dir)
			if tc.id == "" {
				if got.ID != "" || got.Name != noAssertion {
					t.Fatalf("unclassified text = %#v, want %s", got, noAssertion)
				}
				return
			}
			if got.ID != tc.id {
				t.Fatalf("detectLicense = %#v, want SPDX %q", got, tc.id)
			}
		})
	}
}

func TestDetectLicense_MissingDirectoryIsNoAssertion(t *testing.T) {
	for _, dir := range []string{"", filepath.Join(t.TempDir(), "does-not-exist")} {
		if got := detectLicense(dir); got.Name != noAssertion {
			t.Fatalf("detectLicense(%q) = %#v, want %s", dir, got, noAssertion)
		}
	}
}

func TestParseModuleList(t *testing.T) {
	stream := `{"Path":"github.com/bjo163/mwx-isp","Main":true}
{"Path":"layeh.com/radius","Version":"v0.0.0-20231213012653-1006025d24f8","Replace":{"Path":"github.com/talkincode/radius","Version":"v0.1.0"}}
{"Path":"github.com/google/uuid","Version":"v1.6.0","Indirect":true}
`
	mods, err := parseModuleStream(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("parseModuleStream: %v", err)
	}
	mainComp, comps, err := componentsFrom(t.TempDir(), mods)
	if err != nil {
		t.Fatalf("componentsFrom: %v", err)
	}
	if mainComp.Name != "github.com/bjo163/mwx-isp" || mainComp.Type != "application" {
		t.Fatalf("main component = %#v", mainComp)
	}
	if len(comps) != 2 {
		t.Fatalf("got %d components, want 2 (the main module is metadata)", len(comps))
	}

	sortComponents(comps)
	byName := map[string]comp{}
	for _, c := range comps {
		byName[c.Name] = c
	}

	uuid, ok := byName["github.com/google/uuid"]
	if !ok {
		t.Fatalf("uuid component missing: %#v", comps)
	}
	if !hasProperty(uuid, "go:indirect", "true") {
		t.Fatalf("uuid component lost its indirect property: %#v", uuid)
	}

	// A replaced module is inventoried under the code that is actually built,
	// with the original requirement kept for traceability.
	fork, ok := byName["github.com/talkincode/radius"]
	if !ok {
		t.Fatalf("replacement not inventoried: %#v", comps)
	}
	if fork.Version != "v0.1.0" {
		t.Fatalf("replacement version = %q, want v0.1.0", fork.Version)
	}
	if !hasProperty(fork, "go:replaces", "layeh.com/radius@v0.0.0-20231213012653-1006025d24f8") {
		t.Fatalf("replacement lost its original requirement: %#v", fork)
	}
}

func hasProperty(c comp, name, value string) bool {
	for _, p := range c.Properties {
		if p.Name == name && p.Value == value {
			return true
		}
	}
	return false
}

func TestUndownloadedModules(t *testing.T) {
	mods := []goModule{
		{Path: "github.com/bjo163/mwx-isp", Main: true},
		{Path: "example.com/cached", Version: "v1.0.0", Dir: "/cache/cached"},
		{Path: "example.com/missing", Version: "v1.2.3"},
		{Path: "example.com/fork", Version: "v0.1.0", Replace: &goModule{Path: "example.com/real", Version: "v0.2.0"}},
	}
	got := undownloadedModules(mods)
	want := []string{"example.com/fork@v0.1.0", "example.com/missing@v1.2.3"}
	if len(got) != len(want) {
		t.Fatalf("undownloadedModules = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("undownloadedModules = %v, want %v", got, want)
		}
	}
}

func TestGoPURL_EncodesIncompatibleVersion(t *testing.T) {
	if got, want := goPURL("example.com/mod", "v2.0.0+incompatible"), "pkg:golang/example.com/mod@v2.0.0%2Bincompatible"; got != want {
		t.Fatalf("goPURL = %q, want %q", got, want)
	}
}

func TestNormalizeNPMBOM_IsReproducible(t *testing.T) {
	first := `{
  "$schema": "http://cyclonedx.org/schema/bom-1.5.schema.json",
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "serialNumber": "urn:uuid:00000000-0000-0000-0000-000000000000",
  "version": 1,
  "metadata": {"timestamp": "2026-10-28T00:00:00.000Z", "tools": [{"vendor": "npm", "name": "cli", "version": "11.19.1"}]},
  "components": [
    {"bom-ref": "z-pkg@1.0.0", "type": "library", "name": "z-pkg", "version": "1.0.0"},
    {"bom-ref": "a-pkg@2.0.0", "type": "library", "name": "a-pkg", "version": "2.0.0"}
  ]
}`
	// Same dependency set, different serial number, timestamp, tool version and
	// component order: the normalized document must be byte-identical.
	second := `{"bomFormat":"CycloneDX","specVersion":"1.5","$schema":"http://cyclonedx.org/schema/bom-1.5.schema.json","serialNumber":"urn:uuid:11111111-1111-1111-1111-111111111111","version":1,"metadata":{"timestamp":"2026-12-31T23:59:59.000Z","tools":[{"vendor":"npm","name":"cli","version":"10.0.0"}]},"components":[{"version":"2.0.0","name":"a-pkg","type":"library","bom-ref":"a-pkg@2.0.0"},{"version":"1.0.0","name":"z-pkg","type":"library","bom-ref":"z-pkg@1.0.0"}]}`

	one, err := normalizeNPMBOM([]byte(first))
	if err != nil {
		t.Fatalf("normalize first: %v", err)
	}
	two, err := normalizeNPMBOM([]byte(second))
	if err != nil {
		t.Fatalf("normalize second: %v", err)
	}
	if string(one) != string(two) {
		t.Fatalf("normalized documents differ:\n%s\n---\n%s", one, two)
	}

	var doc map[string]any
	if err := json.Unmarshal(one, &doc); err != nil {
		t.Fatalf("unmarshal normalized document: %v", err)
	}
	if _, ok := doc["serialNumber"]; ok {
		t.Fatal("serialNumber must be dropped: it changes on every run")
	}
	md, _ := doc["metadata"].(map[string]any)
	if _, ok := md["timestamp"]; ok {
		t.Fatal("metadata.timestamp must be dropped: it changes on every run")
	}
	tools, _ := md["tools"].([]any)
	tool0, _ := tools[0].(map[string]any)
	if _, ok := tool0["version"]; ok {
		t.Fatal("tool version must be dropped: it differs per machine")
	}
	comps, _ := doc["components"].([]any)
	if len(comps) != 2 || ref(comps[0]) != "a-pkg@2.0.0" || ref(comps[1]) != "z-pkg@1.0.0" {
		t.Fatalf("components are not sorted by bom-ref: %v", comps)
	}
}
