// Контент лендинга Физтех.Ответ — вся копия в одном месте,
// модули собираются из этих данных.

export const landingNav = [
  { id: 'how-it-works', label: 'Как работает', href: '#how-it-works' },
  { id: 'features', label: 'Возможности', href: '#features' },
  { id: 'categories', label: 'Темы', href: '#categories' },
  { id: 'faq', label: 'Вопросы', href: '#faq' },
] as const

export const landingHero = {
  brand: 'Физтех.Ответ',
  brandTagline: 'навигатор по регламентам',
  lines: ['Порядок', 'действий', 'по любому', 'регламенту'] as const,
  accentFrom: 2,
  subtitle:
    'Задайте вопрос обычным языком — ассистент найдёт нужные регламенты МФТИ, соберёт пошаговую инструкцию и покажет источники.',
  primaryCta: { label: 'Открыть ассистента', to: '/auth/login' },
  secondaryCta: { label: 'Как это работает', href: '#how-it-works' },
  scrollLabel: 'Листайте вниз',
  scrollTarget: '#hero-continue',
  suggestions: [
    'Как оформить командировку на конференцию?',
    'Как открыть ФЛС для научного проекта?',
    'Сломался рабочий компьютер — куда писать?',
  ],
}

export const landingDock = [
  { id: 'how-it-works', label: 'Как работает', href: '#how-it-works' },
  { id: 'features', label: 'Возможности', href: '#features' },
  { id: 'categories', label: 'Темы', href: '#categories' },
  { id: 'faq', label: 'Вопросы', href: '#faq' },
] as const

export const landingChatPreview = {
  question: 'Как оформить командировку для участия в научной конференции?',
  answerIntro: 'По положению о командировках порядок такой:',
  steps: [
    'Согласуйте поездку с руководителем подразделения',
    'Подайте заявку и получите аванс на расходы',
    'После возвращения сдайте авансовый отчёт',
  ],
  sources: [
    'Положение о командировках · п. 3.2',
    'Порядок выдачи авансов · п. 1.4',
  ],
  composer: 'Спросите про любую процедуру…',
  nav: ['Диалог', 'Документы', 'Инструкции', 'История'],
}

export const landingHowItWorks = {
  id: 'how-it-works',
  label: 'Как работает',
  titleLines: ['От вопроса', 'до инструкции'],
  steps: [
    {
      id: 'step-01',
      number: '01',
      title: 'Задайте вопрос',
      description:
        'Обычным языком, как коллеге — не нужно знать названия регламентов и номера приказов.',
      span: 'a',
      tone: 'deep',
    },
    {
      id: 'step-02',
      number: '02',
      title: 'Поиск по регламентам',
      description:
        'Система находит подходящие пункты в актуальной базе документов института.',
      span: 'b',
      tone: 'soft',
    },
    {
      id: 'step-03',
      number: '03',
      title: 'Пошаговая инструкция',
      description:
        'Ответ собирается в понятный порядок действий под вашу ситуацию.',
      span: 'c',
      tone: 'paper',
    },
    {
      id: 'step-04',
      number: '04',
      title: 'Источники для проверки',
      description:
        'К каждому шагу прилагается документ и пункт — любой вывод можно проверить по первоисточнику.',
      span: 'd',
      tone: 'white',
    },
  ],
}

export type FeaturesMode = 'employee' | 'admin'

export const landingFeatures = {
  id: 'features',
  label: 'Возможности',
  titleLines: ['Одна система —', 'две роли'],
  modes: [
    { id: 'employee' as const, label: 'Сотрудник' },
    { id: 'admin' as const, label: 'Администратор' },
  ],
  views: {
    employee: {
      subtitle:
        'Чат-ассистент для ежедневных вопросов: от командировки до заявки в ЦИТ — с источниками и историей диалогов.',
      steps: [
        {
          id: 'ask',
          number: '1',
          title: 'Диалог с ассистентом',
          description:
            'Вопрос свободной формой превращается в пошаговую инструкцию со ссылками на регламенты.',
          previewLabel: 'Чат с ассистентом',
        },
        {
          id: 'categories',
          number: '2',
          title: 'Категории процедур',
          description:
            'Готовые сценарии по частым темам: командировки, ФЛС, сметы, подбор персонала.',
          previewLabel: 'Каталог тем',
        },
        {
          id: 'sources',
          number: '3',
          title: 'Проверка по источникам',
          description:
            'Каждый ответ ведёт к документу и пункту — откройте первоисточник в один клик.',
          previewLabel: 'Панель источников',
        },
        {
          id: 'history',
          number: '4',
          title: 'История диалогов',
          description:
            'Прошлые ответы сохраняются — вернитесь к инструкции, когда она снова понадобится.',
          previewLabel: 'Список диалогов',
        },
      ],
    },
    admin: {
      subtitle:
        'Кабинет для тех, кто ведёт базу знаний: документы загружаются, индексируются и сразу доступны ассистенту.',
      steps: [
        {
          id: 'upload',
          number: '1',
          title: 'Загрузка документов',
          description:
            'Новые регламенты и инструкции добавляются через админ-панель в пару кликов.',
          previewLabel: 'Загрузка регламента',
        },
        {
          id: 'sync',
          number: '2',
          title: 'Синхронизация индекса',
          description:
            'Документы автоматически попадают в поисковый ML-индекс при добавлении и правках.',
          previewLabel: 'Индексация',
        },
        {
          id: 'status',
          number: '3',
          title: 'Статусы обработки',
          description:
            'Видно, какие документы проиндексированы, а какие ещё ждут обработки.',
          previewLabel: 'Таблица документов',
        },
        {
          id: 'manage',
          number: '4',
          title: 'Актуальная база знаний',
          description:
            'Редактирование и удаление устаревших редакций — ответы всегда по действующим документам.',
          previewLabel: 'Управление базой',
        },
      ],
    },
  } satisfies Record<
    FeaturesMode,
    {
      subtitle: string
      steps: {
        id: string
        number: string
        title: string
        description: string
        previewLabel: string
      }[]
    }
  >,
}

export const landingCategories = {
  id: 'categories',
  label: 'Темы',
  titleLines: ['Что уже умеет', 'находить'],
  subtitle:
    'Ассистент ориентируется в реальных процедурах института — от поездок до заявок.',
  items: [
    { id: 'trips', label: 'Командировки', size: 'lg' },
    { id: 'fls', label: 'Финансы и ФЛС', size: 'lg' },
    { id: 'budget', label: 'Сметы и бюджет', size: 'md' },
    { id: 'hr', label: 'Подбор персонала', size: 'md' },
    { id: 'it', label: 'Техническая поддержка', size: 'md' },
    { id: 'conf', label: 'Конференции', size: 'sm' },
    { id: 'borrow', label: 'Заимствование средств', size: 'md' },
    { id: 'report', label: 'Авансы и отчёты', size: 'sm' },
    { id: 'contract', label: 'Договоры', size: 'sm' },
    { id: 'access', label: 'Доступы и заявки', size: 'sm' },
    { id: 'docs', label: 'Реквизиты и счета', size: 'sm' },
  ],
}

export const landingApproach = {
  id: 'approach',
  label: 'Принципы',
  titleLines: ['Три принципа', 'работы ассистента'],
  principles: [
    {
      id: 'verifiable',
      title: 'Каждый ответ можно проверить',
      name: 'Проверяемость',
      description:
        'Инструкция всегда ссылается на документ и пункт — никаких «ответов из воздуха».',
    },
    {
      id: 'fresh',
      title: 'Только действующие регламенты',
      name: 'Актуальность',
      description:
        'Поисковый индекс синхронизируется с базой документов — устаревшие редакции не попадают в ответ.',
    },
    {
      id: 'simple',
      title: 'Вопрос без бюрократии',
      name: 'Простой язык',
      description:
        'Не нужно знать название положения или номер приказа — опишите ситуацию своими словами.',
    },
  ],
}

export const landingFaq = {
  label: 'Вопросы',
  title: 'Частые вопросы',
  items: [
    {
      id: 'faq-1',
      question: 'Нужно ли знать название регламента?',
      answer:
        'Нет. Опишите ситуацию своими словами — например, «как получить аванс на поездку». Ассистент сам найдёт подходящие документы.',
    },
    {
      id: 'faq-2',
      question: 'Можно ли доверять ответам?',
      answer:
        'Каждый шаг инструкции подкреплён источником — документом и пунктом. Вы всегда можете открыть первоисточник и проверить формулировку.',
    },
    {
      id: 'faq-3',
      question: 'Как документы попадают в систему?',
      answer:
        'Через админ-панель: регламенты загружаются, автоматически индексируются и становятся доступны поиску.',
    },
    {
      id: 'faq-4',
      question: 'Что будет, если модель недоступна?',
      answer:
        'Система работает в деградированном режиме: лексический поиск по документам и ответы-экстракты вместо генерации.',
    },
    {
      id: 'faq-5',
      question: 'Сохраняется ли история вопросов?',
      answer:
        'Да, диалоги сохраняются — к любой инструкции можно вернуться и продолжить уточнять детали.',
    },
    {
      id: 'faq-6',
      question: 'Кто поддерживает базу знаний?',
      answer:
        'Администраторы базы: они добавляют новые регламенты, следят за статусами индексации и убирают устаревшие редакции.',
    },
  ],
}

export const landingCta = {
  id: 'cta',
  title: 'Задайте первый вопрос ассистенту',
  text: 'Командировка, ФЛС, смета или заявка в ЦИТ — ответ с источниками уже ждёт.',
  primaryCta: { label: 'Зарегистрироваться', to: '/auth/register' },
  secondaryCta: { label: 'Посмотреть темы', href: '#categories' },
}

export const landingFooter = {
  brand: 'Физтех.Ответ',
  brandTagline: 'навигатор по регламентам',
  tagline:
    'Интеллектуальный помощник по внутренним регламентам и административным процессам МФТИ.',
  ctaTitle: 'Регламенты — простым языком',
  columns: [
    {
      title: 'Разделы',
      links: [
        { label: 'Как работает', href: '#how-it-works' },
        { label: 'Возможности', href: '#features' },
        { label: 'Темы', href: '#categories' },
        { label: 'Вопросы', href: '#faq' },
      ],
    },
    {
      title: 'Приложение',
      links: [
        { label: 'Открыть ассистента', href: '/auth/login' },
        { label: 'Документы', href: '/app/documents' },
        { label: 'Инструкции', href: '/app/instructions' },
      ],
    },
    {
      title: 'Администрирование',
      links: [
        { label: 'Панель администратора', href: '/admin' },
        { label: 'Загрузка документов', href: '/admin/documents/upload' },
      ],
    },
  ],
}
