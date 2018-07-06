<a name=""></a>
#  (2018-07-06)



<a name="0.6.2"></a>
## [0.6.2](https://github.com/viasite/planfix-toggl-server/compare/0.6.1...0.6.2) (2018-07-06)


### Features

* /toggl/entries/current, получение объекта текущей задачи ([7b5efcc](https://github.com/viasite/planfix-toggl-server/commit/7b5efcc))



<a name="0.6.1"></a>
## [0.6.1](https://github.com/viasite/planfix-toggl-server/compare/0.5.1...0.6.1) (2018-06-18)


### Bug Fixes

* бекапы конфигов назывались неправильно (y-d-m вместо y-m-d) ([82211f6](https://github.com/viasite/planfix-toggl-server/commit/82211f6))
* сертификаты для localhost с инструкцией по установке ([9efb124](https://github.com/viasite/planfix-toggl-server/commit/9efb124))


### Features

* /api/v1/toggl/entries теперь выдает записи за последние 7 дней ([0c6d426](https://github.com/viasite/planfix-toggl-server/commit/0c6d426))



<a name="0.6.0"></a>
# [0.6.0](https://github.com/viasite/planfix-toggl-server/compare/0.5.3...0.6.0) (2018-03-26)


### Bug Fixes

* теперь пустые SMTP поля конфига выдают ошибку, только если не настроена отправка через Планфикс API ([39fd868](https://github.com/viasite/planfix-toggl-server/commit/39fd868)), closes [#9](https://github.com/viasite/planfix-toggl-server/issues/9)


### Features

* отправка более 50 записей за один раз ([6e08d54](https://github.com/viasite/planfix-toggl-server/commit/6e08d54)), closes [#10](https://github.com/viasite/planfix-toggl-server/issues/10)
* отправка записей с группировкой по дням ([f4b36d9](https://github.com/viasite/planfix-toggl-server/commit/f4b36d9)), closes [#11](https://github.com/viasite/planfix-toggl-server/issues/11)



<a name="0.5.3"></a>
## [0.5.3](https://github.com/viasite/planfix-toggl-server/compare/0.5.2...0.5.3) (2018-03-15)


### Bug Fixes

* отправка через Планфикс API не работала ([b9b8ffc](https://github.com/viasite/planfix-toggl-server/commit/b9b8ffc))



<a name="0.5.2"></a>
## [0.5.2](https://github.com/viasite/planfix-toggl-server/compare/0.5.0...0.5.2) (2018-03-11)


### Features

* /api/v1/planfix/analitics для выбиралки аналитик ([080aae4](https://github.com/viasite/planfix-toggl-server/commit/080aae4)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)



<a name="0.5.0"></a>
# [0.5.0](https://github.com/viasite/planfix-toggl-server/compare/0.4.1...0.5.0) (2018-03-11)


### Bug Fixes

* /api/v1/toggl/planfix-task -> /api/v1/toggl/entries/planfix ([9d7d851](https://github.com/viasite/planfix-toggl-server/commit/9d7d851))
* всегда отдавать последнюю версию конфига из файла, а не из рантайма ([7456e01](https://github.com/viasite/planfix-toggl-server/commit/7456e01)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)
* запуск веб-интерфейса даже если конфиг неправильный ([2ba2645](https://github.com/viasite/planfix-toggl-server/commit/2ba2645)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)
* запуск веб-интерфейса даже если конфиг неправильный ([30d3f57](https://github.com/viasite/planfix-toggl-server/commit/30d3f57)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)


### Features

* /api/v1/config/reload, подменяет в рантайме все конфиги ([a446b0b](https://github.com/viasite/planfix-toggl-server/commit/a446b0b)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)
* https, GetEntriesV2, GetEntriesByTag, api /toggl/planfix-task/{taskID}, /toggl/planfix-task/{taskID}/last ([2cff548](https://github.com/viasite/planfix-toggl-server/commit/2cff548))
* windows icon ([aea1a57](https://github.com/viasite/planfix-toggl-server/commit/aea1a57))
* открывать веб-интерфейс в случае ошибки, проверяется соответствие toggl workspace id ([0ce06f8](https://github.com/viasite/planfix-toggl-server/commit/0ce06f8)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)
* проверка конфига на пустые поля ([f9493e0](https://github.com/viasite/planfix-toggl-server/commit/f9493e0))
* сохранение и загрузка конфига по /api/v1/config ([1471314](https://github.com/viasite/planfix-toggl-server/commit/1471314)), closes [#7](https://github.com/viasite/planfix-toggl-server/issues/7)


### BREAKING CHANGES

* урл интерфейса сменился с http://localhost:8096 на https://localhost:8097



<a name="0.4.1"></a>
## [0.4.1](https://github.com/viasite/planfix-toggl-server/compare/0.4.0...0.4.1) (2018-03-04)


### Bug Fixes

* понятная ошибка и раннее падение в случае, если поля аналитики указаны неправильно ([d13f3d5](https://github.com/viasite/planfix-toggl-server/commit/d13f3d5))
* при отправке в Планфикс через email отправлять контрольное письмо себе на ящик только при debug: true ([2534800](https://github.com/viasite/planfix-toggl-server/commit/2534800))


### Features

* DryRun режим (-dry-run в командной строке) ([de1f42a](https://github.com/viasite/planfix-toggl-server/commit/de1f42a))



<a name="0.4.0"></a>
# [0.4.0](https://github.com/viasite/planfix-toggl-server/compare/0.3.1...0.4.0) (2018-02-28)


### Bug Fixes

* логирование через переданный логгер ([36cab68](https://github.com/viasite/planfix-toggl-server/commit/36cab68))
* получение id вида работ ([32410c5](https://github.com/viasite/planfix-toggl-server/commit/32410c5))


### Features

* автоочистка тега sent из активной toggl записи ([35064b8](https://github.com/viasite/planfix-toggl-server/commit/35064b8)), closes [#4](https://github.com/viasite/planfix-toggl-server/issues/4)



<a name="0.3.1"></a>
## [0.3.1](https://github.com/viasite/planfix-toggl-server/compare/0.3.0...0.3.1) (2018-02-27)


### Bug Fixes

* non-windows build ([27f4a6c](https://github.com/viasite/planfix-toggl-server/commit/27f4a6c))



<a name="0.3.0"></a>
# [0.3.0](https://github.com/viasite/planfix-toggl-server/compare/0.2.1...0.3.0) (2018-02-27)


### Bug Fixes

* CORS headers ([4aee6f6](https://github.com/viasite/planfix-toggl-server/commit/4aee6f6))
* конфиг: apiToken -> togglApiToken, workspaceId -> togglWorkspaceId ([4f480d3](https://github.com/viasite/planfix-toggl-server/commit/4f480d3))


### Features

* получение id аналитики и ее полей из названий, кроме вида работ ([1328b17](https://github.com/viasite/planfix-toggl-server/commit/1328b17)), closes [#2](https://github.com/viasite/planfix-toggl-server/issues/2)
* скрытие консоли при запуске ([0834901](https://github.com/viasite/planfix-toggl-server/commit/0834901)), closes [#1](https://github.com/viasite/planfix-toggl-server/issues/1)


### BREAKING CHANGES

* - изменены опции в конфиге: apiToken -> togglApiToken, workspaceId -> togglWorkspaceId
- planfixAccount больше не имеет значения по умолчанию



<a name="0.2.1"></a>
## [0.2.1](https://github.com/viasite/planfix-toggl-server/compare/0.2.0...0.2.1) (2018-02-21)


### Bug Fixes

* отметка всех сгруппирванных записей toggl как sent ([b799e39](https://github.com/viasite/planfix-toggl-server/commit/b799e39)), closes [#3](https://github.com/viasite/planfix-toggl-server/issues/3)
* отправка реального юзера вместо меня ([c13f870](https://github.com/viasite/planfix-toggl-server/commit/c13f870))
* отправка реальной даты записи toggl вместо сегодня ([711af86](https://github.com/viasite/planfix-toggl-server/commit/711af86))
* отправка реальных минут вместо тестовых данных ([3b6a275](https://github.com/viasite/planfix-toggl-server/commit/3b6a275))
* сохранение sid при повторной авторизации ([4e392bb](https://github.com/viasite/planfix-toggl-server/commit/4e392bb))



<a name="0.2.0"></a>
# [0.2.0](https://github.com/viasite/planfix-toggl-server/compare/0.1.0...0.2.0) (2018-02-19)


### Bug Fixes

* email from field ([a4b030a](https://github.com/viasite/planfix-toggl-server/commit/a4b030a))
* logging tune ([1724b0f](https://github.com/viasite/planfix-toggl-server/commit/1724b0f))


### Features

* отправка через popstas/planfix-go api ([4633810](https://github.com/viasite/planfix-toggl-server/commit/4633810))
* получение user id из planfix api ([d85a647](https://github.com/viasite/planfix-toggl-server/commit/d85a647))



<a name="0.1.0"></a>
# [0.1.0](https://github.com/viasite/planfix-toggl-server/compare/1cbd9a9...0.1.0) (2018-02-15)


### Bug Fixes

* add project color ([1cbd9a9](https://github.com/viasite/planfix-toggl-server/commit/1cbd9a9))
* working entries lists ([ebdf096](https://github.com/viasite/planfix-toggl-server/commit/ebdf096))


### Features

* RunSender with SendInterval ([07e0dbc](https://github.com/viasite/planfix-toggl-server/commit/07e0dbc))



