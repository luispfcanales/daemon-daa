# 🚀 Monitor de Dominios UNAMAD - Sistema Concurrente

Sistema de monitoreo de dominios con arquitectura hexagonal y modelo Actor para verificación concurrente de IPs.

---

## 📋 Descripción

Sistema de monitoreo en tiempo real que verifica concurrentemente que los dominios de la UNAMAD mantengan las IPs esperadas. Desarrollado con **Go** usando el modelo Actor de **Hollywood** para máxima concurrencia y resiliencia.

---

## 🎯 Características Principales

- ✅ **Verificación concurrente** de múltiples dominios simultáneamente
- ✅ **Arquitectura hexagonal** para mantenibilidad y testabilidad
- ✅ **Modelo Actor con Hollywood** para manejo seguro de concurrencia
- ✅ **Monitoreo automático** cada 30 segundos
- ✅ **Sistema de alertas** para IPs inesperadas
- ✅ **Auto-limpieza** de actores temporales
- ✅ **Logs detallados** para debugging y monitoreo

---

## 🏗️ Arquitectura del Sistema

```
Hollywood Engine
    │
    ├── MonitorActor (Coordinador)
    │       └── Crea → SingleDomainChecker (Temporales)
    │
    └── ConsoleLogger (Suscriptor)
            └── Muestra resultados en consola
```

### Dominios Monitoreados

| Dominio                      | IP Esperada    |
|------------------------------|----------------|
| intranet.unamad.edu.pe       | 110.238.69.0   |
| aulavirtual.unamad.edu.pe    | 110.238.69.0   |
| matricula.unamad.edu.pe      | 110.238.69.0   |

---

## 🚀 Instalación y Ejecución

### Prerrequisitos

- Go 1.21 o superior
- Conexión a internet para resolución DNS

### Instalación

```bash
# Clonar el proyecto
git clone <repository-url>
cd monitor-dominios

# Instalar dependencias
go mod tidy

# Ejecutar el sistema
go run cmd/main.go
```

### Ejecución con logs detallados

```bash
# Para ver toda la concurrencia en acción
go run cmd/main.go
```

---

## 📊 Salida Esperada

```
🚀 Iniciando Monitor de Dominios UNAMAD CON CONCURRENCIA
=========================================================

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
monitor-dominios/
├── cmd/
│   └── main.go                 # Punto de entrada
├── internal/
│   ├── core/
│   │   ├── domain/             # Entidades de dominio
│   │   └── ports/              # Interfaces/contratos
│   ├── infrastructure/
│   │   ├── adapters/           # Adaptadores externos (DNS)
│   │   └── repositories/       # Implementaciones de repositorio
│   └── application/
│       └── actors/             # Sistema de actores Hollywood
└── go.mod
```

---

## 🔧 Configuración

### Modificar Dominios Monitoreados

Editar `internal/infrastructure/repositories/domain_repository.go`:

```go
configs: []domain.DomainConfig{
    {Domain: "intranet.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
    {Domain: "aulavirtual.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
    // Agregar nuevos dominios aquí
    {Domain: "nuevodominio.unamad.edu.pe", ExpectedIP: "110.238.69.0"},
},
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

---

## 🚨 Manejo de Errores

- **DNS Resolution:** Error si no se puede resolver el dominio
- **IP Validation:** Alerta si las IPs no coinciden con las esperadas
- **Actor Failures:** Reinicio automático de actores fallidos
- **Graceful Shutdown:** Manejo elegante de señales de sistema

---

## 🔮 Próximas Mejoras

- [ ] Persistencia en base de datos
- [ ] API REST para consultas
- [ ] Dashboard web en tiempo real
- [ ] Notificaciones por email/telegram
- [ ] Métricas Prometheus
- [ ] Configuración via archivo YAML

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

---

## 📞 Soporte

Para reportar problemas o sugerencias, por favor crea un issue en el repositorio.

**¡Gracias por usar el Monitor de Dominios UNAMAD!** 🎓
