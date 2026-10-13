# IDEF0: модель, вложенность и подключение

IDEF0 показывает **что делает функция** и какие входы, правила, результаты и исполнители с ней связаны. Блок обозначает функцию, стрелки — её интерфейс. Декомпозиция раскрывает функцию в несколько дочерних блоков. Так удобно описывать бизнес-процессы Careeria на разных уровнях, не смешивая их с порядком экранных действий или технической структурой сервисов.

Схемы строятся из текста через [IDEF0-SVG](https://github.com/jimmyjazz/IDEF0-SVG). В репозитории закреплён исходный Ruby-генератор; локальный адаптер добавляет цвета и выбор нескольких представлений одной модели.

## Файлы одной модели

| Файл | Роль |
| --- | --- |
| `docs/idef0-svg/models/hiring.idef0` | Единственный редактируемый источник функций, стрелок и вложенности процесса найма. |
| `docs/idef0-svg/models/hiring.idef0.json` | Выбор функции и режима для каждого вида, общие и отдельные цвета. Необязателен. |
| `docs/idef0-svg/models/hiring.svg` | Сгенерированная главная схема. |
| `docs/idef0-svg/models/hiring-review-applications.svg` | Сгенерированный вложенный вид из того же `.idef0`. |
| Markdown-страница | Объясняет цель и границы модели, подключает SVG и связывает уровни ссылками. |

Сохраняйте собственные модели в `docs/idef0-svg/models/`. Имя файла делайте коротким и понятным по процессу, например `candidate-assessment.idef0`.

## Как записать функцию и ICOM-стрелки

Строка DSL имеет вид `Function predicate Concept`. Имя функции и понятия пишутся по-английски с заглавной буквы, предикат — строчными буквами. Каждая связь занимает отдельную строку; строка с `#` в начале — комментарий.

| Предикат | Вид стрелки | Смысл | Сторона блока |
| --- | --- | --- | --- |
| `receives` | Input | Что функция получает и преобразует. | Слева |
| `respects` | Control | Правило или ограничение, которое определяет выполнение. | Сверху |
| `produces` | Output | Результат выполнения. | Справа |
| `requires` | Mechanism | Кто или что выполняет работу. | Снизу |
| `is composed of` | Декомпозиция | Дочерняя функция следующего уровня. | Создаёт блок внутри выбранного вида |

```text
Manage Hiring receives Hiring Request
Manage Hiring respects Hiring Policy
Manage Hiring requires Recruitment Team
Manage Hiring produces Hiring Decision
Manage Hiring is composed of Review Applications
```

Повторяйте одно и то же имя понятия **точно**, если оно связывает блоки: например, `Screen Applications produces Eligible Applications` и `Evaluate Candidates receives Eligible Applications`. Разные названия образуют разные стрелки. Разным функциям давайте разные имена: одинаковые имена генератор объединяет.

## Как устроена вложенность

Для каждого блока следующего уровня добавьте отдельную строку `Parent is composed of Child`. Затем опишите ICOM-стрелки дочерних функций. Дочернюю функцию можно раскрыть ещё на один уровень тем же способом.

```text
Manage Hiring is composed of Review Applications
Review Applications is composed of Screen Applications
Review Applications is composed of Evaluate Candidates
Review Applications is composed of Prepare Shortlist
Screen Applications receives Candidate Applications
Screen Applications produces Eligible Applications
Evaluate Candidates receives Eligible Applications
Evaluate Candidates produces Evaluations
Prepare Shortlist receives Evaluations
Prepare Shortlist produces Shortlist
```

В [модели найма](../idef0-svg/models/hiring.idef0) это дерево выглядит так:

```text
Manage Hiring
├── Publish Vacancy
├── Review Applications
│   ├── Screen Applications
│   ├── Evaluate Candidates
│   └── Prepare Shortlist
└── Make Decision
```

Не создавайте циклов вложенности. Перед публикацией вручную сверяйте стрелки родителя и его декомпозиции: входы, управления, выходы и механизмы на границе должны сохранять смысл. Исходный IDEF0-SVG эту балансировку не проверяет.

## Какие виды создаёт генератор

| Режим | Что показывает | Когда выбирать |
| --- | --- | --- |
| `decompose` | Непосредственные дочерние функции выбранного процесса. | Основной вид для перехода на следующий уровень. |
| `focus` | Один блок с его ICOM-стрелками. | Контекст функции без дочерних блоков. |
| `schematic` | Конечные функции выбранной ветки вместе. | Общий обзор ветки, когда нужен один SVG. |

Без JSON создаётся один `<имя>.svg` для корневой функции в режиме `decompose`. Чтобы показать несколько уровней без копирования DSL, положите рядом `<имя>.idef0.json`:

```json
{
  "mode": "decompose",
  "process": "Manage Hiring",
  "views": [
    { "name": "review-applications", "mode": "decompose", "process": "Review Applications" }
  ]
}
```

Первый вид создаст `hiring.svg`, дополнительный — `hiring-review-applications.svg`. `process` должен точно совпадать с именем функции в `.idef0`. `name` дополнительного вида — уникальная последовательность строчных латинских слов, цифр и дефисов. Для функции без детей выбирайте `focus`.

| Поле JSON | Значение |
| --- | --- |
| `mode` | Режим основного `<имя>.svg`; по умолчанию `decompose`. |
| `process` | Функция для основного SVG; без поля выбирается корень модели. |
| `views` | Список дополнительных SVG, генерируемых из того же `.idef0`. |
| `views[].name` | Уникальный суффикс имени файла: `<имя>-<name>.svg`. |
| `views[].mode` | Режим дополнительного SVG; по умолчанию `decompose`. |
| `views[].process` | Функция, которую показывает дополнительный SVG. |
| `defaults` | Общая палитра блоков, стрелок и подписей. |
| `blocks`, `arrows` | Переопределения цветов по точному имени функции или понятия. |

## Цвета блоков и стрелок

Цвета задаются в том же JSON. `defaults` действует на всю схему; `blocks` и `arrows` переопределяют его для элементов по точному имени из DSL.

```json
{
  "defaults": {
    "arrow": "#43566D",
    "arrow_text": "#26384B",
    "block_fill": "#F3F7FB",
    "block_stroke": "#27466A",
    "block_text": "#172B41"
  },
  "blocks": {
    "Make Decision": { "fill": "#EAF6EE", "text": "#14532D" }
  },
  "arrows": {
    "Hiring Decision": { "stroke": "#15803D", "text": "#14532D" }
  }
}
```

| Поле | Что окрашивает |
| --- | --- |
| `defaults.arrow` / `arrows.<имя>.stroke` | Линию и наконечник стрелки. |
| `defaults.arrow_text` / `arrows.<имя>.text` | Подпись стрелки. |
| `defaults.block_fill` / `blocks.<имя>.fill` | Фон блока. |
| `defaults.block_stroke` / `blocks.<имя>.stroke` | Контур блока. |
| `defaults.block_text` / `blocks.<имя>.text` | Текст функции в блоке. |

Используйте `#RGB`, `#RRGGBB` или имя CSS-цвета. Без настройки остаётся чёрно-белый стиль IDEF0-SVG. Если одно понятие показано несколькими стрелками, его настройка цвета применяется ко всем стрелкам с этим именем. [Полный JSON примера](../idef0-svg/models/hiring.idef0.json) показывает виды и цвета вместе.

## Как происходит синхронизация

Генерация идёт в одном направлении: **`.idef0` + `.idef0.json` → `.svg` → Markdown-страница**. Перед сборкой MkDocs хук `docs/tools/idef0_render.py` находит модели в `docs/idef0-svg/models/`, запускает закреплённый IDEF0-SVG и записывает SVG только при изменении результата. `make docs` повторяет сборку при правке модели или настроек; `make docs-build` делает то же при статической сборке. `make idef0-render` обновляет SVG без сборки всего сайта.

SVG не является источником модели: ручная правка пропадёт при следующей генерации. После изменения `.idef0` или JSON проверьте все виды и сохраните в Git исходник, настройки и SVG. При удалении записи из `views` удалите прежний SVG вручную: генератор не удаляет файлы исчезнувших видов.

```sh
make idef0-render
make docs
make docs-test
```

## Как подключить схему на страницу

В Markdown используйте обычную относительную ссылку на SVG. Путь считается от текущего `.md`-файла. Для этой страницы он выглядит так:

```markdown
![Декомпозиция процесса найма](../idef0-svg/models/hiring.svg)

[Вложенный уровень](../idef0-svg/models/hiring-review-applications.svg) ·
[Исходник модели](../idef0-svg/models/hiring.idef0)
```

![Декомпозиция процесса найма](../idef0-svg/models/hiring.svg)

На странице с диаграммой напишите, какую функцию она раскрывает, что находится вне её границы, что значат ICOM-стрелки и как читать дочерние блоки. Дайте переходы к родительскому и дочерним уровням. Если уровень должен иметь собственный пункт меню, создайте для него `.md`-страницу и подключите нужный SVG там; пример структуры страницы находится в разделе [«Структура и страницы»](01-structure.md#page-structure).
