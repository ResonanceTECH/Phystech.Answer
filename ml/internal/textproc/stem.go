// Package textproc provides Russian-aware tokenisation and stemming for
// lexical retrieval (BM25). The stemmer is a pure-Go implementation of the
// Snowball Russian algorithm (Porter-style suffix stripping).
package textproc

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+(?:[-][\p{L}\p{N}]+)*`)

// Stopwords: frequent Russian function words without retrieval signal.
var stopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`и в во не что он на я с со как а то все она так его но да ты к у же вы за бы по только ее её мне было вот от меня еще ещё нет о из ему теперь когда даже ну вдруг ли если уже или ни быть был него до вас нибудь опять уж вам ведь там потом себя ничего ей может они тут где есть надо ней для мы тебя их чем была сам чтоб без будто чего раз тоже себе под будет ж тогда кто этот того потому этого какой совсем ним здесь этом один почти мой тем чтобы нее сейчас были куда зачем всех никогда можно при наконец два об другой хоть после над больше тот через эти нас про всего них какая много разве три эту моя впрочем хорошо свою этой перед иногда лучше чуть том нельзя такой им более всегда конечно всю между это мне нужно нужен нужна как какие каков который которая которые которых также либо данный настоящий`) {
		stopwords[w] = true
	}
}

// Tokens returns lower-cased word tokens with ё→е normalisation.
func Tokens(s string) []string {
	s = strings.ReplaceAll(strings.ToLower(s), "ё", "е")
	return tokenRe.FindAllString(s, -1)
}

// Terms returns stemmed index terms: stopwords and 1-rune tokens removed;
// numbers are kept as-is (номера приказов, пунктов).
func Terms(s string) []string {
	toks := Tokens(s)
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		if stopwords[t] || utf8.RuneCountInString(t) < 2 {
			continue
		}
		out = append(out, Stem(t))
	}
	return out
}

// ---- Snowball Russian stemmer ----

var (
	perfectiveGerund1 = []string{"вшись", "вши", "в"}
	perfectiveGerund2 = []string{"ившись", "ывшись", "ивши", "ывши", "ив", "ыв"}
	adjective         = []string{"ими", "ыми", "его", "ого", "ему", "ому", "ее", "ие", "ые", "ое", "ей", "ий", "ый", "ой", "ем", "им", "ым", "ом", "их", "ых", "ую", "юю", "ая", "яя", "ою", "ею"}
	participle1       = []string{"ем", "нн", "вш", "ющ", "щ"}
	participle2       = []string{"ивш", "ывш", "ующ"}
	reflexive         = []string{"ся", "сь"}
	verb1             = []string{"ете", "йте", "ешь", "нно", "ла", "на", "ли", "ем", "ло", "но", "ет", "ют", "ны", "ть", "й", "л", "н"}
	verb2             = []string{"ейте", "уйте", "ила", "ыла", "ена", "ите", "или", "ыли", "ило", "ыло", "ено", "ует", "уют", "ены", "ить", "ыть", "ишь", "ей", "уй", "ил", "ыл", "им", "ым", "ен", "ят", "ит", "ыт", "ую", "ю"}
	noun              = []string{"иями", "ями", "ами", "иях", "ией", "иям", "ием", "ев", "ов", "ие", "ье", "еи", "ии", "ей", "ой", "ий", "ям", "ем", "ам", "ом", "ах", "ях", "ию", "ью", "ия", "ья", "а", "е", "и", "й", "о", "у", "ы", "ь", "ю", "я"}
	superlative       = []string{"ейше", "ейш"}
	derivational      = []string{"ость", "ост"}
)

func isVowel(r rune) bool { return strings.ContainsRune("аеиоуыэюя", r) }

// Stem returns the Snowball Russian stem of a lower-case word. Non-Cyrillic
// tokens are returned unchanged.
func Stem(word string) string {
	w := []rune(word)
	if len(w) < 3 || !isCyrillic(w) {
		return word
	}
	rv := len(w)
	for i, r := range w {
		if isVowel(r) {
			rv = i + 1
			break
		}
	}
	r2 := regionR2(w)

	// Step 1.
	if n := endingWithPrecedent(w, rv, perfectiveGerund1); n > 0 {
		w = w[:len(w)-n]
	} else if n := ending(w, rv, perfectiveGerund2); n > 0 {
		w = w[:len(w)-n]
	} else {
		if n := ending(w, rv, reflexive); n > 0 {
			w = w[:len(w)-n]
		}
		if n := adjectival(w, rv); n > 0 {
			w = w[:len(w)-n]
		} else if n := endingWithPrecedent(w, rv, verb1); n > 0 {
			w = w[:len(w)-n]
		} else if n := ending(w, rv, verb2); n > 0 {
			w = w[:len(w)-n]
		} else if n := ending(w, rv, noun); n > 0 {
			w = w[:len(w)-n]
		}
	}
	// Step 2.
	if len(w) > rv && w[len(w)-1] == 'и' {
		w = w[:len(w)-1]
	}
	// Step 3.
	if n := ending(w, r2, derivational); n > 0 {
		w = w[:len(w)-n]
	}
	// Step 4.
	if hasSuffix(w, rv, "нн") {
		w = w[:len(w)-1]
	} else if n := ending(w, rv, superlative); n > 0 {
		w = w[:len(w)-n]
		if hasSuffix(w, rv, "нн") {
			w = w[:len(w)-1]
		}
	} else if len(w) > rv && w[len(w)-1] == 'ь' {
		w = w[:len(w)-1]
	}
	return string(w)
}

func isCyrillic(w []rune) bool {
	for _, r := range w {
		if r >= 'а' && r <= 'я' {
			return true
		}
	}
	return false
}

// regionR2 returns the start index of R2 (standard Snowball definition).
func regionR2(w []rune) int {
	r1 := len(w)
	for i := 1; i < len(w); i++ {
		if !isVowel(w[i]) && isVowel(w[i-1]) {
			r1 = i + 1
			break
		}
	}
	for i := r1 + 1; i < len(w); i++ {
		if !isVowel(w[i]) && isVowel(w[i-1]) {
			return i + 1
		}
	}
	return len(w)
}

func hasSuffix(w []rune, region int, suf string) bool {
	s := []rune(suf)
	if len(w)-len(s) < region || len(s) > len(w) {
		return false
	}
	return string(w[len(w)-len(s):]) == suf
}

// ending returns the length of the longest suffix from list inside region.
func ending(w []rune, region int, list []string) int {
	best := 0
	for _, suf := range list {
		if n := utf8.RuneCountInString(suf); n > best && hasSuffix(w, region, suf) {
			best = n
		}
	}
	return best
}

// endingWithPrecedent: suffix must be preceded by а or я (which stays).
func endingWithPrecedent(w []rune, region int, list []string) int {
	best := 0
	for _, suf := range list {
		n := utf8.RuneCountInString(suf)
		if n <= best || !hasSuffix(w, region, suf) {
			continue
		}
		i := len(w) - n - 1
		if i >= region && (w[i] == 'а' || w[i] == 'я') {
			best = n
		}
	}
	return best
}

// adjectival = adjective, optionally preceded by a participle suffix.
func adjectival(w []rune, rv int) int {
	n := ending(w, rv, adjective)
	if n == 0 {
		return 0
	}
	rest := w[:len(w)-n]
	if p := endingWithPrecedent(rest, rv, participle1); p > 0 {
		return n + p
	}
	if p := ending(rest, rv, participle2); p > 0 {
		return n + p
	}
	return n
}
