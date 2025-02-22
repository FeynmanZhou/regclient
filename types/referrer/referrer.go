// Package referrer is used for responses to the referrers to a manifest
package referrer

import (
	"bytes"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/ref"
)

// ReferrerList contains the response to a request for referrers
type ReferrerList struct {
	Ref         ref.Ref            `json:"ref"`                   // reference queried
	Descriptors []types.Descriptor `json:"descriptors"`           // descriptors found in Index
	Annotations map[string]string  `json:"annotations,omitempty"` // annotations extracted from Index
	Manifest    manifest.Manifest  `json:"-"`                     // returned OCI Index
	Tags        []string           `json:"-"`                     // tags matched when fetching referrers
}

// Add appends an entry to rl.Manifest, used to modify the client managed Index
func (rl *ReferrerList) Add(m manifest.Manifest) error {
	rlM, ok := rl.Manifest.GetOrig().(v1.Index)
	if !ok {
		return fmt.Errorf("referrer list manifest is not an OCI index for %s", rl.Ref.CommonName())
	}
	// if entry already exists, return
	mDesc := m.GetDescriptor()
	for _, d := range rlM.Manifests {
		if d.Digest == mDesc.Digest {
			return nil
		}
	}
	// update descriptor, pulling up artifact type and annotations
	switch mOrig := m.GetOrig().(type) {
	case v1.ArtifactManifest:
		mDesc.Annotations = mOrig.Annotations
		mDesc.ArtifactType = mOrig.ArtifactType
	case v1.Manifest:
		mDesc.Annotations = mOrig.Annotations
		mDesc.ArtifactType = mOrig.Config.MediaType
	default:
		// other types are not supported
		return fmt.Errorf("invalid manifest for referrer \"%t\": %w", m.GetOrig(), types.ErrUnsupportedMediaType)
	}
	// append descriptor to index
	rlM.Manifests = append(rlM.Manifests, mDesc)
	err := rl.Manifest.SetOrig(rlM)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes an entry from rl.Manifest, used to modify the client managed Index
func (rl *ReferrerList) Delete(m manifest.Manifest) error {
	rlM, ok := rl.Manifest.GetOrig().(v1.Index)
	if !ok {
		return fmt.Errorf("referrer list manifest is not an OCI index for %s", rl.Ref.CommonName())
	}
	// delete matching entries from the list
	mDesc := m.GetDescriptor()
	found := false
	for i := len(rlM.Manifests) - 1; i >= 0; i-- {
		if rlM.Manifests[i].Digest == mDesc.Digest {
			if i < len(rlM.Manifests)-1 {
				rlM.Manifests = append(rlM.Manifests[:i], rlM.Manifests[i+1:]...)
			} else {
				rlM.Manifests = rlM.Manifests[:i]
			}
			found = true
		}
	}
	if !found {
		return fmt.Errorf("refers not found in referrer list%.0w", types.ErrNotFound)
	}
	err := rl.Manifest.SetOrig(rlM)
	if err != nil {
		return err
	}
	return nil
}

// IsEmpty reports if the returned Index contains no manifests
func (rl ReferrerList) IsEmpty() bool {
	rlM, ok := rl.Manifest.GetOrig().(v1.Index)
	if !ok || len(rlM.Manifests) == 0 {
		return true
	}
	return false
}

// MarshalPretty is used for printPretty template formatting
func (rl ReferrerList) MarshalPretty() ([]byte, error) {
	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	if rl.Ref.Reference != "" {
		fmt.Fprintf(tw, "Refers:\t%s\n", rl.Ref.Reference)
	}
	rRef := rl.Ref
	rRef.Tag = ""
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Referrers:\t\n")
	for _, d := range rl.Descriptors {
		fmt.Fprintf(tw, "\t\n")
		if rRef.Reference != "" {
			rRef.Digest = d.Digest.String()
			fmt.Fprintf(tw, "  Name:\t%s\n", rRef.CommonName())
		}
		err := d.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	if rl.Annotations != nil && len(rl.Annotations) > 0 {
		fmt.Fprintf(tw, "Annotations:\t\n")
		keys := make([]string, 0, len(rl.Annotations))
		for k := range rl.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			val := rl.Annotations[name]
			fmt.Fprintf(tw, "  %s:\t%s\n", name, val)
		}
	}
	tw.Flush()
	return buf.Bytes(), nil
}

// FallbackTag returns the ref that should be used when the registry does not support the referrers API
func FallbackTag(r ref.Ref) (ref.Ref, error) {
	rr := r
	dig, err := digest.Parse(r.Digest)
	if err != nil {
		return rr, fmt.Errorf("failed to parse digest for referrers: %w", err)
	}
	rr.Digest = ""
	rr.Tag = fmt.Sprintf("%s-%s", dig.Algorithm(), stringMax(dig.Hex(), 64))
	return rr, nil
}
func stringMax(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
