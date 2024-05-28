package config

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

type Config struct {
    ComputingPower       int
    TimeAdditionMs       int
    TimeSubtractionMs    int
    TimeMultiplicationsMs int
    TimeDivisionsMs      int
}

func LoadConfig(filepath string) (Config, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return Config{}, err
    }
    defer file.Close()

    config := Config{}
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if strings.HasPrefix(line, "#") || line == "" {
            continue
        }

        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            return Config{}, fmt.Errorf("invalid config line: %s", line)
        }

        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        switch key {
        case "COMPUTING_POWER":
            config.ComputingPower, err = strconv.Atoi(value)
            if err != nil {
                return Config{}, fmt.Errorf("invalid value for COMPUTING_POWER: %s", value)
            }
        case "TIME_ADDITION_MS":
            config.TimeAdditionMs, err = strconv.Atoi(value)
            if err != nil {
                return Config{}, fmt.Errorf("invalid value for TIME_ADDITION_MS: %s", value)
            }
        case "TIME_SUBTRACTION_MS":
            config.TimeSubtractionMs, err = strconv.Atoi(value)
            if err != nil {
                return Config{}, fmt.Errorf("invalid value for TIME_SUBTRACTION_MS: %s", value)
            }
        case "TIME_MULTIPLICATIONS_MS":
            config.TimeMultiplicationsMs, err = strconv.Atoi(value)
            if err != nil {
                return Config{}, fmt.Errorf("invalid value for TIME_MULTIPLICATIONS_MS: %s", value)
            }
        case "TIME_DIVISIONS_MS":
            config.TimeDivisionsMs, err = strconv.Atoi(value)
            if err != nil {
                return Config{}, fmt.Errorf("invalid value for TIME_DIVISIONS_MS: %s", value)
            }
        default:
            return Config{}, fmt.Errorf("unknown key: %s", key)
        }
    }

    if err := scanner.Err(); err != nil {
        return Config{}, err
    }

    return config, nil
}
