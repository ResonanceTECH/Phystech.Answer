package chunk

import (
	"strings"
	"testing"

	"phystech-answer/ml/internal/model"
)

const sampleReg = `1. Общие положения

1.1. Настоящий регламент определяет порядок командирования работников.

1.2. Командировкой признаётся поездка по распоряжению работодателя.

2. Инициирование командировки

2.1. Направление оформляется распоряжением руководителя.

2.2. Для рассмотрения вопроса работник подаёт заявление не позднее чем за 14 календарных дней до даты начала командировки. В заявлении указываются цель, место, даты, источник финансирования и обоснование необходимости поездки.

3. Согласование и оформление

3.1. Руководитель подразделения согласовывает заявку с курирующим проректором, если поездка связана с расходами из внебюджетных средств или участием в конференции.

3.2. Оформление командировки на поезде или самолёте производится через сервис «Проезд» не позднее чем за 10 рабочих дней до отправления.

4. Авансирование и отчётность

4.1. Аванс выдаётся по заявлению в бухгалтерии не позднее чем за 3 рабочих дня до выезда.

4.2. По возвращении работник в течение 3 рабочих дней сдаёт авансовый отчёт с подтверждающими документами.`

func TestSplitKeepsPunkti(t *testing.T) {
	doc := model.Document{ID: "d1", Title: "Регламент командировок", Content: sampleReg}
	chunks := Split(doc)
	if len(chunks) == 0 {
		t.Fatal("no chunks produced")
	}

	joined := ""
	for _, c := range chunks {
		joined += c.Text + "\n"
	}

	// Every пункт's key phrase must survive chunking.
	mustContain := []string{
		"1.1. Настоящий регламент",
		"2.2. Для рассмотрения вопроса",
		"3.1. Руководитель подразделения",
		"4.2. По возвращении",
	}
	for _, phrase := range mustContain {
		if !strings.Contains(joined, phrase) {
			t.Errorf("chunking lost text %q", phrase)
		}
	}

	// Sections should be attributed.
	headings := map[string]bool{}
	for _, c := range chunks {
		headings[c.Heading] = true
		if c.Section == "" {
			t.Errorf("chunk without citation label: %q", c.Text)
		}
		if !strings.Contains(c.Context, "Регламент командировок") {
			t.Errorf("context header missing doc title: %q", c.Context)
		}
	}
	for _, want := range []string{"2. Инициирование командировки", "4. Авансирование и отчётность"} {
		if !headings[want] {
			t.Errorf("missing heading %q, got %v", want, headings)
		}
	}
}

func TestSplitLongDocument(t *testing.T) {
	var b strings.Builder
	b.WriteString("1. Общие положения\n\n")
	for i := 1; i <= 200; i++ {
		b.WriteString("1.1. Это длинный пункт регламента номер ")
		b.WriteString(strings.Repeat("очень ", 10))
		b.WriteString("с подробным описанием требований.\n\n")
	}
	doc := model.Document{ID: "d2", Title: "Длинный", Content: b.String()}
	chunks := Split(doc)
	if len(chunks) < 5 {
		t.Fatalf("expected many chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if len([]rune(c.Text)) > maxChunkRunes {
			t.Errorf("chunk too large: %d runes", len([]rune(c.Text)))
		}
	}
}

func TestLabelsAndLoneNumbers(t *testing.T) {
	text := "РЕГЛАМЕНТ бронирования\n1. Общие положения\n1.1.\nНастоящий регламент устанавливает порядок.\n1.2. Зал бронируется заранее.\n2. Порядок бронирования\n2.1. Подать заявку за 14 дней."
	chunks := Split(model.Document{ID: "d", Title: "Регламент", Number: "1-1", Date: "2025-01-01", Content: text})
	var labels []string
	for _, c := range chunks {
		labels = append(labels, c.Section+" @ "+c.Heading)
	}
	joined := strings.Join(labels, " | ")
	for _, want := range []string{"п. 1–1.2 @ 1. Общие положения", "п. 2–2.1 @ 2. Порядок бронирования"} {
		if !strings.Contains(joined, want) {
			t.Errorf("want %q in %q", want, joined)
		}
	}
	if !strings.Contains(chunks[0].Context, "№ 1-1 от 2025-01-01") {
		t.Errorf("context: %q", chunks[0].Context)
	}
}

func TestHeadingWithDotAndNoSpaceNumbers(t *testing.T) {
	text := "1. Общие положения\n1.1. Текст первого пункта достаточно длинный.\n2. Порядок бронирования Концертного зала.\n2.1. График формируется в начале года.\n2.3. Служебная записка подаётся за два месяца.\n3. Подготовка мероприятия\n3.1.При планировании учитывать параметры зала."
	chunks := Split(model.Document{ID: "d", Title: "Регламент", Content: text})
	got := map[string]string{}
	for _, c := range chunks {
		got[c.Heading] = c.Section
	}
	if got["2. Порядок бронирования Концертного зала."] != "п. 2–2.3" {
		t.Errorf("section 2: %v", got)
	}
	if got["3. Подготовка мероприятия"] != "п. 3–3.1" {
		t.Errorf("section 3: %v", got)
	}
}

func TestLabelMonotonic(t *testing.T) {
	bs := []*block{{number: "2"}, {number: "2.2"}, {number: "1"}, {number: "11"}}
	if l := label(bs); l != "п. 2–2.2" {
		t.Errorf("label = %q", l)
	}
}

func TestWrappedLineIsNotHeading(t *testing.T) {
	text := "19. В случае задержки в пути работник представляет документ.\n20. В случае наступления в период командировки временной нетрудоспособности работник\nобязан незамедлительно уведомить работодателя.\n3. Срок заимствования может составлять от 1 до 4 месяцев."
	for _, c := range Split(model.Document{ID: "d", Title: "Т", Content: text}) {
		if strings.HasPrefix(c.Heading, "20.") || strings.HasPrefix(c.Heading, "3. Срок") {
			t.Errorf("sentence taken as heading: %q", c.Heading)
		}
	}
}
