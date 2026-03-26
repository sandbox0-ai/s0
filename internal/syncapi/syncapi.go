package syncapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	syncsdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"

	"github.com/sandbox0-ai/s0/internal/syncstate"
)

const defaultChangeLimit int32 = 256

// Client wraps the low-level sync endpoints needed by the local worker.
type Client struct {
	sdk *syncsdk.Client
}

// ReseedRequiredError reports that the local replica must bootstrap again.
type ReseedRequiredError struct {
	RetainedAfterSeq int64
	HeadSeq          int64
	Message          string
}

func (e *ReseedRequiredError) Error() string {
	if e == nil {
		return "reseed required"
	}
	if e.Message != "" {
		return e.Message
	}
	return "reseed required"
}

// BootstrapCompatibilityError reports namespace incompatibilities during bootstrap.
type BootstrapCompatibilityError struct {
	Message string
	Details *apispec.VolumeSyncBootstrapCompatibilityConflictDetails
}

func (e *BootstrapCompatibilityError) Error() string {
	if e == nil {
		return "bootstrap compatibility conflict"
	}
	if e.Message != "" {
		return e.Message
	}
	return "bootstrap compatibility conflict"
}

// New creates a sync API wrapper from the shared SDK client.
func New(client *syncsdk.Client) *Client {
	return &Client{sdk: client}
}

// UpsertReplica registers or refreshes the local replica record.
func (c *Client) UpsertReplica(ctx context.Context, attachment *syncstate.Attachment) (*apispec.VolumeSyncReplicaEnvelope, error) {
	req := &apispec.UpsertSyncReplicaRequest{
		DisplayName:   apispec.NewOptString(displayNameFor(attachment)),
		Platform:      apispec.NewOptString(attachment.Platform),
		RootPath:      apispec.NewOptString(attachment.WorkspaceRoot),
		CaseSensitive: apispec.NewOptBool(attachment.Capabilities.CaseSensitive),
		Capabilities:  apispec.NewOptVolumeSyncFilesystemCapabilities(toSDKCapabilities(attachment.Capabilities)),
	}
	resp, err := c.sdk.API().APIV1SandboxvolumesIDSyncReplicasReplicaIDPut(ctx, req, apispec.APIV1SandboxvolumesIDSyncReplicasReplicaIDPutParams{
		ID:        attachment.VolumeID,
		ReplicaID: attachment.ReplicaID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, errors.New("sync replica response missing data")
	}
	return &data, nil
}

// Bootstrap creates a snapshot seed plus replay anchor.
func (c *Client) Bootstrap(ctx context.Context, attachment *syncstate.Attachment) (*apispec.VolumeSyncBootstrap, error) {
	req := apispec.NewOptCreateVolumeSyncBootstrapRequest(apispec.CreateVolumeSyncBootstrapRequest{
		Capabilities: apispec.NewOptVolumeSyncFilesystemCapabilities(toSDKCapabilities(attachment.Capabilities)),
	})
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncBootstrapPost(ctx, req, apispec.APIV1SandboxvolumesIDSyncBootstrapPostParams{
		ID: attachment.VolumeID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	success, ok := res.(*apispec.SuccessVolumeSyncBootstrapResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected bootstrap response type %T", res)
	}
	data, ok := success.Data.Get()
	if !ok {
		return nil, errors.New("sync bootstrap response missing data")
	}
	return &data, nil
}

// DownloadBootstrapArchive downloads the tar.gz archive bytes for one bootstrap snapshot.
func (c *Client) DownloadBootstrapArchive(ctx context.Context, volumeID, snapshotID string) ([]byte, error) {
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncBootstrapArchiveGet(ctx, apispec.APIV1SandboxvolumesIDSyncBootstrapArchiveGetParams{
		ID:         volumeID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	okRes, ok := res.(*apispec.APIV1SandboxvolumesIDSyncBootstrapArchiveGetOK)
	if !ok {
		return nil, fmt.Errorf("unexpected bootstrap archive response type %T", res)
	}
	return ioReadAll(okRes.Data)
}

// ListChanges lists journal entries after one known sequence.
func (c *Client) ListChanges(ctx context.Context, volumeID string, after int64, limit int32) (*apispec.ListVolumeSyncChangesResponse, error) {
	params := apispec.APIV1SandboxvolumesIDSyncChangesGetParams{
		ID:    volumeID,
		After: apispec.NewOptInt64(after),
		Limit: apispec.NewOptInt32(limitOrDefault(limit)),
	}
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncChangesGet(ctx, params)
	if err != nil {
		return nil, classifySyncError(err)
	}
	success, ok := res.(*apispec.SuccessVolumeSyncChangeListResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected change list response type %T", res)
	}
	data, ok := success.Data.Get()
	if !ok {
		return nil, errors.New("sync change list response missing data")
	}
	return &data, nil
}

// AppendChanges pushes one replica mutation batch.
func (c *Client) AppendChanges(ctx context.Context, attachment *syncstate.Attachment, baseSeq int64, requestID string, changes []apispec.ChangeRequest) (*apispec.AppendReplicaChangesResponse, error) {
	req := &apispec.AppendReplicaChangesRequest{
		RequestID: requestID,
		BaseSeq:   baseSeq,
		Changes:   changes,
	}
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncReplicasReplicaIDChangesPost(ctx, req, apispec.APIV1SandboxvolumesIDSyncReplicasReplicaIDChangesPostParams{
		ID:        attachment.VolumeID,
		ReplicaID: attachment.ReplicaID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	success, ok := res.(*apispec.SuccessVolumeSyncAppendResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected append response type %T", res)
	}
	data, ok := success.Data.Get()
	if !ok {
		return nil, errors.New("sync append response missing data")
	}
	return &data, nil
}

// UpdateCursor persists the highest fully applied sequence for this replica.
func (c *Client) UpdateCursor(ctx context.Context, attachment *syncstate.Attachment, lastAppliedSeq int64) (*apispec.VolumeSyncReplicaEnvelope, error) {
	req := &apispec.UpdateSyncReplicaCursorRequest{LastAppliedSeq: lastAppliedSeq}
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncReplicasReplicaIDCursorPut(ctx, req, apispec.APIV1SandboxvolumesIDSyncReplicasReplicaIDCursorPutParams{
		ID:        attachment.VolumeID,
		ReplicaID: attachment.ReplicaID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	success, ok := res.(*apispec.SuccessVolumeSyncReplicaResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected cursor response type %T", res)
	}
	data, ok := success.Data.Get()
	if !ok {
		return nil, errors.New("sync cursor response missing data")
	}
	return &data, nil
}

// ListConflicts lists conflicts for one volume.
func (c *Client) ListConflicts(ctx context.Context, volumeID, status string, limit int32) ([]apispec.SyncConflict, error) {
	params := apispec.APIV1SandboxvolumesIDSyncConflictsGetParams{
		ID:    volumeID,
		Limit: apispec.NewOptInt32(limitOrDefault(limit)),
	}
	if status != "" {
		params.Status = apispec.NewOptString(status)
	}
	resp, err := c.sdk.API().APIV1SandboxvolumesIDSyncConflictsGet(ctx, params)
	if err != nil {
		return nil, classifySyncError(err)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, nil
	}
	return data.Conflicts, nil
}

// ResolveConflict marks one conflict as resolved or ignored.
func (c *Client) ResolveConflict(ctx context.Context, volumeID, conflictID string, ignored bool) (*apispec.SyncConflict, error) {
	status := apispec.ResolveVolumeSyncConflictRequestStatusResolved
	if ignored {
		status = apispec.ResolveVolumeSyncConflictRequestStatusIgnored
	}
	req := &apispec.ResolveVolumeSyncConflictRequest{Status: status}
	res, err := c.sdk.API().APIV1SandboxvolumesIDSyncConflictsConflictIDPut(ctx, req, apispec.APIV1SandboxvolumesIDSyncConflictsConflictIDPutParams{
		ID:         volumeID,
		ConflictID: conflictID,
	})
	if err != nil {
		return nil, classifySyncError(err)
	}
	success, ok := res.(*apispec.SuccessVolumeSyncConflictResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected conflict resolution response type %T", res)
	}
	data, ok := success.Data.Get()
	if !ok {
		return nil, errors.New("sync conflict response missing data")
	}
	return &data, nil
}

func classifySyncError(err error) error {
	var apiErr *syncsdk.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	if reseed := parseReseedRequired(apiErr); reseed != nil {
		return reseed
	}
	if bootstrap := parseBootstrapConflict(apiErr); bootstrap != nil {
		return bootstrap
	}
	return err
}

func parseReseedRequired(apiErr *syncsdk.APIError) *ReseedRequiredError {
	if apiErr == nil {
		return nil
	}
	if apiErr.Code != "conflict" && apiErr.Code != "invalid_request" {
		// storage-proxy currently uses the generic conflict envelope here.
	}
	details := &apispec.VolumeSyncReseedRequiredDetails{}
	if !decodeAPIErrorDetails(apiErr, details) || details.Reason != apispec.VolumeSyncReseedRequiredDetailsReasonReseedRequired {
		return nil
	}
	return &ReseedRequiredError{
		RetainedAfterSeq: details.RetainedAfterSeq,
		HeadSeq:          details.HeadSeq,
		Message:          apiErr.Message,
	}
}

func parseBootstrapConflict(apiErr *syncsdk.APIError) *BootstrapCompatibilityError {
	if apiErr == nil {
		return nil
	}
	details := &apispec.VolumeSyncBootstrapCompatibilityConflictDetails{}
	if !decodeAPIErrorDetails(apiErr, details) || details.Reason != apispec.VolumeSyncBootstrapCompatibilityConflictDetailsReasonNamespaceIncompatible {
		return nil
	}
	return &BootstrapCompatibilityError{
		Message: apiErr.Message,
		Details: details,
	}
}

func decodeAPIErrorDetails(apiErr *syncsdk.APIError, target any) bool {
	if apiErr == nil || target == nil {
		return false
	}
	if apiErr.Details != nil {
		if raw, err := json.Marshal(apiErr.Details); err == nil && len(raw) > 0 && json.Unmarshal(raw, target) == nil {
			return true
		}
	}
	if len(apiErr.Body) == 0 {
		return false
	}
	var envelope struct {
		Error struct {
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(apiErr.Body, &envelope); err != nil || len(envelope.Error.Details) == 0 {
		return false
	}
	return json.Unmarshal(envelope.Error.Details, target) == nil
}

func displayNameFor(attachment *syncstate.Attachment) string {
	if attachment == nil {
		return ""
	}
	if attachment.DisplayName != "" {
		return attachment.DisplayName
	}
	return attachment.ReplicaID
}

func toSDKCapabilities(caps syncstate.FilesystemCaps) apispec.VolumeSyncFilesystemCapabilities {
	return apispec.VolumeSyncFilesystemCapabilities{
		CaseSensitive:                   caps.CaseSensitive,
		UnicodeNormalizationInsensitive: caps.UnicodeNormalizationInsensitive,
		WindowsCompatiblePaths:          caps.WindowsCompatiblePaths,
	}
}

func limitOrDefault(limit int32) int32 {
	if limit <= 0 {
		return defaultChangeLimit
	}
	return limit
}

func ioReadAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	return io.ReadAll(r)
}
