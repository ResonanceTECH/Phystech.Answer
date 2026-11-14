// Command eval measures retrieval and answer quality on a labelled question
// set (eval/questions.jsonl) against a running ML service and writes a
// markdown report.
//
//	eval -url http://localhost:8000 -questions questions.jsonl -out report.md \
//	     -stage retrieval -configs bm25,vector,hybrid,hybrid+rerank,hybrid+rerank+amend
//	eval ... -stage e2e
//
// Retrieval metrics (per configuration):
//   - DocHit@K   — expected document is among retrieved fragments
//     (composite questions: all expected documents);
//   - MRR        — reciprocal rank of the first expected document;
//   - SectionHit — the expected пункт is inside a retrieved fragment;
//   - NF-acc     — "нет информации" questions correctly rejected;
//   - FalseRej   — answerable questions wrongly rejected.
//
// End-to-end metrics: status accuracy, expected document cited, expected
// пункт cited, share of steps with citations, invalid references removed,
// latency.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type question struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Docs     []string `json:"expect_docs"`
	Section  string   `json:"expect_section"`
	Type     string   `json:"type"`
}

type hit struct {
	Title   string  `json:"title"`
	Number  string  `json:"number"`
	Section string  `json:"section"`
	Score   float64 `json:"score"`
}

type options struct {
	Mode       string `json:"mode"`
	Reranker   string `json:"reranker"`
	Amendments bool   `json:"amendments"`
	Decompose  bool   `json:"decompose"`
}

func parseConfig(name string) options {
	o := options{Reranker: "off"}
	for i, part := range strings.Split(name, "+") {
		switch {
		case i == 0:
			o.Mode = part
		case part == "rerank":
			o.Reranker = "llm"
		case part == "decide":
			o.Reranker = "decision"
		case part == "amend":
			o.Amendments = true
		case part == "decomp":
			o.Decompose = true
		}
	}
	return o
}

var client = &http.Client{Timeout: 5 * time.Minute}

func post(url string, body any, out any) error {
	b, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func main() {
	url := flag.String("url", "http://localhost:8000", "ML service URL")
	qpath := flag.String("questions", "eval/questions.jsonl", "question set")
	out := flag.String("out", "", "markdown report path (default: stdout)")
	stage := flag.String("stage", "retrieval", "retrieval | e2e")
	configs := flag.String("configs", "bm25,vector,hybrid,hybrid+rerank,hybrid+rerank+amend", "comma-separated pipeline configs")
	flag.Parse()

	qs := loadQuestions(*qpath)
	var report strings.Builder
	switch *stage {
	case "retrieval":
		runRetrieval(*url, qs, strings.Split(*configs, ","), &report)
	case "e2e":
		runE2E(*url, qs, strings.Split(*configs, ","), &report)
	default:
		log.Fatalf("unknown stage %q", *stage)
	}
	if *out == "" {
		fmt.Print(report.String())
		return
	}
	if err := os.WriteFile(*out, []byte(report.String()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Print(report.String())
}

func loadQuestions(path string) []question {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var qs []question
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			var q question
			if err := json.Unmarshal([]byte(line), &q); err != nil {
				log.Fatalf("bad line %q: %v", line, err)
			}
			qs = append(qs, q)
		}
	}
	return qs
}

// ---------- retrieval ----------

type retrieveResp struct {
	Chunks   []hit `json:"chunks"`
	Evidence bool  `json:"evidence"`
}

func runRetrieval(url string, qs []question, configs []string, w *strings.Builder) {
	fmt.Fprintf(w, "## Качество поиска (%d вопросов)\n\n", len(qs))
	w.WriteString("| Конфигурация | DocHit@6 | MRR | SectionHit | Нет ответа: верно отклонено | Ложные отказы | Время, мс |\n")
	w.WriteString("|---|---|---|---|---|---|---|\n")
	var misses []string
	for _, cfg := range configs {
		o := parseConfig(cfg)
		var answerable, docHits, secTotal, secHits, nf, nfOK, falseRej int
		var mrr float64
		var total time.Duration
		for _, q := range qs {
			if q.Type == "invalid" {
				continue
			}
			var r retrieveResp
			start := time.Now()
			if err := post(url+"/api/retrieve", map[string]any{"question": q.Question, "options": o}, &r); err != nil {
				log.Printf("%s %s: %v", cfg, q.ID, err)
				continue
			}
			total += time.Since(start)
			if q.Type == "not_found" {
				nf++
				if !r.Evidence {
					nfOK++
				}
				continue
			}
			answerable++
			if !r.Evidence {
				falseRej++
			}
			if allDocs(q.Docs, r.Chunks, q.Type == "composite") {
				docHits++
			} else if cfg == configs[len(configs)-1] {
				misses = append(misses, fmt.Sprintf("- `%s` %s — найдено: %s", q.ID, q.Question, titles(r.Chunks)))
			}
			if rank := firstRank(q.Docs, r.Chunks); rank > 0 {
				mrr += 1 / float64(rank)
			}
			if q.Section != "" {
				secTotal++
				if sectionHit(q, r.Chunks) {
					secHits++
				}
			}
		}
		n := len(qs)
		fmt.Fprintf(w, "| `%s` | %s | %.2f | %s | %s | %s | %d |\n", cfg,
			pct(docHits, answerable), mrr/float64(max(answerable, 1)), pct(secHits, secTotal),
			pct(nfOK, nf), pct(falseRej, answerable), total.Milliseconds()/int64(max(n, 1)))
	}
	if len(misses) > 0 {
		fmt.Fprintf(w, "\nПромахи последней конфигурации (`%s`):\n\n%s\n", configs[len(configs)-1], strings.Join(misses, "\n"))
	}
}

// ---------- end-to-end ----------

type askResp struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Steps   []struct {
		Text string `json:"text"`
		Refs []int  `json:"refs"`
	} `json:"steps"`
	Conflicts []any  `json:"conflicts"`
	Sources   []hit  `json:"sources"`
	Answer    string `json:"answer"`
	Trace     struct {
		LatencyMs   int64 `json:"latencyMs"`
		DroppedRefs int   `json:"droppedRefs"`
	} `json:"trace"`
}

func runE2E(url string, qs []question, configs []string, w *strings.Builder) {
	cfg := configs[len(configs)-1]
	o := parseConfig(cfg)
	fmt.Fprintf(w, "## Качество ответов end-to-end (`%s`, %d вопросов)\n\n", cfg, len(qs))
	var statusOK, docCited, answerable, secTotal, secCited, procs, procSteps, stepsTotal, stepsCited, dropped int
	var lat []int64
	var rows []string
	for _, q := range qs {
		var r askResp
		if err := post(url+"/api/ask", map[string]any{"question": q.Question, "options": o}, &r); err != nil {
			log.Printf("%s: %v", q.ID, err)
			rows = append(rows, fmt.Sprintf("| `%s` | ошибка | %v |", q.ID, err))
			continue
		}
		lat = append(lat, r.Trace.LatencyMs)
		dropped += r.Trace.DroppedRefs
		want := map[string]string{"not_found": "not_found", "invalid": "invalid"}[q.Type]
		ok := false
		switch want {
		case "":
			ok = r.Status == "answered" || r.Status == "partial"
		default:
			ok = r.Status == want
		}
		if ok {
			statusOK++
		}
		cited := ""
		if want == "" {
			answerable++
			if allDocs(q.Docs, r.Sources, q.Type == "composite") {
				docCited++
				cited = "✓"
			} else {
				cited = "✗"
			}
			if q.Section != "" {
				secTotal++
				if sectionHit(q, r.Sources) {
					secCited++
				}
			}
		}
		if q.Type == "procedure" {
			procs++
			if len(r.Steps) > 0 {
				procSteps++
			}
		}
		for _, s := range r.Steps {
			stepsTotal++
			if len(s.Refs) > 0 {
				stepsCited++
			}
		}
		summary := strings.ReplaceAll(r.Summary, "|", "/")
		if rr := []rune(summary); len(rr) > 140 {
			summary = string(rr[:140]) + "…"
		}
		rows = append(rows, fmt.Sprintf("| `%s` | %s | %s → %s %s | %s | %d |", q.ID, q.Type, okMark(ok), r.Status, cited, summary, r.Trace.LatencyMs))
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	w.WriteString("| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(w, "| Верный статус (ответ / нет информации / некорректный ввод) | %s |\n", pct(statusOK, len(qs)))
	fmt.Fprintf(w, "| В источниках ответа есть нужный документ | %s |\n", pct(docCited, answerable))
	fmt.Fprintf(w, "| В источниках ответа есть нужный пункт | %s |\n", pct(secCited, secTotal))
	fmt.Fprintf(w, "| Процедурные вопросы с пошаговой инструкцией | %s |\n", pct(procSteps, procs))
	fmt.Fprintf(w, "| Шаги со ссылкой на источник | %s |\n", pct(stepsCited, stepsTotal))
	fmt.Fprintf(w, "| Удалено недействительных ссылок модели | %d |\n", dropped)
	if len(lat) > 0 {
		fmt.Fprintf(w, "| Задержка p50 / p90, с | %.1f / %.1f |\n", float64(lat[len(lat)/2])/1000, float64(lat[len(lat)*9/10])/1000)
	}
	w.WriteString("\n<details><summary>Ответы по вопросам</summary>\n\n| id | тип | статус | краткий ответ | мс |\n|---|---|---|---|---|\n")
	w.WriteString(strings.Join(rows, "\n"))
	w.WriteString("\n\n</details>\n")
}

// ---------- matching ----------

func docMatch(expect string, h hit) bool {
	e := strings.ToLower(expect)
	return strings.EqualFold(h.Number, expect) || strings.Contains(strings.ToLower(h.Title), e)
}

func allDocs(expect []string, hits []hit, all bool) bool {
	found := 0
	for _, e := range expect {
		for _, h := range hits {
			if docMatch(e, h) {
				found++
				break
			}
		}
	}
	if all {
		return found == len(expect)
	}
	return found > 0
}

func firstRank(expect []string, hits []hit) int {
	for i, h := range hits {
		for _, e := range expect {
			if docMatch(e, h) {
				return i + 1
			}
		}
	}
	return 0
}

func sectionHit(q question, hits []hit) bool {
	for _, h := range hits {
		matched := false
		for _, e := range q.Docs {
			if docMatch(e, h) {
				matched = true
			}
		}
		if matched && labelContains(h.Section, q.Section) {
			return true
		}
	}
	return false
}

// labelContains: "п. 2.1–2.5" contains "2.3"; "п. 3.1–3.4" contains "3".
func labelContains(label, want string) bool {
	label = strings.TrimSpace(strings.TrimPrefix(label, "п."))
	lo, hi, found := strings.Cut(label, "–")
	if !found {
		hi = lo
	}
	w := nums(want)
	l, h := nums(lo), nums(hi)
	if len(w) == 0 || len(l) == 0 {
		return false
	}
	if isPrefix(w, l) || isPrefix(w, h) {
		return true
	}
	return cmp(l, w) <= 0 && cmp(w, h) <= 0
}

func nums(s string) []int {
	var out []int
	for _, p := range strings.Split(strings.Trim(strings.TrimSpace(s), "."), ".") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}

func isPrefix(p, x []int) bool {
	if len(p) > len(x) {
		return false
	}
	for i := range p {
		if p[i] != x[i] {
			return false
		}
	}
	return true
}

func cmp(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return len(a) - len(b)
}

func titles(hs []hit) string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hs {
		if !seen[h.Title] {
			seen[h.Title] = true
			out = append(out, "«"+h.Title+"»")
		}
	}
	if len(out) == 0 {
		return "ничего"
	}
	return strings.Join(out, ", ")
}

func pct(a, b int) string {
	if b == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%% (%d/%d)", 100*float64(a)/float64(b), a, b)
}

func okMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
