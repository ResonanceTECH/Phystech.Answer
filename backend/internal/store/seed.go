package store

import (
	"context"
	"log"

	"phystech-answer/backend/internal/model"
)

// SeedDemo fills the store with demo users and regulations for local development.
func (m *Memory) SeedDemo() {
	ctx := context.Background()

	ivan, err := m.CreateUser(ctx, model.User{Name: "Иван Петров", Email: "ivan.petrov@example.com"})
	if err != nil {
		log.Printf("seed: %v", err)
		return
	}
	if _, err := m.CreateUser(ctx, model.User{Name: "Мария Сидорова", Email: "maria.sidorova@example.com"}); err != nil {
		log.Printf("seed: %v", err)
	}

	docs := []model.Document{
		{
			Title:   "Правила внутреннего трудового распорядка",
			Content: "Рабочий день начинается в 9:00 и заканчивается в 18:00. Обеденный перерыв — с 13:00 до 14:00. Опоздания более чем на 15 минут фиксируются руководителем.",
			Tags:    []string{"hr", "обязательный"},
		},
		{
			Title:   "Регламент информационной безопасности",
			Content: "Пароль должен содержать не менее 12 символов и меняться раз в 90 дней. Запрещено передавать корпоративные данные через личные мессенджеры и почту.",
			Tags:    []string{"security", "обязательный"},
		},
		{
			Title:   "Политика удалённой работы",
			Content: "Сотрудник может работать удалённо до 3 дней в неделю по согласованию с руководителем. Для доступа к внутренним системам используется VPN.",
			Tags:    []string{"hr"},
		},
		{
			Title:   "Инструкция по пожарной безопасности",
			Content: "При срабатывании пожарной сигнализации покинуть здание через ближайший выход и собраться на площадке у главного входа. Лифтами пользоваться запрещено.",
			Tags:    []string{"safety", "обязательный"},
		},
	}

	for i, d := range docs {
		doc, err := m.CreateDocument(ctx, d)
		if err != nil {
			log.Printf("seed: %v", err)
			continue
		}
		if i == 0 {
			if _, err := m.MarkRead(ctx, doc.ID, ivan.ID); err != nil {
				log.Printf("seed: %v", err)
			}
		}
	}
}
