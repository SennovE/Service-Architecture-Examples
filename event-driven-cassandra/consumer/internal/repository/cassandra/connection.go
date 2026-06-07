package cassandra

import (
	"consumer/internal/models"
	"fmt"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type Store struct {
	session *gocql.Session
}

func New(contactPoints []string, keyspace, localDC string) (*Store, error) {
	if len(contactPoints) == 0 {
		return nil, fmt.Errorf("cassandra contact points are empty")
	}
	if keyspace == "" {
		return nil, fmt.Errorf("cassandra keyspace is empty")
	}
	if localDC == "" {
		return nil, fmt.Errorf("cassandra local dc is empty")
	}

	cluster := gocql.NewCluster(contactPoints...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 20 * time.Second
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(
		gocql.DCAwareRoundRobinPolicy(localDC),
	)

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("connect cassandra: %w", err))
	}

	return &Store{session: session}, nil
}

func (s *Store) Close() {
	if s != nil && s.session != nil {
		s.session.Close()
	}
}