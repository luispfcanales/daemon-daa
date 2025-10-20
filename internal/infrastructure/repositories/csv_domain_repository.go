package repositories

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
	"github.com/luispfcanales/daemon-daa/internal/core/ports"
)

type CSVDomainRepository struct {
	configsFile string
	checksFile  string
	mutex       sync.RWMutex
}

func NewCSVDomainRepository(configsFile, checksFile string) ports.DomainRepository {
	repo := &CSVDomainRepository{
		configsFile: configsFile,
		checksFile:  checksFile,
	}

	// Inicializar archivos si no existen
	repo.initializeFiles()

	return repo
}

func (r *CSVDomainRepository) initializeFiles() {
	// Inicializar archivo de configuraciones si no existe
	if _, err := os.Stat(r.configsFile); os.IsNotExist(err) {
		file, err := os.Create(r.configsFile)
		if err != nil {
			panic(fmt.Sprintf("No se pudo crear archivo de configuraciones: %v", err))
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Escribir encabezados
		writer.Write([]string{"domain", "expected_ip", "status"})

		// Datos iniciales
		initialConfigs := [][]string{
			{"intranet.unamad.edu.pe", "110.238.69.0", "false"},
			{"aulavirtual.unamad.edu.pe", "110.238.69.0", "false"},
			{"matricula.unamad.edu.pe", "110.238.69.0", "false"},
		}

		for _, config := range initialConfigs {
			writer.Write(config)
		}
	}

	// Inicializar archivo de checks si no existe
	if _, err := os.Stat(r.checksFile); os.IsNotExist(err) {
		file, err := os.Create(r.checksFile)
		if err != nil {
			panic(fmt.Sprintf("No se pudo crear archivo de checks: %v", err))
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Escribir encabezados
		writer.Write([]string{"domain", "expected_ip", "actual_ips", "is_valid", "error", "timestamp", "duration"})
	}
}

// getDomainConfigsUnsafe lee las configuraciones SIN lock
func (r *CSVDomainRepository) getDomainConfigsUnsafe() ([]domain.DomainConfig, error) {
	file, err := os.Open(r.configsFile)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de configuraciones: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo CSV: %v", err)
	}

	var configs []domain.DomainConfig

	// Saltar encabezado
	for i, record := range records {
		if i == 0 {
			continue // Saltar encabezado
		}

		if len(record) >= 3 {
			status := record[2] == "true"
			config := domain.DomainConfig{
				Domain:     record[0],
				ExpectedIP: record[1],
				Status:     status,
			}
			configs = append(configs, config)
		}
	}

	return configs, nil
}

// writeConfigsUnsafe escribe las configuraciones al archivo SIN lock
func (r *CSVDomainRepository) writeConfigsUnsafe(configs []domain.DomainConfig) error {
	file, err := os.Create(r.configsFile)
	if err != nil {
		return fmt.Errorf("error creando archivo de configuraciones: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Escribir encabezado
	if err := writer.Write([]string{"domain", "expected_ip", "status"}); err != nil {
		return fmt.Errorf("error escribiendo encabezado: %v", err)
	}

	// Escribir configuraciones
	for _, config := range configs {
		statusStr := "false"
		if config.Status {
			statusStr = "true"
		}

		record := []string{
			config.Domain,
			config.ExpectedIP,
			statusStr,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error escribiendo registro: %v", err)
		}
	}

	return nil
}

// getChecksUnsafe lee los checks SIN lock
func (r *CSVDomainRepository) getChecksUnsafe() ([]domain.DomainCheck, error) {
	file, err := os.Open(r.checksFile)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de checks: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo CSV: %v", err)
	}

	var checks []domain.DomainCheck

	// Saltar encabezado
	for i, record := range records {
		if i == 0 {
			continue // Saltar encabezado
		}

		if len(record) >= 7 {
			// Parsear IPs desde JSON
			var actualIPs []string
			if err := json.Unmarshal([]byte(record[2]), &actualIPs); err != nil {
				// Fallback: intentar parsear como string simple
				if record[2] != "" && record[2] != "[]" {
					actualIPs = []string{record[2]}
				} else {
					actualIPs = []string{}
				}
			}

			// Parsear boolean
			isValid, err := strconv.ParseBool(record[3])
			if err != nil {
				isValid = false
			}

			// Parsear timestamp
			timestamp, err := time.Parse(time.RFC3339, record[5])
			if err != nil {
				timestamp = time.Now()
			}

			// Parsear DurationMs de string a float64
			var durationMs float64
			if len(record) > 6 && record[6] != "" {
				durationMs, err = strconv.ParseFloat(record[6], 64)
				if err != nil {
					durationMs = 0.0
				}
			}

			check := domain.DomainCheck{
				Domain:     record[0],
				ExpectedIP: record[1],
				ActualIPs:  actualIPs,
				IsValid:    isValid,
				Error:      record[4],
				Timestamp:  timestamp,
				DurationMs: durationMs,
			}
			checks = append(checks, check)
		}
	}

	return checks, nil
}

// GetDomainConfigs obtiene todas las configuraciones de dominios
func (r *CSVDomainRepository) GetDomainConfigs() ([]domain.DomainConfig, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.getDomainConfigsUnsafe()
}

// AddDomainConfig agrega una nueva configuración de dominio
func (r *CSVDomainRepository) AddDomainConfig(config domain.DomainConfig) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Leer configuraciones existentes
	configs, err := r.getDomainConfigsUnsafe()
	if err != nil {
		return err
	}

	// Verificar si el dominio ya existe
	for _, existingConfig := range configs {
		if existingConfig.Domain == config.Domain {
			return fmt.Errorf("el dominio '%s' ya existe", config.Domain)
		}
	}

	// Agregar nueva configuración
	configs = append(configs, config)

	// Escribir todas las configuraciones
	return r.writeConfigsUnsafe(configs)
}

// RemoveDomainConfig elimina una configuración de dominio
func (r *CSVDomainRepository) RemoveDomainConfig(domainName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	configs, err := r.getDomainConfigsUnsafe()
	if err != nil {
		return err
	}

	// Filtrar el dominio a eliminar y verificar que existe
	found := false
	var newConfigs []domain.DomainConfig

	for _, config := range configs {
		if config.Domain != domainName {
			newConfigs = append(newConfigs, config)
		} else {
			found = true
		}
	}

	// Si no se encontró el dominio, retornar error
	if !found {
		return fmt.Errorf("dominio '%s' no encontrado", domainName)
	}

	// Escribir configuraciones actualizadas
	return r.writeConfigsUnsafe(newConfigs)
}

// UpdateDomainConfig actualiza una configuración existente
func (r *CSVDomainRepository) UpdateDomainConfig(domainName string, newConfig domain.DomainConfig) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	configs, err := r.getDomainConfigsUnsafe()
	if err != nil {
		return err
	}

	found := false
	for i, config := range configs {
		if config.Domain == domainName {
			configs[i] = newConfig
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("dominio '%s' no encontrado", domainName)
	}

	return r.writeConfigsUnsafe(configs)
}

// UpdateDomainStatus actualiza solo el estado de un dominio
func (r *CSVDomainRepository) UpdateDomainStatus(domainName string, status bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	configs, err := r.getDomainConfigsUnsafe()
	if err != nil {
		return err
	}

	found := false
	for i, config := range configs {
		if config.Domain == domainName {
			configs[i].Status = status
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("dominio '%s' no encontrado", domainName)
	}

	return r.writeConfigsUnsafe(configs)
}

// SaveDomainCheck guarda un check de dominio
func (r *CSVDomainRepository) SaveDomainCheck(check domain.DomainCheck) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	file, err := os.OpenFile(r.checksFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo archivo de checks: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Convertir slice de IPs a JSON string para almacenar en CSV
	actualIPsJSON, err := json.Marshal(check.ActualIPs)
	if err != nil {
		return fmt.Errorf("error serializando IPs: %v", err)
	}

	// Convertir DomainCheck a fila CSV
	record := []string{
		check.Domain,
		check.ExpectedIP,
		string(actualIPsJSON),
		strconv.FormatBool(check.IsValid),
		check.Error,
		check.Timestamp.Format(time.RFC3339),
		strconv.FormatFloat(check.DurationMs, 'f', 3, 64), // Guardar con 3 decimales
	}

	return writer.Write(record)
}

// GetChecks obtiene todos los checks
func (r *CSVDomainRepository) GetChecks() ([]domain.DomainCheck, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.getChecksUnsafe()
}

// GetChecksByDomain obtiene los checks filtrados por dominio
func (r *CSVDomainRepository) GetChecksByDomain(domainName string) ([]domain.DomainCheck, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	allChecks, err := r.getChecksUnsafe()
	if err != nil {
		return nil, err
	}

	var filteredChecks []domain.DomainCheck
	for _, check := range allChecks {
		if check.Domain == domainName {
			filteredChecks = append(filteredChecks, check)
		}
	}

	return filteredChecks, nil
}

// GetRecentChecks obtiene los últimos N checks
func (r *CSVDomainRepository) GetRecentChecks(limit int) ([]domain.DomainCheck, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	allChecks, err := r.getChecksUnsafe()
	if err != nil {
		return nil, err
	}

	// Ordenar por timestamp descendente (más reciente primero)
	for i, j := 0, len(allChecks)-1; i < j; i, j = i+1, j-1 {
		allChecks[i], allChecks[j] = allChecks[j], allChecks[i]
	}

	if limit > len(allChecks) {
		limit = len(allChecks)
	}

	return allChecks[:limit], nil
}

// GetChecksByTimeRange obtiene checks dentro de un rango de tiempo
func (r *CSVDomainRepository) GetChecksByTimeRange(start, end time.Time) ([]domain.DomainCheck, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	allChecks, err := r.getChecksUnsafe()
	if err != nil {
		return nil, err
	}

	var filteredChecks []domain.DomainCheck
	for _, check := range allChecks {
		if (check.Timestamp.Equal(start) || check.Timestamp.After(start)) &&
			(check.Timestamp.Equal(end) || check.Timestamp.Before(end)) {
			filteredChecks = append(filteredChecks, check)
		}
	}

	return filteredChecks, nil
}

// GetDomainStats obtiene estadísticas de un dominio específico
func (r *CSVDomainRepository) GetDomainStats(domainName string) (domain.StatsDomain, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Usar método unsafe ya que tenemos el lock
	allChecks, err := r.getChecksUnsafe()
	if err != nil {
		return domain.StatsDomain{}, err
	}

	// Filtrar por dominio
	var checks []domain.DomainCheck
	for _, check := range allChecks {
		if check.Domain == domainName {
			checks = append(checks, check)
		}
	}

	if len(checks) == 0 {
		return domain.StatsDomain{}, nil
	}

	successCount := 0
	var successResponseTimes []float64
	var totalResponseTime float64
	minResponseTime := math.MaxFloat64
	maxResponseTime := 0.0
	lastCheck := checks[len(checks)-1].Timestamp

	for _, check := range checks {
		if check.IsValid && check.Error == "" {
			successCount++

			// Considerar para métricas de tiempo si DurationMs > 0
			if check.DurationMs > 0 {
				successResponseTimes = append(successResponseTimes, check.DurationMs)
				totalResponseTime += check.DurationMs

				if check.DurationMs < minResponseTime {
					minResponseTime = check.DurationMs
				}
				if check.DurationMs > maxResponseTime {
					maxResponseTime = check.DurationMs
				}
			}
		}
	}

	successRate := float64(successCount) / float64(len(checks)) * 100

	// Calcular métricas de response time
	avgResponseTime := 0.0
	p95ResponseTime := 0.0

	if len(successResponseTimes) > 0 {
		avgResponseTime = totalResponseTime / float64(len(successResponseTimes))
		p95ResponseTime = calculatePercentile(successResponseTimes, 95)
	}

	// Si no se encontraron tiempos válidos, resetear min
	if minResponseTime == math.MaxFloat64 {
		minResponseTime = 0.0
	}

	return domain.StatsDomain{
		TotalChecks:      len(checks),
		SuccessCount:     successCount,
		FailureCount:     len(checks) - successCount,
		SuccessRate:      math.Round(successRate*100) / 100,
		AverageUptime:    math.Round(successRate*100) / 100,
		LastCheck:        lastCheck,
		AvgResponseTime:  math.Round(avgResponseTime*100) / 100,
		MinResponseTime:  math.Round(minResponseTime*100) / 100,
		MaxResponseTime:  math.Round(maxResponseTime*100) / 100,
		P95ResponseTime:  math.Round(p95ResponseTime*100) / 100,
		ChecksWithTiming: len(successResponseTimes),
	}, nil
}

// Función para calcular percentiles
func calculatePercentile(times []float64, percentile int) float64 {
	if len(times) == 0 {
		return 0.0
	}

	sorted := make([]float64, len(times))
	copy(sorted, times)
	slices.Sort(sorted)

	index := float64(percentile) / 100.0 * float64(len(sorted)-1)

	if index == float64(int64(index)) {
		return sorted[int(index)]
	}

	lower := sorted[int(math.Floor(index))]
	upper := sorted[int(math.Ceil(index))]
	weight := index - math.Floor(index)

	return lower + (upper-lower)*weight
}
