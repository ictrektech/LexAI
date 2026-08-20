package officeengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	enginev1 "github.com/Tencent/WeKnora/internal/infrastructure/officeengine/gen/office/engine/v1"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	client  enginev1.OfficeEngineServiceClient
	name    string
	timeout time.Duration
	reason  string
}

func NewClients(cfg *config.Config) interfaces.DocumentEngineSet {
	if cfg == nil || cfg.OfficeEngine == nil || !cfg.OfficeEngine.Enabled {
		return interfaces.DocumentEngineSet{
			Adeu:      unavailable("adeu", "office editing is disabled"),
			OfficeCLI: unavailable("officecli", "office editing is disabled"),
		}
	}
	return interfaces.DocumentEngineSet{
		Adeu:      newClient("adeu", cfg.OfficeEngine.AdeuAddr, cfg.OfficeEngine.AdeuTimeout),
		OfficeCLI: newClient("officecli", cfg.OfficeEngine.OfficeCLIAddr, cfg.OfficeEngine.OfficeCLITimeout),
	}
}

func newClient(name, address string, timeout time.Duration) types.DocumentEngine {
	address = strings.TrimSpace(address)
	if address == "" {
		return unavailable(name, "worker address is not configured")
	}
	target := address
	if !strings.Contains(target, "://") {
		target = "dns:///" + target
	}
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(250*1024*1024),
		grpc.MaxCallSendMsgSize(75*1024*1024),
	))
	if err != nil {
		return unavailable(name, fmt.Sprintf("worker dial failed: %v", err))
	}
	return &Client{conn: conn, client: enginev1.NewOfficeEngineServiceClient(conn), name: name, timeout: timeout}
}

type unavailableClient struct {
	name   string
	reason string
}

func unavailable(name, reason string) types.DocumentEngine {
	return &unavailableClient{name: name, reason: reason}
}

func (c *unavailableClient) Capabilities(context.Context) (types.DocumentEngineCapabilities, error) {
	return types.DocumentEngineCapabilities{EngineName: c.name}, nil
}

func (c *unavailableClient) Health(context.Context) (types.DocumentEngineHealth, error) {
	return types.DocumentEngineHealth{EngineName: c.name, Status: "unavailable", Message: c.reason}, nil
}

func (c *unavailableClient) Inspect(context.Context, *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	return nil, errors.New(c.reason)
}

func (c *unavailableClient) Outline(context.Context, *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	return nil, errors.New(c.reason)
}

func (c *unavailableClient) Search(context.Context, *types.DocumentEngineRequest, string, bool, bool, int) ([]types.DocumentEngineSearchMatch, error) {
	return nil, errors.New(c.reason)
}

func (c *unavailableClient) Apply(context.Context, *types.DocumentEngineRequest, *types.EditPlan, string) (*types.DocumentEngineResult, error) {
	return nil, errors.New(c.reason)
}

func (c *unavailableClient) Validate(context.Context, *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	return nil, errors.New(c.reason)
}

func (c *unavailableClient) Render(context.Context, *types.DocumentEngineRequest, string) (*types.DocumentEngineResult, error) {
	return nil, errors.New(c.reason)
}

func (c *Client) callContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) meta(req *types.DocumentEngineRequest) *enginev1.RequestMeta {
	if req == nil {
		return &enginev1.RequestMeta{}
	}
	return &enginev1.RequestMeta{
		RequestId:   req.RequestID,
		JobId:       req.JobID,
		Format:      string(req.Format),
		InputSha256: req.SHA256,
	}
}

func (c *Client) Capabilities(ctx context.Context) (types.DocumentEngineCapabilities, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.GetCapabilities(callCtx, &enginev1.CapabilitiesRequest{})
	if err != nil {
		return types.DocumentEngineCapabilities{}, err
	}
	for _, capability := range response.Capabilities {
		if capability.Format != enginev1.DocumentFormat_DOCUMENT_FORMAT_DOCX {
			continue
		}
		return types.DocumentEngineCapabilities{
			EngineName:      response.EngineName,
			EngineVersion:   response.EngineVersion,
			ProtocolVersion: response.ProtocolVersion,
			Format:          types.DocumentEditFormatDOCX,
			Operations:      append([]string(nil), capability.Operations...),
			TrackedChanges:  capability.TrackedChanges,
			Comments:        capability.Comments,
			Rendering:       capability.Rendering,
			Validation:      capability.Validation,
		}, nil
	}
	return types.DocumentEngineCapabilities{EngineName: response.EngineName, EngineVersion: response.EngineVersion, ProtocolVersion: response.ProtocolVersion}, nil
}

func (c *Client) Health(ctx context.Context) (types.DocumentEngineHealth, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.Health(callCtx, &enginev1.HealthRequest{})
	if err != nil {
		return types.DocumentEngineHealth{EngineName: c.name, Status: "unavailable", Message: err.Error()}, err
	}
	return types.DocumentEngineHealth{
		EngineName:      response.EngineName,
		EngineVersion:   response.EngineVersion,
		ProtocolVersion: response.ProtocolVersion,
		Status:          response.Status,
		Message:         response.Message,
	}, nil
}

func (c *Client) Inspect(ctx context.Context, req *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	stream, err := c.client.InspectStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, c.documentChunk(req)); err != nil {
		return nil, err
	}
	response, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return documentResult(response)
}

func (c *Client) Outline(ctx context.Context, req *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	stream, err := c.client.OutlineStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, c.documentChunk(req)); err != nil {
		return nil, err
	}
	response, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return documentResult(response)
}

func (c *Client) Search(ctx context.Context, req *types.DocumentEngineRequest, query string, regex, caseSensitive bool, maxMatches int) ([]types.DocumentEngineSearchMatch, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	first := c.documentChunk(req)
	first.Query = query
	first.Regex = regex
	first.CaseSensitive = caseSensitive
	first.MaxMatches = int32(maxMatches)
	stream, err := c.client.SearchStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, first); err != nil {
		return nil, err
	}
	response, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	if err := searchResponseError(response); err != nil {
		return nil, err
	}
	matches := make([]types.DocumentEngineSearchMatch, 0, len(response.Matches))
	for _, match := range response.Matches {
		matches = append(matches, types.DocumentEngineSearchMatch{Part: match.Part, Start: int(match.Start), End: int(match.End), Quote: match.Quote, Context: match.Context})
	}
	return matches, nil
}

func (c *Client) Apply(ctx context.Context, req *types.DocumentEngineRequest, plan *types.EditPlan, author string) (*types.DocumentEngineResult, error) {
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	first := c.documentChunk(req)
	first.EditPlanJson = string(planJSON)
	first.Author = author
	stream, err := c.client.ApplyStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, first); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	response, err := collectStreamResponse(stream)
	if err != nil {
		return nil, err
	}
	return documentResult(response)
}

func (c *Client) Validate(ctx context.Context, req *types.DocumentEngineRequest) (*types.DocumentEngineResult, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	stream, err := c.client.ValidateStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, c.documentChunk(req)); err != nil {
		return nil, err
	}
	response, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	return documentResult(response)
}

func (c *Client) Render(ctx context.Context, req *types.DocumentEngineRequest, format string) (*types.DocumentEngineResult, error) {
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	first := c.documentChunk(req)
	first.RenderFormat = format
	stream, err := c.client.RenderStream(callCtx)
	if err != nil {
		return nil, err
	}
	if err := sendDocumentChunks(stream, first); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	response, err := collectStreamResponse(stream)
	if err != nil {
		return nil, err
	}
	return documentResult(response)
}

type documentChunkSender interface {
	Send(*enginev1.DocumentChunk) error
}

func (c *Client) documentChunk(req *types.DocumentEngineRequest) *enginev1.DocumentChunk {
	var document []byte
	if req != nil {
		document = req.Document
	}
	return &enginev1.DocumentChunk{Meta: c.meta(req), Document: document}
}

func sendDocumentChunks(sender documentChunkSender, first *enginev1.DocumentChunk) error {
	if first == nil {
		return errors.New("office engine document request is empty")
	}
	document := first.GetDocument()
	first.Document = nil
	const chunkSize = 1024 * 1024
	if len(document) == 0 {
		return sender.Send(first)
	}
	for offset := 0; offset < len(document); offset += chunkSize {
		end := offset + chunkSize
		if end > len(document) {
			end = len(document)
		}
		chunk := &enginev1.DocumentChunk{Document: document[offset:end]}
		if offset == 0 {
			chunk.Meta = first.Meta
			chunk.Query = first.Query
			chunk.Regex = first.Regex
			chunk.CaseSensitive = first.CaseSensitive
			chunk.MaxMatches = first.MaxMatches
			chunk.EditPlanJson = first.EditPlanJson
			chunk.Author = first.Author
			chunk.RenderFormat = first.RenderFormat
		}
		if err := sender.Send(chunk); err != nil {
			return err
		}
	}
	return nil
}

func collectStreamResponse(stream interface {
	Recv() (*enginev1.StreamDocumentResponse, error)
}) (*enginev1.DocumentResponse, error) {
	response := &enginev1.DocumentResponse{}
	artifacts := make(map[string]*enginev1.Artifact)
	artifactOrder := make([]string, 0)
	for {
		part, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if part == nil {
			continue
		}
		if response.EngineName == "" {
			response.EngineName = part.EngineName
			response.EngineVersion = part.EngineVersion
			response.Status = part.Status
			response.ErrorCode = part.ErrorCode
			response.ErrorMessage = part.ErrorMessage
			response.Text = part.Text
			response.OutlineJson = part.OutlineJson
			response.ValidationJson = part.ValidationJson
			response.Warnings = append(response.Warnings, part.Warnings...)
			response.OperationResults = append(response.OperationResults, part.OperationResults...)
		}
		if part.Artifact != nil {
			key := part.Artifact.Kind + "\x00" + part.Artifact.FileName
			artifact, ok := artifacts[key]
			if !ok {
				artifact = &enginev1.Artifact{
					Kind:     part.Artifact.Kind,
					FileName: part.Artifact.FileName,
					MimeType: part.Artifact.MimeType,
					Sha256:   part.Artifact.Sha256,
				}
				artifacts[key] = artifact
				artifactOrder = append(artifactOrder, key)
			}
			artifact.Content = append(artifact.Content, part.Artifact.Content...)
		}
		if part.EndOfResponse {
			break
		}
	}
	for _, key := range artifactOrder {
		response.Artifacts = append(response.Artifacts, artifacts[key])
	}
	return response, nil
}

func documentResult(response *enginev1.DocumentResponse) (*types.DocumentEngineResult, error) {
	if response == nil {
		return nil, errors.New("office engine returned an empty response")
	}
	result := &types.DocumentEngineResult{
		EngineName:     response.EngineName,
		EngineVersion:  response.EngineVersion,
		Status:         response.Status,
		Text:           response.Text,
		OutlineJSON:    response.OutlineJson,
		ValidationJSON: response.ValidationJson,
		Warnings:       append([]string(nil), response.Warnings...),
	}
	for _, operation := range response.OperationResults {
		result.OperationResults = append(result.OperationResults, types.DocumentEngineOperationResult{
			OperationID:   operation.OperationId,
			Kind:          operation.Kind,
			Status:        operation.Status,
			ActualMatches: int(operation.ActualMatches),
			EngineName:    operation.EngineName,
			Message:       operation.Message,
		})
	}
	for _, artifact := range response.Artifacts {
		result.Artifacts = append(result.Artifacts, types.DocumentEngineArtifact{
			Kind: artifact.Kind, FileName: artifact.FileName, MimeType: artifact.MimeType,
			Content: append([]byte(nil), artifact.Content...), SHA256: artifact.Sha256,
		})
	}
	if response.Status != "ok" {
		code := response.ErrorCode
		if code == "" {
			code = "worker_error"
		}
		return result, fmt.Errorf("office engine %s: %s: %s", code, response.EngineName, response.ErrorMessage)
	}
	return result, nil
}

func searchResponseError(response *enginev1.SearchResponse) error {
	if response == nil {
		return errors.New("office engine returned an empty search response")
	}
	if response.Status != "ok" {
		return fmt.Errorf("office engine %s: %s", response.ErrorCode, response.ErrorMessage)
	}
	return nil
}
