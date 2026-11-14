package main

import (
	"fmt"
	"os"
	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/relations"
)

func main() {
	data, _ := os.ReadFile("data/parsed/460-1 Открытие ФЛС_приказ об изменении в приказе на открытие ФЛС.txt")
	rels := relations.Extract(model.Document{ID: "x", Content: string(data)})
	fmt.Printf("relations: %d\n", len(rels))
	for _, r := range rels {
		fmt.Printf("- %s → №%s от %s | %.80s\n", r.RelationType, r.TargetDocumentNumber, r.TargetDocumentDate, r.Evidence)
	}
}
