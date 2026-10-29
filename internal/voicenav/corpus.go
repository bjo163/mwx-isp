package voicenav

import (
	"sort"
	"strings"
)

// Fixture is one transcript, the ASR error class it stands for, why that error
// is the realistic one for this console, and what the study requires the
// resolver to answer. The phrases are authored; the verdicts are measured.
type Fixture struct {
	Utterance string
	Class     string
	Note      string
	Expect    string
}

// Expectation vocabulary.
//
//	navigate:/path   - must resolve to exactly that screen
//	ambiguous        - must ask, because two screens are equally close
//	refused          - must stop, and say so
//	ask-or-refuse    - must not confidently name any screen
//	any-safe         - a mutation instruction the derived vocabulary does not
//	                   cover: navigation is allowed, an operation never is
const (
	expectAmbiguous   = "ambiguous"
	expectAskOrRefuse = "ask-or-refuse"
	expectAnySafe     = "any-safe"
	expectNavigatePfx = "navigate:"
)

// corpus is the transcript set. It is written for the two locales the console
// actually ships, because those are the only words an operator can be heard
// saying, and every expected screen is a menu destination measured from
// CustomMenu.tsx.
func corpus() []Fixture {
	return []Fixture{
		// The clean commands: what the console says on screen, said back.
		{`pelanggan`, `canonical`, `the Indonesian label for the customer list, spoken alone`, `navigate:/isp/customers`},
		{`paket internet`, `canonical`, `two-word Indonesian label, in order`, `navigate:/isp/packages`},
		{`langganan`, `canonical`, `the subscriptions label`, `navigate:/isp/subscriptions`},
		{`voucher hotspot`, `canonical`, `the Indonesian word order, not the English one`, `navigate:/isp/vouchers`},
		{`tiket gangguan`, `canonical`, `trouble tickets in Indonesian`, `navigate:/isp/tickets`},
		{`tagihan`, `canonical`, `the invoices label in Indonesian, which is not the English word`, `navigate:/isp/invoices`},
		{`pembayaran`, `canonical`, `the payments label`, `navigate:/isp/payments`},
		{`manajemen profil`, `canonical`, `the RADIUS profiles label`, `navigate:/radius/profiles`},
		{`manajemen pengguna`, `canonical`, `the RADIUS users label, one word from the profiles label`, `navigate:/radius/users`},
		{`sesi online`, `canonical`, `the online sessions label`, `navigate:/radius/online`},
		{`log akuntansi`, `canonical`, `the accounting logs label`, `navigate:/radius/accounting`},
		{`perangkat nas`, `canonical`, `the NAS devices label, with an acronym at the end`, `navigate:/network/nas`},
		{`aset odp ftth`, `canonical`, `three acronyms in one Indonesian label`, `navigate:/network/odp`},
		{`node jaringan`, `canonical`, `the network nodes label`, `navigate:/network/nodes`},
		{`konfigurasi sistem`, `canonical`, `the system configuration label`, `navigate:/system/config`},
		{`manajemen operator`, `canonical`, `the operators label - a word that is also the tail of a write route`, `navigate:/system/operators`},
		{`sertifikat`, `canonical`, `the certificates label`, `navigate:/system/certificate`},
		{`branding produk`, `canonical`, `the branding label, with the Indonesian word order`, `navigate:/system/branding`},
		{`tenant isp`, `canonical`, `the platform tenants label`, `navigate:/platform/tenants`},
		{`dasbor`, `canonical`, `the dashboard label, which is also the route the menu numbers 1`, `navigate:/`},

		// The English set, and an operator who code-switches: the console renders
		// one locale per session but the product is used in both.
		{`invoices`, `canonical-en`, `the English invoices label, exact`, `navigate:/isp/invoices`},
		{`hotspot vouchers`, `code-switch`, `an English plural over an Indonesian-rendered screen`, `navigate:/isp/vouchers`},
		{`online sessions`, `code-switch`, `the English word order, which is the opposite of the Indonesian label`, `navigate:/radius/online`},
		{`network nodes`, `canonical-en`, `the English nodes label`, `navigate:/network/nodes`},
		{`system configuration`, `canonical-en`, `the English configuration label, two words`, `navigate:/system/config`},

		// Lead-in words: a person does not say a screen name alone.
		{`buka halaman manajemen pengguna`, `lead-in`, `the most common Indonesian phrasing, with a verb and a noun in front`, `navigate:/radius/users`},
		{`tolong tampilkan dasbor`, `lead-in`, `a polite prefix that carries no destination`, `navigate:/`},
		{`go to internet packages`, `lead-in`, `an English navigation prefix before an English label`, `navigate:/isp/packages`},
		{`ke halaman tiket gangguan`, `lead-in`, `two Indonesian lead-in words in front of the label; the score only asks for the label`, `navigate:/isp/tickets`},

		// Real recognizer damage: the word is wrong, the screen is not.
		{`peladangan`, `homophone`, `a plausible Indonesian near-word for pelanggan`, `navigate:/isp/customers`},
		{`manajeman profil`, `homophone`, `a transposed vowel in a label word`, `navigate:/radius/profiles`},
		{`pembayran`, `deletion`, `a dropped vowel, the most common ASR slip in Indonesian`, `navigate:/isp/payments`},
		{`konfigurasi sist`, `truncation`, `the recognizer cut the last two letters of the second word`, `navigate:/system/config`},
		{`lang gan an`, `boundary`, `one word, three tokens: what a Spanish-trained model does to langganan`, `navigate:/isp/subscriptions`},
		{`sesi on line`, `boundary`, `online rendered as two words`, `navigate:/radius/online`},
		{`anu voucher hotspot`, `insertion`, `a filler word inserted before the label`, `navigate:/isp/vouchers`},
		{`manajemen profil radius`, `insertion`, `an extra true word that belongs to no label`, `navigate:/radius/profiles`},
		{`dashboard`, `code-switch`, `the English label, spelled with the American 'd' the id locale does not use`, `navigate:/`},

		// Damage the study requires the resolver to refuse rather than guess.
		{`paket`, `truncation`, `one word of a two-word label: a fragment, not a command`, `ask-or-refuse`},
		{`manajemen`, `truncation`, `the word three different labels share`, `ask-or-refuse`},
		{`jaringan`, `truncation`, `the word two different labels share`, `ask-or-refuse`},

		// Two screens in one breath.
		{`buka tagihan dan pembayaran`, `two-targets`, `a coordinator that joins two real labels at equal distance`, expectAmbiguous},
		{`pelanggan dan paket internet`, `two-targets`, `two Indonesian labels in one utterance`, expectAmbiguous},

		// Silence and noise, which must produce a prompt, not a page.
		{``, `silence`, `an endpoint that heard nothing`, `refused`},
		{`   `, `silence`, `noise that normalized to nothing`, `refused`},
		{`hm ah uh`, `noise`, `a filler-only transcript`, `refused`},
		{`saya mau lihat sesuatu yang biasa saya buka`, `noise`, `a fluent sentence that names no screen at all`, `refused`},

		// Operations the console performs by POST/PUT/DELETE. The criterion says a
		// mis-transcription must not cause one, so each of these must stop.
		{`putuskan sesi pelanggan`, `action`, `the Indonesian label the server's own disconnect action renders`, `refused`},
		{`suspend pelanggan ini`, `action`, `the action word the API guard accepts, spelled exactly`, `refused`},
		{`reactivate langganan`, `action`, `another guard-accepted action word`, `refused`},
		{`terminate subscription`, `action`, `the guard's fifth verb, in English`, `refused`},
		{`reset branding`, `action`, `a write route's trailing segment plus a screen noun: the action wins`, `refused`},
		{`reverse pembayaran`, `action`, `the payment-reversal route's segment, mixed with the Indonesian payments label`, `refused`},
		{`disconnect`, `action`, `the action word alone`, `refused`},

		// Operations whose verb the derived vocabulary does not know. These are the
		// coverage limit, stated as a row rather than argued away: the resolver may
		// navigate, and navigation cannot mutate.
		{`bayar tagihan pelanggan itu`, `action-uncovered`, `a payment instruction in words no route segment uses`, expectAnySafe},
		{`hapus pelanggan`, `action-uncovered`, `a deletion instruction; the English verb is a route label but the Indonesian one is not`, expectAnySafe},
		{`kirim whatsapp ke pelanggan`, `action-uncovered`, `the send-whatsapp route described in Indonesian, with only the brand word carried over`, expectAnySafe},
		{`tangguhkan langganan billing yang menunggak`, `action-uncovered`, `an operation sentence whose verb is Indonesian and whose only screen noun is a whole label: the resolver lands on the read screen`, expectAnySafe},
	}
}

// verdicts scores the corpus against the resolver.
func verdicts(fs []Fixture, phrases []Phrase, ban map[string]string) []Verdict {
	out := make([]Verdict, 0, len(fs))
	for _, f := range fs {
		v := Resolve(f.Utterance, phrases, ban)
		v.Class = f.Class
		v.Note = f.Note
		v.Expect = f.Expect
		v.Ok = matches(f.Expect, v)
		out = append(out, v)
	}
	return out
}

func matches(expect string, v Verdict) bool {
	switch {
	case expect == expectAmbiguous:
		return v.Kind == ambiguous
	case expect == expectAskOrRefuse:
		return v.Kind != navigate
	case expect == expectAnySafe:
		// Every verdict is safe by construction; the row records what the refusal
		// vocabulary did, not whether a mutation happened.
		return true
	case strings.HasPrefix(expect, expectNavigatePfx):
		return v.Kind == navigate && v.Path == strings.TrimPrefix(expect, expectNavigatePfx)
	default:
		return v.Kind == refused
	}
}

// ClassRow counts, per error class, how the corpus landed.
type ClassRow struct {
	Class     string
	Fixtures  int
	Navigated int
	Asked     int
	Refused   int
	Matched   int
}

func classRows(vs []Verdict) []ClassRow {
	by := map[string]*ClassRow{}
	order := []string{}
	for _, v := range vs {
		r, ok := by[v.Class]
		if !ok {
			r = &ClassRow{Class: v.Class}
			by[v.Class] = r
			order = append(order, v.Class)
		}
		r.Fixtures++
		switch v.Kind {
		case navigate:
			r.Navigated++
		case ambiguous:
			r.Asked++
		default:
			r.Refused++
		}
		if v.Ok {
			r.Matched++
		}
	}
	out := make([]ClassRow, 0, len(order))
	for _, c := range order {
		out = append(out, *by[c])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Class < out[j].Class })
	return out
}
