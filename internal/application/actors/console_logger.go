package actors

import (
	"fmt"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type ConsoleLogger struct{}

func NewConsoleLogger() actor.Producer {
	return func() actor.Receiver {
		return &ConsoleLogger{}
	}
}

func (cl *ConsoleLogger) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("ConsoleLogger started", "pid", c.PID())

	case DomainChecked:
		cl.handleDomainChecked(msg)

	case Alert:
		cl.handleAlert(msg)

	case actor.Stopped:
		slog.Info("ConsoleLogger stopped", "pid", c.PID())
	}
}

func (cl *ConsoleLogger) handleDomainChecked(msg DomainChecked) {
	status := "✅ VÁLIDO"
	if !msg.Check.IsValid {
		status = "❌ INVÁLIDO"
	}

	fmt.Printf("[%s] Dominio: %s\n", status, msg.Check.Domain)
	fmt.Printf("   IP Esperada: %s\n", msg.Check.ExpectedIP)
	fmt.Printf("   IPs Obtenidas: %v\n", msg.Check.ActualIPs)
	if msg.Check.Error != "" {
		fmt.Printf("   Error: %s\n", msg.Check.Error)
	}
	fmt.Println()
}

func (cl *ConsoleLogger) handleAlert(msg Alert) {
	emoji := "⚠️"
	if msg.Level == "ERROR" {
		emoji = "🚨"
	}

	fmt.Printf("%s %s: %s\n\n", emoji, msg.Level, msg.Message)
}
