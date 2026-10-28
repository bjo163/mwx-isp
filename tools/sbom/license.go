package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// licenseFileLimit bounds how much of a license file is read; the identifying
// text is always near the top.
const licenseFileLimit = 64 << 10

// licenseHeaderLength is how much of the normalized text counts as the header.
// The GPL family, MPL and Apache put their title first, while their bodies
// cross-reference other licenses — MPL 2.0 §3.3 names the GNU Affero General
// Public License, so a body-wide marker would misfile every MPL file as AGPL.
const licenseHeaderLength = 1200

// detectLicense classifies the license of a module directory by reading its
// license file (LICENSE, the British spelling of the same word, or COPYING).
// It is a fingerprint table, not a full license scanner: a recognized text maps
// to its SPDX identifier, anything else
// is reported as NOASSERTION so the gap shows up in review instead of being
// guessed. A missing directory or file is also NOASSERTION.
func detectLicense(dir string) license {
	if dir == "" {
		return license{Name: noAssertion}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return license{Name: noAssertion}
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isLicenseFileName(entry.Name()) {
			candidates = append(candidates, entry.Name())
		}
	}
	sort.Strings(candidates)
	for _, name := range candidates {
		id := classifyLicenseFile(filepath.Join(dir, name))
		if id != "" {
			return license{ID: id}
		}
	}
	return license{Name: noAssertion}
}

func isLicenseFileName(name string) bool {
	upper := strings.ToUpper(name)
	base := strings.TrimSuffix(upper, filepath.Ext(upper))
	switch base {
	case "LICENSE", "LICENCE", "COPYING", "UNLICENSE", "COPYRIGHT": //nolint:misspell // LICENCE is a real British-spelled file name
		return true
	}
	// LICENSE-MIT, LICENSE-APACHE, COPYING.LESSER and similar suffixed names.
	return strings.HasPrefix(base, "LICENSE-") || strings.HasPrefix(base, "LICENCE-") || //nolint:misspell // British spelling appears in real module trees
		strings.HasPrefix(base, "COPYING-") || strings.HasPrefix(base, "COPYING.")
}

// classifyLicenseFile returns the SPDX identifier for the text in path, or ""
// when no fingerprint matches.
func classifyLicenseFile(path string) string {
	f, err := os.Open(path) //nolint:gosec // G304: path is a license file discovered inside a module cache directory
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, licenseFileLimit)
	n, _ := f.Read(buf)
	text := normalizeLicenseText(string(buf[:n]))
	if text == "" {
		return ""
	}
	header := text
	if len(header) > licenseHeaderLength {
		header = header[:licenseHeaderLength]
	}
	for _, fp := range licenseFingerprints {
		haystack := text
		if fp.headerOnly {
			haystack = header
		}
		if strings.Contains(haystack, fp.marker) && fp.extra(text) {
			return fp.id
		}
	}
	return ""
}

func normalizeLicenseText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

type fingerprint struct {
	id     string
	marker string
	// headerOnly restricts the marker to the start of the file, for licenses
	// that open with a title but mention other licenses further down.
	headerOnly bool
	extra      func(string) bool
}

func always(string) bool { return true }

// licenseFingerprints is ordered most-specific first: GPL variants share the
// "general public license" phrase, and the BSD family shares its opening line.
var licenseFingerprints = []fingerprint{
	{"AGPL-3.0-only", "gnu affero general public license", true, always},
	{"LGPL-3.0-only", "gnu lesser general public license", true, func(t string) bool { return strings.Contains(t, "version 3") }},
	{"LGPL-2.1-only", "gnu lesser general public license", true, always},
	{"GPL-3.0-only", "gnu general public license", true, func(t string) bool { return strings.Contains(t, "version 3") }},
	{"GPL-2.0-only", "gnu general public license", true, always},
	{"Apache-2.0", "apache license", true, func(t string) bool { return strings.Contains(t, "version 2.0") }},
	// HashiCorp projects write "Mozilla Public License, version 2.0".
	{"MPL-2.0", "mozilla public license", true, func(t string) bool { return strings.Contains(t, "version 2.0") }},
	{"BSD-3-Clause", "redistribution and use in source and binary forms", false, func(t string) bool {
		return strings.Contains(t, "neither the name")
	}},
	{"BSD-2-Clause", "redistribution and use in source and binary forms", false, always},
	// coder/websocket and other ISC variants drop the "and/or", so only the
	// prefix and the fee clause are matched; 0BSD is ISC without the copyright
	// notice requirement.
	{"0BSD", "permission to use, copy, modify", false, func(t string) bool {
		return strings.Contains(t, "for any purpose with or without fee") &&
			!strings.Contains(t, "the above copyright notice and this permission notice appear")
	}},
	{"ISC", "permission to use, copy, modify", false, func(t string) bool {
		return strings.Contains(t, "for any purpose with or without fee")
	}},
	{"MIT", "permission is hereby granted, free of charge, to any person obtaining a copy", false, always},
	{"Unlicense", "this is free and unencumbered software released into the public domain", false, always},
	{"CC0-1.0", "cc0 1.0 universal", false, always},
	{"BSL-1.0", "boost software license", false, always},
	{"Zlib", "origin of this software must not be misrepresented", false, always},
}
