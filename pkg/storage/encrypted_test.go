package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pomerium/pomerium/pkg/cryptutil"
	"github.com/pomerium/pomerium/pkg/grpc/databroker"
	"github.com/pomerium/pomerium/pkg/protoutil"
)

func TestEncryptedBackend(t *testing.T) {
	ctx := context.Background()

	m := map[string]*anypb.Any{}
	backend := &mockBackend{
		put: func(ctx context.Context, records []*databroker.Record) (uint64, error) {
			for _, record := range records {
				record.ModifiedAt = timestamppb.Now()
				record.Version++
				m[record.GetId()] = record.GetData()
			}
			return 0, nil
		},
		get: func(ctx context.Context, recordType, id string) (*databroker.Record, error) {
			data, ok := m[id]
			if !ok {
				return nil, errors.New("not found")
			}
			return &databroker.Record{
				Id:         id,
				Data:       data,
				Version:    1,
				ModifiedAt: timestamppb.Now(),
			}, nil
		},
	}

	e, err := NewEncryptedBackend(cryptutil.NewKey(), backend)
	if !assert.NoError(t, err) {
		return
	}

	t.Run("simple", func(t *testing.T) {
		any := protoutil.NewAny(wrapperspb.String("HELLO WORLD"))
		rec := &databroker.Record{
			Type: "",
			Id:   "TEST-1",
			Data: any,
		}
		_, err = e.Put(ctx, []*databroker.Record{rec})
		if !assert.NoError(t, err) {
			return
		}
		if assert.NotNil(t, m["TEST-1"], "key should be set") {
			assert.NotEqual(t, any.TypeUrl, m["TEST-1"].TypeUrl, "encrypted data should be a bytes type")
			assert.NotEqual(t, any.Value, m["TEST-1"].Value, "value should be encrypted")
			assert.NotNil(t, rec.ModifiedAt)
			assert.NotZero(t, rec.Version)
		}

		record, err := e.Get(ctx, "", "TEST-1")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, any.TypeUrl, record.Data.TypeUrl, "type should be preserved")
		assert.Equal(t, any.Value, record.Data.Value, "value should be preserved")
		assert.NotEqual(t, any.TypeUrl, record.Type, "record type should be preserved")
	})

	t.Run("index", func(t *testing.T) {
		s, err := structpb.NewStruct(map[string]interface{}{
			"$index": map[string]interface{}{
				"cidr": "192.168.0.0/16",
			},
			"example": "value",
		})
		require.NoError(t, err)
		any := protoutil.NewAny(s)
		record := &databroker.Record{
			Id:   "TEST-2",
			Data: any,
		}
		_, err = e.Put(ctx, []*databroker.Record{record})
		require.NoError(t, err)

		if assert.NotNil(t, m["TEST-2"], "key should be set") {
			assert.NotEqual(t, any.Value, m["TEST-2"].Value, "value should be encrypted")
			assert.NotNil(t, record.ModifiedAt)
			assert.NotZero(t, record.Version)
		}

		record, err = e.Get(ctx, "", "TEST-2")
		require.NoError(t, err)
		assert.Equal(t, any.TypeUrl, record.Data.TypeUrl, "type should be preserved")
		assert.Equal(t, any.Value, record.Data.Value, "value should be preserved")
		assert.NotEqual(t, any.TypeUrl, record.Type, "record type should be preserved")
	})
}
