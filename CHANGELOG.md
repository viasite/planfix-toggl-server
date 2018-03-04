<a name=""></a>
#  (2018-03-04)



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



