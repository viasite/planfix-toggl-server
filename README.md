# Planfix-Toggl server
[![Build Status](https://travis-ci.org/viasite/planfix-toggl-server.svg?branch=master)](https://travis-ci.org/viasite/planfix-toggl-server)
[![Coverage Status](https://coveralls.io/repos/github/viasite/planfix-toggl-server/badge.svg?branch=master)](https://coveralls.io/github/viasite/planfix-toggl-server?branch=master)
[![Scrutinizer Code Quality](https://scrutinizer-ci.com/g/viasite/planfix-toggl-server/badges/quality-score.png?b=master)](https://scrutinizer-ci.com/g/viasite/planfix-toggl-server/?branch=master)

Интеграция Planfix и Toggl, отправляет данные из Toggl в Планфикс, сделан для того, чтобы избавить людей,
трекающих свою активность в Toggl, от ручного переноса данных в Планфикс.

Вольное описание в [блоге](http://blog.popstas.ru/blog/2018/03/01/planfix-toggl-integration/), конкретное - ниже.

![tray demo](assets/tray-demo.png)

## Правила
Вы должны указывать записям в Toggl id задач Планфикса в виде тегов, например, 12345.

При запуске скрипт получает последние 50 записей, находит среди них записи с id задач и отправляет на email задачи.
Если из 50 записей нашлось, что отправить, запрашиваются следующие 50 записей, так может продолжаться до 1000 записей (20 страниц).

После успешной отправки к записи добавляется тег `sent`, чтобы не отправить повторно.

Если запустить уже отправленную запись toggl, из нее в течение минуты будет автоматом стерт тег `sent`.

Записи, сделанные не вами (в командном аккаунте) игнорируются.



## Установка
1. Скачайте [последний релиз](https://github.com/viasite/planfix-toggl-server/releases)  
   [Windows](https://github.com/viasite/planfix-toggl-server/releases/download/0.8.0/planfix-toggl-windows.zip), [Linux](https://github.com/viasite/planfix-toggl-server/releases/download/0.8.0/planfix-toggl-linux.zip), [MacOS](https://github.com/viasite/planfix-toggl-server/releases/download/0.6.4/planfix-toggl-darwin.zip)
2. Установите сертификат certs/server.crt в систему как доверенный корневой, [подробнее](certs/)
3. Запустите planfix-toggl-server.exe
4. Откроется веб-интерфейс, заполните настройки, нажмите все кнопки "Проверить"

### Linux
На Ubuntu я не проверял, но должно работать.

### MacOS
На MacOS никогда не проверял, но должно работать.
Последняя версия, которая собиралась на MacOS - [0.6.4](https://github.com/viasite/planfix-toggl-server/releases/download/0.6.4/planfix-toggl-darwin.zip),
после этого была добавлена иконка в трей, которую проблематично компилировать под макось, а у меня нет мотивации это делать (сижу на Windows).


## Использование
Просто запустите.

Для облегчения проставления тегов в записи было дописано официальное расширение Toggl Chrome,
скорее всего, pull request никогда не примут, поэтому я форкнул расширение и опубликовал, для
[Chrome](https://chrome.google.com/webstore/detail/toggl-button-planfix-edit/hkhchfdjhfegkhkgjongbodaphidfmcl) и
[Firefox](https://addons.mozilla.org/ru/firefox/addon/toggl-button-planfix/).

Кое-какую информацию можно смотреть через веб-интерфейс на https://localhost:8097



## Постоянное использование
Добавьте задачу в планировщик заданий Windows:

- при включении компьютера
- путь к planfix-toggl-server.exe
- аргументы: `-no-console`
- укажите рабочую папку
- отложить запуск на 1 минуту
- запускать сразу, если время запуска пропущено



## Настройка

### Конфиг
В конфиге `config.default.yml` указаны некоторые настройки по умолчанию, если хотите переопределить их, скопируйте в `config.yml`.

На данный момент (0.4) существует два способа отправки данных в Планфикс: через email и через Планфикс API,
если есть возможность, пользуйтесь вторым способом, он надежнее и требует меньше настроек. Режим активируется,
если указаны логин и пароль от Планфикса.
  
Настройки для всех:

- `togglSentTag` - тег, которым помечаются отправленные toggl-записи
- `togglApiToken` - токен Toggl, в настройках profile в Toggl
- `togglWorkspaceId` - посмотрите в url вашего workspace в Toggl
- `planfixAccount` - поддомен вашего Планфикс аккаунта
- `sendInterval` - период отправки данных в Планфикс, в минутах

Настройки для отправки через email:

- `smtpHost`, `smtpPort`, `smtpSecure` - настройки SMTP для отправки. Нужно настроить на свой рабочий ящик, который связан с аккаунтом в Планфиксе
- `smtpLogin`, `smtpPassword` - логин и пароль от вашей почты (настройки по умолчанию для Яндекс почты)
- `smtpEmailFrom` - должен совпадать с email вашего аккаунта в Планфиксе и у smtp должно быть право отправлять письма от этого имени
- `planfixAnaliticTypeValue` - как называется поминутная аналитика, которую вы хотите проставлять в Планфикс
- `planfixAuthorName` - ваше Имя Фамилия в Планфиксе

Настройки для отправки через Планфикс API:
- `planfixApiKey` - приватный API ключ, есть у владельца аккаунта Планфикса
- `planfixApiUrl` - URL API, для аккаунтов в России он будет другим
- `planfixUserName`, `planfixUserPassword` - ваши логин и пароль в Планфиксе

Также, нужно описать все поля аналитики, которые будут заполняться:
- `planfixAnaliticName` - выработка
- `planfixAnaliticTypeName` - вид работы (справочник работ)
- `planfixAnaliticTypeValue` - поминутная работа программиста (вид работы)
- `planfixAnaliticCountName` - кол-во (минут)
- `planfixAnaliticCommentName` - комментарий / ссылка (текст, описание аналитики)
- `planfixAnaliticDateName` - дата (день, без времени)
- `planfixAnaliticUsersName` - сотрудник (мультиполе сотрудников)

Для осторожных: все данные, включая пароли, отправляются только на `planfixApiUrl`, все исходники открыты,
из внешних зависимостей используется только go-toggl.

Прочие настройки:
- `debug` - включает больше вывода (которого и без того много)
- `logFile` - лог, туда отправляется все то же, что и в консоль
- `dryRun` - тестовый режим, без реальной отправки данных в Планфикс



### Аргументы командной строки:

```
  -dry-run
    	Don't actually change data
  -no-console
    	Hide console window
```



### Настройка Планфикса для обработки email
Управление аккаунтом -> Работа с помощью e-mail -> Правила обработки для задач -> Новое правло

У вас конечно будут другие названия полей, если вы не работаете в viasite.

#### Параметры отбора:
- Тема письма содержит текст: `@toggl`
- Содержание письма содержит слово: `time:`
#### Операции:
- Добавить аналитику: Выработка
- Вид работы: `Вид работы:` (до конца строки)
- Дата: `Дата:` (до конца строки)
- Кол-во: `time:` (до конца строки)
- Сотрудник: `Автор:` (до конца строки)
#### Также
- Удалить всё, начиная с метки: `Вид работы:` (в содержании письма)
