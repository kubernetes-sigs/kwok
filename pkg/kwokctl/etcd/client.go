/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Client is an interface that defines the operations that can be performed on an etcd client.
type Client interface {
	// Get is a method that retrieves a key-value pair from the etcd server.
	// It returns the revision of the key-value pair
	Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error)

	// Watch is a method that watches for changes to a key-value pair on the etcd server.
	Watch(ctx context.Context, prefix string, opOpts ...OpOption) error

	// Delete is a method that deletes a key-value pair from the etcd server.
	Delete(ctx context.Context, prefix string, opOpts ...OpOption) error

	// Put is a method that sets a key-value pair on the etcd server.
	Put(ctx context.Context, prefix string, value []byte, opOpts ...OpOption) error
}

// client is the etcd client.
type client struct {
	client *clientv3.Client
}

// ClientConfig is the etcd client config.
type ClientConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string
	// TLS holds the client secure credentials, if any.
	TLS *tls.Config
	// Username is a user name for authentication.
	Username string
	// Password is a password for authentication.
	Password string
}

// NewClient creates a new etcd client.
func NewClient(conf ClientConfig) (Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: conf.Endpoints,
		TLS:       conf.TLS,
		Username:  conf.Username,
		Password:  conf.Password,
	})
	if err != nil {
		return nil, err
	}
	return &client{
		client: cli,
	}, nil
}

func (c *client) getPrefix(prefix string, opt Op) (string, bool, error) {
	var single bool
	var arr [4]string
	s := arr[:0]
	s = append(s, prefix)

	if !opt.gvr.Empty() {
		p, err := PrefixFromGVR(opt.gvr)
		if err != nil {
			return "", false, err
		}
		s = append(s, p)
		if opt.namespace != "" {
			s = append(s, opt.namespace)
		}
		if opt.name != "" {
			s = append(s, opt.name)
			single = true
		}
	}
	return strings.Join(s, "/"), single, nil
}

// Op is the option for the operation.
type Op struct {
	gvr       schema.GroupVersionResource
	name      string
	namespace string
	response  func(kv *KeyValue) error
	pageLimit int64
	keysOnly  bool
	revision  int64
}

// OpOption is the option for the operation.
type OpOption func(*Op)

// WithGVR sets the gvr for the target.
func WithGVR(gvr schema.GroupVersionResource) OpOption {
	return func(o *Op) {
		o.gvr = gvr
	}
}

// WithName sets the name and namespace for the target.
func WithName(name, namespace string) OpOption {
	return func(o *Op) {
		o.name = name
		o.namespace = namespace
	}
}

// WithResponse sets the response callback for the target.
func WithResponse(response func(kv *KeyValue) error) OpOption {
	return func(o *Op) {
		o.response = response
	}
}

// WithPageLimit sets the page limit for the target.
func WithPageLimit(pageLimit int64) OpOption {
	return func(o *Op) {
		o.pageLimit = pageLimit
	}
}

// WithKeysOnly sets the keys only for the target.
func WithKeysOnly() OpOption {
	return func(o *Op) {
		o.keysOnly = true
	}
}

// WithRevision sets the revision for the target.
func WithRevision(revision int64) OpOption {
	return func(o *Op) {
		o.revision = revision
	}
}

func opOption(opts []OpOption) Op {
	var opt Op
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func (c *client) Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error) {
	opt := opOption(opOpts)
	if opt.response == nil {
		return 0, fmt.Errorf("response is required")
	}

	prefix, single, err := c.getPrefix(prefix, opt)
	if err != nil {
		return 0, err
	}

	opts := []clientv3.OpOption{}
	if opt.keysOnly {
		opts = append(opts, clientv3.WithKeysOnly())
	}

	if single || opt.pageLimit == 0 {
		if !single {
			opts = append(opts, clientv3.WithPrefix())
		}
		resp, err := c.client.Get(ctx, prefix, opts...)
		if err != nil {
			return 0, err
		}
		for _, kv := range resp.Kvs {
			r := &KeyValue{
				Key:   kv.Key,
				Value: kv.Value,
			}
			err := opt.response(r)
			if err != nil {
				return 0, err
			}
		}
		return resp.Header.Revision, nil
	}

	respchan := make(chan clientv3.GetResponse, 10)
	errchan := make(chan error, 1)
	var revision int64

	go func() {
		defer close(respchan)
		defer close(errchan)

		var key string

		opts := append(opts, clientv3.WithLimit(opt.pageLimit))
		if opt.revision != 0 {
			revision = opt.revision
			opts = append(opts, clientv3.WithRev(revision))
		}

		if len(prefix) == 0 {
			// If len(s.prefix) == 0, we will sync the entire key-value space.
			// We then range from the smallest key (0x00) to the end.
			opts = append(opts, clientv3.WithFromKey())
			key = "\x00"
		} else {
			// If len(s.prefix) != 0, we will sync key-value space with given prefix.
			// We then range from the prefix to the next prefix if exists. Or we will
			// range from the prefix to the end if the next prefix does not exists.
			opts = append(opts, clientv3.WithRange(clientv3.GetPrefixRangeEnd(prefix)))
			key = prefix
		}

		for {
			resp, err := c.client.Get(ctx, key, opts...)
			if err != nil {
				errchan <- err
				return
			}

			respchan <- *resp

			if revision == 0 {
				revision = resp.Header.Revision
				opts = append(opts, clientv3.WithRev(resp.Header.Revision))
			}

			if !resp.More {
				return
			}

			// move to next key
			key = string(append(resp.Kvs[len(resp.Kvs)-1].Key, 0))
		}
	}()

	for resp := range respchan {
		for _, kv := range resp.Kvs {
			r := &KeyValue{
				Key:   kv.Key,
				Value: kv.Value,
			}
			err := opt.response(r)
			if err != nil {
				return 0, err
			}
		}
	}

	err = <-errchan
	if err != nil {
		return 0, err
	}

	return revision, nil
}

func (c *client) Watch(ctx context.Context, prefix string, opOpts ...OpOption) error {
	opt := opOption(opOpts)
	if opt.response == nil {
		return fmt.Errorf("response is required")
	}

	prefix, single, err := c.getPrefix(prefix, opt)
	if err != nil {
		return err
	}

	opts := []clientv3.OpOption{}
	if opt.keysOnly {
		opts = append(opts, clientv3.WithKeysOnly())
	}

	if !single {
		opts = append(opts, clientv3.WithPrefix())
	}

	if opt.revision != 0 {
		opts = append(opts, clientv3.WithRev(opt.revision))
	}

	opts = append(opts, clientv3.WithPrevKV())

	watchChan := c.client.Watch(ctx, prefix, opts...)
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			r := &KeyValue{
				Key:   event.Kv.Key,
				Value: event.Kv.Value,
			}
			if event.PrevKv != nil {
				r.PrevValue = event.PrevKv.Value
			}
			err := opt.response(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *client) Delete(ctx context.Context, prefix string, opOpts ...OpOption) error {
	opt := opOption(opOpts)
	prefix, _, err := c.getPrefix(prefix, opt)
	if err != nil {
		return err
	}

	opts := []clientv3.OpOption{}

	if opt.name == "" {
		opts = append(opts, clientv3.WithPrefix())
	}

	if opt.response != nil {
		if opt.keysOnly {
			opts = append(opts, clientv3.WithKeysOnly())
		}
		opts = append(opts, clientv3.WithPrevKV())
	}

	resp, err := c.client.Delete(ctx, prefix, opts...)
	if err != nil {
		return err
	}

	if opt.response != nil {
		for _, kv := range resp.PrevKvs {
			r := &KeyValue{
				Key:       kv.Key,
				PrevValue: kv.Value,
			}
			err = opt.response(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *client) Put(ctx context.Context, prefix string, value []byte, opOpts ...OpOption) error {
	opt := opOption(opOpts)
	prefix, single, err := c.getPrefix(prefix, opt)
	if err != nil {
		return err
	}
	if !single {
		return fmt.Errorf("put only support single")
	}

	opts := []clientv3.OpOption{}

	if opt.response != nil {
		if opt.keysOnly {
			opts = append(opts, clientv3.WithKeysOnly())
		}
		opts = append(opts, clientv3.WithPrevKV())
	}

	resp, err := c.client.Put(ctx, prefix, string(value), opts...)
	if err != nil {
		return err
	}

	if opt.response != nil {
		var r *KeyValue
		if resp.PrevKv != nil {
			r = &KeyValue{
				Key:       resp.PrevKv.Key,
				Value:     value,
				PrevValue: resp.PrevKv.Value,
			}
		}
		err = opt.response(r)
		if err != nil {
			return err
		}
	}
	return nil
}

// KeyValue is the key-value pair.
type KeyValue struct {
	Key       []byte
	Value     []byte
	PrevValue []byte
}
