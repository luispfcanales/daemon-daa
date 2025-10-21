# 🚀 Monitor de Dominios UNAMAD - Sistema Concurrente

Sistema de monitoreo de dominios con arquitectura hexagonal y modelo Actor para verificación concurrente de IPs, implementado como servicio de Windows.

---

## 📋 Descripción

Sistema de monitoreo en tiempo real que verifica concurrentemente que los dominios de la UNAMAD mantengan las IPs esperadas. Desarrollado con **Go** usando el modelo Actor de **Hollywood** para máxima concurrencia y resiliencia. Funciona como servicio de Windows para monitoreo continuo.

---

## 🎯 Características Principales

- ✅ **Verificación concurrente** de múltiples dominios simultáneamente
- ✅ **Arquitectura hexagonal** para mantenibilidad y testabilidad
- ✅ **Modelo Actor con Hollywood** para manejo seguro de concurrencia
- ✅ **Monitoreo automático** cada 30 segundos
- ✅ **Sistema de alertas** para IPs inesperadas
- ✅ **Auto-limpieza** de actores temporales
- ✅ **Logs detallados** para debugging y monitoreo
- ✅ **Servicio de Windows** para ejecución en segundo plano
- ✅ **Persistencia CSV** para configuración y resultados
- ✅ **Modo consola y servicio** para diferentes escenarios de uso

---

## 🏗️ Arquitectura del Sistema

```
Hollywood Engine
    │
    ├── MonitorActor (Coordinador)
    │       └── Crea → SingleDomainChecker (Temporales)
    │
    ├── ConsoleLogger (Suscriptor)
    │       └── Muestra resultados en consola
    │
    └── API HTTP
            └── Endpoints para consulta de estado
```

### Dominios Monitoreados

Los dominios se configuran en un archivo CSV externo (`domain_configs.csv`) con el siguiente formato:

```
dominio,ip_esperada
intranet.unamad.edu.pe,110.238.69.0
aulavirtual.unamad.edu.pe,110.238.69.0
matricula.unamad.edu.pe,110.238.69.0
```

---

## 🚀 Instalación y Ejecución

### Prerrequisitos

- Go 1.21 o superior
- Conexión a internet para resolución DNS
- Windows (para modo servicio)

### Instalación

```bash
# Clonar el proyecto
git clone https://github.com/luispfcanales/daemon-daa.git
cd daemon-daa

# Instalar dependencias
go mod tidy

# Compilar el ejecutable
go build -o monitor-dominios.exe ./cmd
```

### Modos de Ejecución

#### Modo Consola

```bash
# Ejecutar en modo consola para desarrollo y pruebas
.\monitor-dominios.exe
```

#### Instalación como Servicio de Windows

```bash
# Instalar como servicio de Windows (requiere privilegios de administrador)
.\iis-service-manager.bat install
```

#### Gestión del Servicio

```bash
# Iniciar el servicio
.\iis-service-manager.bat start

# Detener el servicio
.\iis-service-manager.bat stop

# Desinstalar el servicio
.\iis-service-manager.bat uninstall
```

---

## 📊 Salida Esperada (Modo Consola)

```
🚀 Iniciando Monitor de Dominios UNAMAD CON CONCURRENCIA
=========================================================
Directorio de trabajo: C:\ruta\al\ejecutable

🔍 Verificación inicial CONCURRENTE de dominios:

[✅ VÁLIDO] Dominio: intranet.unamad.edu.pe
   IP Esperada: 110.238.69.0
   IPs Obtenidas: [110.238.69.0]

[✅ VÁLIDO] Dominio: aulavirtual.unamad.edu.pe
   IP Esperada: 110.238.69.0
   IPs Obtenidas: [110.238.69.0]

[❌ INVÁLIDO] Dominio: matricula.unamad.edu.pe
   IP Esperada: 110.238.69.0
   IPs Obtenidas: [192.168.1.100]

🚨 WARNING: ALERTA: Dominio matricula.unamad.edu.pe tiene IPs inesperadas...

🔄 Iniciando monitoreo automático concurrente cada 30 segundos...
```

---

## 🎭 Sistema de Actores

### Actores Principales

#### 1. MonitorActor
**Función:** Coordinador principal del sistema

**Responsabilidades:**
- Crear checkers temporales para cada dominio
- Programar verificaciones periódicas
- Gestionar el estado del monitoreo
- Recibir y almacenar resultados
- Persistir resultados en CSV

#### 2. SingleDomainChecker
**Función:** Verificador temporal de un dominio específico

**Características:**
- Se auto-destruye después de procesar
- Ejecuta en su propia goroutine
- Realiza resolución DNS concurrente
- Envía alertas si detecta IPs inesperadas

#### 3. ConsoleLogger
**Función:** Presentación de resultados

**Características:**
- Suscriptor del event stream
- Muestra resultados formateados en consola
- Destaca alertas con emojis y colores
- Escribe logs en archivo para modo servicio

---

## 🔄 Flujo de Mensajes

```go
// Mensajes del sistema
CheckAllDomains{}        // → Disparador de verificación completa
CheckDomain{Name: "..."} // → Verificación de dominio específico
DomainChecked{...}       // → Resultado de verificación
Alert{...}               // → Alerta de IP inesperada
StartMonitoring{30}      // → Inicio de monitoreo automático
```

---

## ⚡ Concurrencia Demostrada

### Comparación de Rendimiento

| Escenario    | Tiempo Total | Ganancia |
|--------------|--------------|----------|
| Secuencial   | 15ms         | 1x       |
| Concurrente  | 5ms          | 3x       |

### Evidencia en Logs

```
# Inicio simultáneo
time=17:43:51.485 msg="Checking domain concurrently" domain=intranet...
time=17:43:51.485 msg="Checking domain concurrently" domain=aulavirtual...
time=17:43:51.486 msg="Checking domain concurrently" domain=matricula...

# Finalización paralela  
time=17:43:51.491 msg="Domain check completed" domain=aulavirtual...
time=17:43:51.491 msg="Domain check completed" domain=intranet...
time=17:43:51.491 msg="Domain check completed" domain=matricula...
```

---

## 🛠️ Estructura del Proyecto

```
daemon-daa/
├── cmd/
│   ├── main.go                 # Punto de entrada principal
│   └── service_windows.go      # Implementación del servicio Windows
├── internal/
│   ├── application/
│   │   ├── actors/             # Sistema de actores Hollywood
│   │   ├── api/                # API HTTP para consultas
│   │   └── events/             # Eventos del sistema
│   ├── core/
│   │   ├── domain/             # Entidades de dominio
│   │   └── ports/              # Interfaces/contratos
│   └── infrastructure/
│       ├── adapters/           # Adaptadores externos (DNS)
│       ├── repositories/       # Implementaciones de repositorio
│       └── services/           # Servicios de infraestructura
├── ecosystem.config.cjs        # Configuración para PM2 (opcional)
├── go.mod                      # Dependencias Go
├── go.sum                      # Checksums de dependencias
├── iis-service-manager.bat     # Script para gestión del servicio
└── makefile                    # Comandos de compilación
```

---

## 🔧 Configuración

### Archivos de Configuración

El sistema utiliza dos archivos CSV para su funcionamiento:

1. **domain_configs.csv**: Configuración de dominios a monitorear
   ```
   dominio,ip_esperada
   intranet.unamad.edu.pe,110.238.69.0
   ```

2. **domain_checks.csv**: Registro histórico de verificaciones
   ```
   timestamp,dominio,ip_esperada,ips_obtenidas,valido
   2023-11-15T14:30:45Z,intranet.unamad.edu.pe,110.238.69.0,[110.238.69.0],true
   ```

### Ajustar Intervalo de Monitoreo

En `cmd/main.go`:

```go
engine.Send(monitorPID, actors.StartMonitoring{Interval: 30}) // segundos
```

---

## 🧪 Testing y Desarrollo

### Ejecutar en modo debug

```bash
# Ver logs detallados de concurrencia
go run cmd/main.go
```

### Verificar logs del servicio

Los logs se guardan en el mismo directorio que el ejecutable:

```
C:\ruta\al\ejecutable\service.log
```

### Verificar concurrencia

Los logs mostrarán:
- ✅ Creación simultánea de actores
- ✅ Procesamiento paralelo de DNS
- ✅ Tiempos de ejecución similares
- ✅ Auto-destrucción de actores temporales

---

## 📈 Métricas y Monitoreo

El sistema provee:
- Duración de verificaciones por dominio
- Estado de validación (✅ VÁLIDO / ❌ INVÁLIDO)
- Alertas en tiempo real para IPs inesperadas
- Logs de concurrencia para debugging
- Historial de verificaciones en CSV
- API HTTP para consulta de estado

---

## 🚨 Manejo de Errores

- **DNS Resolution:** Error si no se puede resolver el dominio
- **IP Validation:** Alerta si las IPs no coinciden con las esperadas
- **Actor Failures:** Reinicio automático de actores fallidos
- **Graceful Shutdown:** Manejo elegante de señales de sistema
- **Servicio Windows:** Reinicio automático en caso de fallo
- **Persistencia:** Manejo de errores de lectura/escritura de archivos

---

## 🔮 Próximas Mejoras

- [x] Implementación como servicio de Windows
- [x] Persistencia en archivos CSV
- [x] API HTTP básica para consultas
- [x] Dashboard web en tiempo real
- [x] Notificaciones por email

---

## 📄 Licencia

Este proyecto es desarrollado para la **UNAMAD**.

---

## 👥 Mantenimiento

**Desarrollado con:**
- Go 1.21+
- Hollywood Actor Framework
- Arquitectura Hexagonal
- Patrón Actor Model
- Windows Service API

---

## 📞 Soporte

Para reportar problemas o sugerencias, por favor crea un issue en el repositorio:
https://github.com/luispfcanales/daemon-daa/issues

**¡Gracias por usar el Monitor de Dominios UNAMAD!** 🎓
