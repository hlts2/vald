package rebalance

import (
	"archive/tar"
	"context"
	"encoding/gob"
	"io"

	"github.com/vdaas/vald/apis/grpc/v1/payload"
	"github.com/vdaas/vald/internal/client/v1/client"
	"github.com/vdaas/vald/internal/errgroup"
	"github.com/vdaas/vald/internal/errors"
	ctxio "github.com/vdaas/vald/internal/io"
	"github.com/vdaas/vald/internal/log"
	"github.com/vdaas/vald/internal/safety"
	"github.com/vdaas/vald/pkg/rebalancer/storage/job/service/storage"
)

const (
	kvsDBName = "ngt-meta.kvsdb"
)

type Rebalance interface {
	Start(ctx context.Context) (chan<- error, error)
}

type rebalance struct {
	storage storage.Storage
	eg      errgroup.Group

	client client.Client
}

func (r *rebalance) Start(ctx context.Context) (chan<- error, error) {
	errCh := make(chan error)

	pr, pw := io.Pipe()
	defer pr.Close()

	// NOTE: think about error handling

	// 1. Start rebalancer process.
	r.eg.Go(func() error {

		r.eg.Go(safety.RecoverFunc(func() (err error) {
			defer pw.Close()
			defer func() {
				if err != nil {
					select {
					case <-ctx.Done():
						errCh <- errors.Wrap(err, ctx.Err().Error())
					case errCh <- err:
					}
				}
			}()

			sr, err := r.storage.Reader(ctx)
			if err != nil {
				return err
			}

			sr, err = ctxio.NewReadCloserWithContext(ctx, sr)
			if err != nil {
				return err
			}
			defer func() {
				e := sr.Close()
				if e != nil {
					log.Errorf("error on closing blob-storage reader: %s", e)
				}
			}()

			_, err = io.Copy(pw, sr)
			if err != nil {
				return err
			}

			return nil
		}))

		idm, err := r.loadKVS(ctx, pr)
		if err != nil {
			select {
			case <-ctx.Done():
				errCh <- errors.Wrap(err, ctx.Err().Error())
			case errCh <- err:
			}
			return err
		}

		// 5. calculate to process data from vector ids.
		// TODO:

		// 6. send request for getting vector.
		// 7. send request for updateing vector.
		for uid := range idm {
			resp, err := r.client.GetObject(ctx, &payload.Object_VectorRequest{
				Id: &payload.Object_ID{
					Id: uid,
				},
			})
			if err != nil {
				log.Error(err)
				continue
			}

			_, err = r.client.Update(ctx, &payload.Update_Request{
				Vector: &payload.Object_Vector{
					Id:     resp.GetId(),
					Vector: resp.GetVector(),
				},
			})
			if err != nil {
				log.Error(err)
			}
		}
		return nil
	})

	return errCh, nil
}

func (r *rebalance) loadKVS(ctx context.Context, reader io.Reader) (idm map[string]uint32, err error) {
	tr := tar.NewReader(reader)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var header *tar.Header
		header, err = tr.Next()
		if err != nil {
			if err == io.EOF {
				return
			}

			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if header.Name != kvsDBName {
				continue
			}

			gob.Register(map[string]uint32{})

			// unpacka blob
			idm := make(map[string]uint32)
			err = gob.NewDecoder(tr).Decode(&idm)
			if err != nil {
				return nil, err
			}

			return idm, nil
		}
	}
}
