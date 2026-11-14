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
	sections := map[string]bool{}
	for _, c := range chunks {
		sections[c.Section] = true
	}
	for _, want := range []string{"2. Инициирование командировки", "4. Авансирование и отчётность"} {
		if !sections[want] {
			t.Errorf("missing section %q, got %v", want, sections)
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
		if len([]rune(c.Text)) > maxChunkRunes+overlapRunes+200 {
			t.Errorf("chunk too large: %d runes", len([]rune(c.Text)))
		}
	}
}
