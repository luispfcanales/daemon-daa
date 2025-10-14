package templates

import (
	"fmt"
	"html"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// MonitoringTemplate genera el template HTML para el estado de monitoreo
func MonitoringTemplate(status domain.MonitoringStatus) (htmlContent string) {
	htmlContent = generateMonitoringHTML(status)
	return htmlContent
}

func generateMonitoringHTML(status domain.MonitoringStatus) string {
	statusText := "❌ DETENIDO"
	statusColor := "#dc3545"
	statusIcon := "🔴"

	if status.IsRunning {
		statusText = "✅ EN EJECUCIÓN"
		statusColor = "#28a745"
		statusIcon = "🟢"
	}

	startedAt := "N/A"
	if !status.StartedAt.IsZero() {
		startedAt = status.StartedAt.Format("02/01/2006 15:04:05")
	}

	interval := status.Interval.String()
	if interval == "0s" {
		interval = "N/A"
	}

	message := "No hay mensajes adicionales"
	if status.Message != "" {
		message = html.EscapeString(status.Message)
	}

	template := `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Daemon DAA - Estado del Monitoreo</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .container {
            background: #ffffff;
            border-radius: 10px;
            padding: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 5px solid %s;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .status-badge {
            display: inline-block;
            padding: 8px 16px;
            background-color: %s;
            color: white;
            border-radius: 20px;
            font-weight: bold;
            font-size: 14px;
        }
        .status-section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin: 20px 0;
        }
        .info-item {
            padding: 10px;
            background: #e9ecef;
            border-radius: 5px;
        }
        .info-label {
            font-weight: bold;
            color: #495057;
            font-size: 12px;
            text-transform: uppercase;
        }
        .info-value {
            font-size: 14px;
            color: #212529;
        }
        .message-box {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 5px;
            padding: 15px;
            margin: 20px 0;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #dee2e6;
            color: #6c757d;
            font-size: 12px;
        }
        .icon {
            font-size: 24px;
            margin-right: 10px;
            vertical-align: middle;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s Daemon DAA - Monitoreo</h1>
            <span class="status-badge">%s</span>
        </div>

        <div class="status-section">
            <h3>📊 Información del Estado</h3>
            <div class="info-grid">
                <div class="info-item">
                    <div class="info-label">Estado</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Intervalo</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Iniciado el</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Última actualización</div>
                    <div class="info-value">%s</div>
                </div>
            </div>
        </div>

        <div class="message-box">
            <h4>📝 Mensaje del Sistema</h4>
            <p>%s</p>
        </div>

        <div class="footer">
            <p>Este es un mensaje automático del Daemon DAA - Sistema de Monitoreo.</p>
            <p>No responda a este correo.</p>
            <p>Generado el: %s</p>
        </div>
    </div>
</body>
</html>`

	return fmt.Sprintf(
		template,
		statusColor,
		statusColor,
		statusIcon,
		statusText,
		statusText,
		interval,
		startedAt,
		time.Now().Format("02/01/2006 15:04:05"),
		message,
		time.Now().Format("02/01/2006 15:04:05"),
	)
}
