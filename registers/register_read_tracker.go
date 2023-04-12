package registers

import (
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
)

type RegisterReadEntry struct {
	key  RegisterKey
	read int
}

func (e RegisterReadEntry) String() string {
	return fmt.Sprintf("%v: %v bytes", e.key, e.read)
}

type RemoteRegisterReadTracker struct {
	registerRead []RegisterReadEntry
	filename     string

	log zerolog.Logger
}

var _ RegisterGetWrapper = &RemoteRegisterReadTracker{}

func NewRemoteRegisterReadTracker(result []RegisterReadEntry, log zerolog.Logger) *RemoteRegisterReadTracker {
	return &RemoteRegisterReadTracker{
		registerRead: result,
		log:          log,
	}
}

func (r *RemoteRegisterReadTracker) Wrap(inner RegisterGetRegisterFunc) RegisterGetRegisterFunc {
	return func(owner string, key string) (flow.RegisterValue, error) {
		val, err := inner(owner, key)
		k := RegisterKey{owner, key}.ToReadable()

		if err != nil {
			return nil, err
		}

		r.registerRead = append(r.registerRead, RegisterReadEntry{
			key:  k,
			read: len(val),
		})

		return val, nil
	}
}

func (r *RemoteRegisterReadTracker) Close() error {
	/*
		err := os.MkdirAll(filepath.Dir(r.filename), os.ModePerm)
		if err != nil {
			return err
		}

		csvFile, err := os.Create(r.filename)
		if err != nil {
			return err
		}
		defer func() {
			err := csvFile.Close()
			if err != nil {
				r.log.Error().Err(err).Msg("error closing csv file")
			}
		}()

		csvwriter := csv.NewWriter(csvFile)
		defer csvwriter.Flush()
		err = csvwriter.Write([]string{"# Sequence", "Owner", "Key", "bytes"})
		if err != nil {
			return err
		}
		for n, read := range r.registerRead {

			err := csvwriter.Write([]string{strconv.Itoa(n + 1), read.key.Owner, read.key.Key, strconv.Itoa(read.read)})
			if err != nil {
				return err
			}
		}
	*/
	return nil
}
