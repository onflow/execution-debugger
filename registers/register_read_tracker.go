package registers

import (
	"encoding/csv"
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strconv"
)

type RegisterReadEntry struct {
	key  RegisterKey
	read int
}

func (e RegisterReadEntry) String() string {
	return fmt.Sprintf("%v: %v bytes", e.key, e.read)
}

type RegisterReadTracker struct {
	RegisterReads []RegisterReadEntry
	Log           zerolog.Logger
}

func NewRemoteRegisterReadTracker(log zerolog.Logger) *RegisterReadTracker {
	return &RegisterReadTracker{
		Log: log,
	}
}

var _ RegisterGetWrapper = &RegisterReadTracker{}

func (r *RegisterReadTracker) Wrap(inner RegisterGetRegisterFunc) RegisterGetRegisterFunc {
	return func(owner string, key string) (flow.RegisterValue, error) {
		val, err := inner(owner, key)
		k := RegisterKey{owner, key}.ToReadable()

		if err != nil {
			return nil, err
		}

		r.RegisterReads = append(r.RegisterReads, RegisterReadEntry{
			key:  k,
			read: len(val),
		})

		return val, nil
	}
}

func (r *RegisterReadTracker) Save(directory string) error {
	filename := directory + "/registers"
	err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return err
	}

	csvFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err := csvFile.Close()
		if err != nil {
			r.Log.Error().Err(err).Msg("error closing csv file")
		}
	}()

	csvwriter := csv.NewWriter(csvFile)
	defer csvwriter.Flush()
	err = csvwriter.Write([]string{"# Sequence", "Owner", "Key", "bytes"})
	if err != nil {
		return err
	}
	for n, read := range r.RegisterReads {

		err := csvwriter.Write([]string{strconv.Itoa(n + 1), read.key.Owner, read.key.Key, strconv.Itoa(read.read)})
		if err != nil {
			return err
		}
	}

	return nil
}
