@echo off
setlocal EnableDelayedExpansion
set SERVICE_NAME=IISMonitor
set EXE_PATH=%~dp0main.exe

if "%1"=="install" goto install
if "%1"=="start" goto start
if "%1"=="stop" goto stop
if "%1"=="uninstall" goto uninstall
if "%1"=="status" goto status
if "%1"=="logs" goto logs
if "%1"=="debug" goto debug

echo Uso: %0 [install^|start^|stop^|uninstall^|status^|logs^|debug]
echo.
echo   install   - Instalar como servicio Windows
echo   start     - Iniciar servicio
echo   stop      - Detener servicio
echo   uninstall - Eliminar servicio
echo   status    - Ver estado del servicio
echo   logs      - Ver logs del servicio
echo   debug     - Ejecutar en modo consola
goto end

:install
echo Instalando servicio %SERVICE_NAME%...
sc create %SERVICE_NAME% binPath= "%EXE_PATH%" start= delayed-auto DisplayName= "IIS Monitor Service"
sc description %SERVICE_NAME% "Servicio de monitoreo de sitios IIS"

:: Configurar tipo de inicio como DELAYED-AUTO (más tiempo)
sc config %SERVICE_NAME% start= delayed-auto

:: Aumentar tiempo de espera (30 segundos -> 120 segundos)
sc failure %SERVICE_NAME% reset= 86400 actions= restart/30000/restart/30000/restart/30000

echo Servicio instalado con inicio retardado.
goto end

:start
echo Iniciando servicio %SERVICE_NAME%...
sc start %SERVICE_NAME%
echo Esperando 15 segundos para que el servicio se inicialice...
timeout /t 15 /nobreak >nul
call :status
goto end

:stop
echo Deteniendo servicio %SERVICE_NAME%...
sc stop %SERVICE_NAME%
timeout /t 5 /nobreak >nul
call :status
goto end

:uninstall
echo Eliminando servicio %SERVICE_NAME%...
sc stop %SERVICE_NAME% >nul 2>&1
timeout /t 5 /nobreak >nul
sc delete %SERVICE_NAME%
echo Servicio eliminado.
goto end

:status
echo Estado del servicio:
sc query %SERVICE_NAME%
goto end

:logs
echo Mostrando logs:
if exist service.log (
    echo === ULTIMAS 20 LINEAS ===
    powershell "Get-Content service.log -Tail 20"
) else (
    echo No se encontró service.log
    echo Buscando en directorio: %~dp0
    dir *.log
)
goto end

:debug
echo Ejecutando en modo consola (presiona Ctrl+C para salir)...
echo Directorio: %~dp0
%EXE_PATH%
goto end

:end
pause