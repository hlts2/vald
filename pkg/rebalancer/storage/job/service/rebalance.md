## Rebalance Job

### Code deep dive for Rebalance process

Jobは以下の6つのプロセスを実行することでベクトルデータのリバランスを行う。


本項では各プロセスをコードレベルで説明する。
レビュー済みの詳細設計をどういったコードロジックで実現するかを説明する。
尚`Load configuration`については、以前に紹介したので今回の説明の対象から省く。

1. Load configuration
2. Download tar gz file
3. Unpacka tar file
4. Decode kvsdb file to get vector ids
5. Calculate to process data from the above data
6. Rebalance

↓ Jobコードフレームに肉付けをしながら説明する

Usecase層から実際にStartされるコード。これがJobの全体のフローになる。

```go
pkg/rebalancer/storage/job/service/job.go

const (
	// ベクトルデータのidが保存されているファイル名。
	// Cyberduckなりでtarファイルを解答すると以下のファイルを見つけることができる
	kvsDBName = "ngt-meta.kvsdb"
)

// RebalanceJobのインターフェース。Rebalanceのフローが開始される。
type RebalanceJob interface {
	Start(ctx context.Context) (chan<- error, error)
}

type rebalanceJob struct {
	eg      errgroup.Group
}

// Rebalanceフローを開始するメソッド。Usecaseから呼ばれる
func (rj *rebalanceJob) Start(ctx context.Context) (chan<- error, error) {

	// Rebalance最中に発生したエラーをUsecaseに伝搬するためのチャネル
	errCh := make(chan error)

	// 実際のRebalanceフロー
	rj.eg.Go(func() error {
		
		// Download tar gz file
		
		// Unpacka tar file
		
		// Decode kvsdb file to get vector ids
		
		// Calculate to process data from the above data
		
		// Rebalance

		return nil
	})

	return errCh, nil
}
```

#### Load configuration

Skip 説明は共有済み

#### Download tar gz file

```go

// Rebalanceフローを開始するメソッド。Usecaseから呼ばれる
func (rj *rebalanceJob) Start(ctx context.Context) (chan<- error, error) {
	
	// Rebalance最中に発生したエラーをUsecaseに伝搬するためのチャネル
	errCh := make(chan error)
	
	// io.Pipeは使うかは別途検討
	// なぜ使うのか: パフォーマンス（tarファイルが大きい場合にメモリに乗らないなど？）
	// https://medium.com/eureka-engineering/file-uploads-in-go-with-io-pipe-75519dfa647b
	// pwに書き込まれたデータ（tar）はprで読み出し可能になる。
	pr, pw := io.Pipe()
	defer pr.Close() // ここでいい？ unpack後でもいいかも？
	
	// 実際のRebalanceフロー
	rj.eg.Go(func() error {
		
		// Download tar gz file
		rj.eg.Go(safety.RecoverFunc(func() (err error) {
			defer pw.Close()
			defer func() {
				if err != nil {
					errCh <- err
				}
			}()
			
			// 外部ストレージからtarデータをダウンロードするために、Readerを生成する
			// srのReadを呼べば外部ストレージのデータを読み込むことができる
			sr, err := rj.storage.Reader(ctx)
			if err != nil {
				return err
			}
			
			// contextキャンセルに対応するために、キャンセル可能ReadCloserを生成する
			sr, err = ctxio.NewReadCloserWithContext(ctx, sr)
			if err != nil {
				return err
			}
			defer func() {
				// Readerは開いたあとにCloseする必要があるので、Closeメソッドを呼ぶ
				e := sr.Close()
				if e != nil {
					log.Errorf("error on closing blob-storage reader: %s", e)
				}
			}()
			
			// srから読み込んだtarデータは順次、パイプのpwに送る（pwが書き込まれたらprから読み出すことができる）
			_, err = io.Copy(pw, sr)
			if err != nil {
				return err
			}
			return nil
		}))
		
		// Unpacka tar file
		
		// Decode kvsdb file to get vector ids
		
		// Calculate to process data from the above data
		
		// Rebalance
		
		return nil
	})
	
	return errCh, nil
}
```

#### Unpacka tar file & (Decode kvsdb file to get vector ids)

```go

// Rebalanceフローを開始するメソッド。Usecaseから呼ばれる
func (rj *rebalanceJob) Start(ctx context.Context) (chan<- error, error) {
	errCh := make(chan error)
	
	pr, pw := io.Pipe()
	defer pr.Close() // ここでいい？ unpack後でもいいかも？

	
	// 実際のRebalanceフロー
	rj.eg.Go(func() error {
		
		// Download tar gz file
		rj.eg.Go(safety.RecoverFunc(func() error {
			// 省略
			return nil
		}))
		
		// Unpacka tar file
		// Decode kvsdb file to get vector ids
		idm, err := r.loadKVS(ctx, pr)
		if err != nil {
			errCh <- err
			return nil
		}
		
		// Calculate to process data from the above data
		
		// Rebalance
		
		return nil
	})
	return errCh, nil
}

// readerからtarの読み出しを行う。（パイプのpwに書き込まれたら順次読み込むことができる。
// ファイルに書き出していたら時間がかかるので、ファイル書き込みせずに生のreaderとgobのDecoderを用いてvectorIDデータを取得する
func (r *rebalance) loadKVS(ctx context.Context, reader io.Reader) (idm map[string]uint32, err error) {
	
	tr := tar.NewReader(reader)
	
	for {
		// contextがキャンセルされたらReturnする
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Headerの取得を行う
		// （Header内にファイル名が含まれているの）
		var header *tar.Header
		header, err = tr.Next()
		if err != nil {
			if err == io.EOF {
				// io.EOFが来たら、returnする
				// if err != nilのときにはerrを返しても良さそう？
				// 215行目で説明する内容にはなるが、kvsDBNameが取得できずにio.EOFが来た場合、tar内部にkvsdbファイルがないことなのでエラーでも良さそう？
				return nil, nil
			}
			
			// 取得にエラーがあった場合は早期にReturnする
			return nil, err
		}
		
		// Headerのタイプを確認する
		switch header.Typeflag {
			
		// TypeReg == '0'のときは通常のファイル
		case tar.TypeReg:
		
			// header.Nameにファイル名が含まれているので、対象となるkvsDB（ngt-meta.kvsdb）のファイル名ならば処理を継続、そうでないならContinue
			if header.Name != kvsDBName {
				continue
			}
			
			// gob.Registerで変換する型を登録
			gob.Register(map[string]uint32{})
			
			idm := make(map[string]uint32)
			// tarのreader経由でDecodeを行う。
			err = gob.NewDecoder(tr).Decode(&idm)
			if err != nil {
				// エラーが起これば早期にRetrunする
				return nil, err
			}
			
			// デコードデーtを早期に返す
			return idm, nil
		}
	}
}
```

#### Calculate to process data from the above data

#### Rebalance

