package service

import (
	"context"

	"github.com/vdaas/vald/internal/errgroup"
)

const (
	kvsDBName = "ngt-meta.kvsdb"
)

type RebalanceJob interface {
	Start(ctx context.Context) (chan<- error, error)
}

type rebalanceJob struct {
	storage Storage
	eg      errgroup.Group
}

func (rj *rebalanceJob) Start(ctx context.Context) (chan<- error, error) {
	errCh := make(chan error)

	// Start rebalancer process.
	rj.eg.Go(func() error {

		// 2. download tar gz file

		// 3. unpacka tar file

		// 4. decode vcache filea to get vector ids.

		// 5. calculate to process data from vector ids.

		// 6. send request for getting vector.

		// 7. send request for updateing vector.
		return nil
	})

	return errCh, nil
}
