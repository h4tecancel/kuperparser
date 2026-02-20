# kuperparser

Парсер каталога/товаров **kuper.ru** с двумя режимами работы:

1) **API** (`cmd/kuperparser-api`) — поднимает сервер и отдаёт JSON по эндпойнтам:
   - `GET /categories?storeID=...` вывод категорий магазина
   - `GET /products?storeID=...&categoryID=...` вывод товаров определенного магазина по адресу и категории

2) **CLI** (`cmd/kuperparser + cmd/kuperparser-stores-scan`) — выгружает товары категории в JSON файл в корневую папку / выводит примеры айдишников магазинов + адрес(на данный момент довольно костыльный).

P.S. CLI доступен только под local окружение

Проект включает:
- конфиг профилей окружения `env: local|dev|prod`
- логирование через `slog` (text/json)
- прокси: disabled/list/rotation
- retries + ограничение параллельности запросов
- поддержка запуска с флагами и для api, и для cli
- возможные store id для примера выгрузки определенных адресов
Тестовые выводы

---
```bash
curl "http://localhost:7891/categories?storeID=86"
```
Response
```json
{
    "categories": [
        {
            "id": 69085,
            "parent_id": 0,
            "name": "Скидки и акции",
            "slug": "skidki-i-aktsii",
            "products_count": 33521,
            "has_children": true,
            "level": 0
        },
        {
            "id": 11320576,
            "parent_id": 0,
            "name": "Праздник для любимых",
            "slug": "prazdnik-dlya-lyubimikh-8c5f11b",
            "products_count": 5115,
            "has_children": true,
            "level": 0
        },
        {
            "id": 297584,
            "parent_id": 0,
            "name": "Масленица",
            "slug": "maslenitsa-f83a123",
            "products_count": 3407,
            "has_children": true,
            "level": 0
        },
        {
            "id": 74553,
            "parent_id": 0,
            "name": "Алкоголь",
            "slug": "alkogol-cc137d2",
            "products_count": 4769,
            "has_children": true,
            "level": 0
        },
        {
            "id": 75617,
            "parent_id": 0,
            "name": "Готовая еда",
            "slug": "gotovaya-eda-copy",
            "products_count": 761,
            "has_children": true,
            "level": 0
        },
        {
            "id": 41328,
            "parent_id": 0,
            "name": "Овощи, фрукты, зелень, орехи",
            "slug": "ovoshchi-frukti-orekhi",
            "products_count": 2233,
            "has_children": true,
            "level": 0
        },
        {
            "id": 166211,
            "parent_id": 0,
            "name": "Молоко, сыр, яйца, растительные продукты",
            "slug": "moloko-sir-yajtsa-rastitelnie-produkti-c44b0ed",
            "products_count": 4453,
            "has_children": true,
            "level": 0
        },
        {
            "id": 43519,
            "parent_id": 0,
            "name": "Мясо, птица",
            "slug": "myaso-ptitsa-40ebce3",
            "products_count": 1497,
            "has_children": true,
            "level": 0
        }
        ...
    ],
    "count": 36,
    "fetched_at": "2026-02-19T06:50:21Z",
    "store_id": 86
}
```



```bash
curl "localhost:7891/products?storeID=86&categoryID=68499"
```
Response
```json
{
    "fetched_at": "2026-02-19T06:48:16Z",
    "store": {
        "id": 86,
        "name": "METRO, Московская область, Балашиха, улица Советская, 60",
        "address": "МО, Балашиха, Московская область, Балашиха, улица Советская, 60",
        "retailer_name": "METRO"
    },
    "category": {
        "id": 68499,
        "slug": "vsyo-dlya-remonta-giper"
    },
    "products": [
        {
            "name": "Мойка высокого давления Karcher k 5 power control",
            "price": "49999",
            "url": "https://kuper.ru/products/15167224-moyka-vysokogo-davleniya-karcher-k-5-power-control-e7f30cc"
        },
        {
            "name": "Мойка высокого давления Karcher k 4 power control",
            "price": "43999",
            "url": "https://kuper.ru/products/15167216-moyka-vysokogo-davleniya-karcher-k-4-power-control"
        },
        {
            "name": "Мойка высокого давления Karcher K 3",
            "price": "15449.01",
            "url": "https://kuper.ru/products/151576-moyka-vysokogo-davleniya-karcher-k-3"
        },
        {
            "name": "Ящик Blocker Boombox 24\" для инструмента",
            "price": "2699.01",
            "url": "https://kuper.ru/products/151166-yaschik-blocker-boombox-24-dlya-instrumenta"
        },
        {
            "name": "Брюки повара унисекс Metro Professional чёрно-белые в ассортименте (размер по наличию)",
            "price": "1599.01",
            "url": "https://kuper.ru/products/49923069-bryuki-povara-uniseks-metro-professional-chyorno-belye-v-assortimente-razmer-po-nalichiyu"
        },
        {
            "name": "Сумка для инструментов Zalger 33 x 22,5 x 24 см",
            "price": "1729",
            "url": "https://kuper.ru/products/48594873-sumka-dlya-instrumentov-zalger-33-x-22-5-x-24-sm"
        }
    ],
        "count": 36,
            "fetched_at": "2026-02-19T06:50:21Z",
            "store_id": 86
        }
```

---
Примечание: некоторые верхнеуровневые категории могут быть "витринными/подборками" и не поддерживать выдачу товаров через /products. 
Проект по умолчанию может скрывать "root-leaf" категории (parent_id=0 && has_children=false), чтобы не отдавать неподдерживаемые варианты. Также по мне как конфиг не до идеала выточен, но это просто быстрое решение.
