package regclient

import (
	"context"

	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

type manifestOpt struct {
	d          types.Descriptor
	schemeOpts []scheme.ManifestOpts
}

// ManifestOpts define options for the Manifest* commands
type ManifestOpts func(*manifestOpt)

// WithManifest passes a manifest to ManifestDelete.
func WithManifest(m manifest.Manifest) ManifestOpts {
	return func(opts *manifestOpt) {
		opts.schemeOpts = append(opts.schemeOpts, scheme.WithManifest(m))
	}
}

// WithManifestCheckRefers checks for refers field on ManifestDelete.
func WithManifestCheckRefers() ManifestOpts {
	return func(opts *manifestOpt) {
		opts.schemeOpts = append(opts.schemeOpts, scheme.WithManifestCheckRefers())
	}
}

// WithManifestChild for ManifestPut.
func WithManifestChild() ManifestOpts {
	return func(opts *manifestOpt) {
		opts.schemeOpts = append(opts.schemeOpts, scheme.WithManifestChild())
	}
}

// WithManifestDesc includes the descriptor for ManifestGet.
// This is used to automatically extract a Data field if available.
func WithManifestDesc(d types.Descriptor) ManifestOpts {
	return func(opts *manifestOpt) {
		opts.d = d
	}
}

// ManifestDelete removes a manifest, including all tags pointing to that registry
// The reference must include the digest to delete (see TagDelete for deleting a tag)
// All tags pointing to the manifest will be deleted
func (rc *RegClient) ManifestDelete(ctx context.Context, r ref.Ref, opts ...ManifestOpts) error {
	opt := manifestOpt{schemeOpts: []scheme.ManifestOpts{}}
	for _, fn := range opts {
		fn(&opt)
	}
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return err
	}
	return schemeAPI.ManifestDelete(ctx, r, opt.schemeOpts...)
}

// ManifestGet retrieves a manifest
func (rc *RegClient) ManifestGet(ctx context.Context, r ref.Ref, opts ...ManifestOpts) (manifest.Manifest, error) {
	opt := manifestOpt{schemeOpts: []scheme.ManifestOpts{}}
	for _, fn := range opts {
		fn(&opt)
	}
	if opt.d.Digest != "" {
		r.Digest = opt.d.Digest.String()
		data, err := opt.d.GetData()
		if err == nil {
			return manifest.New(
				manifest.WithDesc(opt.d),
				manifest.WithRaw(data),
				manifest.WithRef(r),
			)
		}
	}
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return nil, err
	}
	return schemeAPI.ManifestGet(ctx, r)
}

// ManifestHead queries for the existence of a manifest and returns metadata (digest, media-type, size)
func (rc *RegClient) ManifestHead(ctx context.Context, r ref.Ref) (manifest.Manifest, error) {
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return nil, err
	}
	return schemeAPI.ManifestHead(ctx, r)
}

// ManifestPut pushes a manifest
// Any descriptors referenced by the manifest typically need to be pushed first
func (rc *RegClient) ManifestPut(ctx context.Context, r ref.Ref, m manifest.Manifest, opts ...ManifestOpts) error {
	opt := manifestOpt{schemeOpts: []scheme.ManifestOpts{}}
	for _, fn := range opts {
		fn(&opt)
	}
	schemeAPI, err := rc.schemeGet(r.Scheme)
	if err != nil {
		return err
	}
	return schemeAPI.ManifestPut(ctx, r, m, opt.schemeOpts...)
}
