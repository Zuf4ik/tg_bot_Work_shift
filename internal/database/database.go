package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"tg_bot/config"
)

func Connect() (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("ошибка конфигурации PostgreSQL: %w", err)
	}

	port, err := strconv.ParseUint(config.GetDBPort(), 10, 16)
	if err != nil || port == 0 {
		return nil, fmt.Errorf("некорректный порт PostgreSQL")
	}

	poolConfig.ConnConfig.Host = config.GetDBHost()
	poolConfig.ConnConfig.Port = uint16(port)
	poolConfig.ConnConfig.User = config.GetDBUser()
	poolConfig.ConnConfig.Password = config.GetDBPassword()
	poolConfig.ConnConfig.Database = config.GetDBName()

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
