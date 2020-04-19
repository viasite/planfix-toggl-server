go get github.com/mitchellh/gox
go get github.com/akavel/rsrc # windows icon generate

## Linux
apt install libgtk-3-dev libappindicator3-dev libwebkit2gtk-4.0-dev

npm version


## Windows

### Ошибка [Unable to create main window: TTM_ADDTOOL failed](https://github.com/getlantern/systray/issues/124)
Надо создать planfix-toggl-server.exe.manifest:
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="Planfix-Toggl server" type="win32"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
    <application xmlns="urn:schemas-microsoft-com:asm.v3">
        <windowsSettings>
            <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2, PerMonitor</dpiAwareness>
            <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">True</dpiAware>
        </windowsSettings>
    </application>
</assembly>
```

Если при запуске сильные тормоза пару секунд, а потом вылетает,
значит, скорее всего, приложение запускается с сетевого диска,
так больше нельзя.


Для отладки в GoLand надо выбрать предсказуемую папку в debug configuration (./build)
и положить туда одноимённый манифест, например:
`build/planfix_toggl_server__dry_run.exe.manifest`

Чтобы не появлялась консоль, надо билдить с `-ldflags "-H=windowsgui"`, но при отладке не стоит отключать.

### Unable to make systray icon visible